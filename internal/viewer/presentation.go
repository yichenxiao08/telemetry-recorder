package viewer

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/signalscout/signalscout/internal/diff"
	"github.com/signalscout/signalscout/internal/model"
)

func anchor(id string) string {
	sum := sha256.Sum256([]byte(id))
	return "e-" + hex.EncodeToString(sum[:])
}

// Query encoding also preserves ML run IDs containing slashes or dot segments.
func runURL(id string) string { return "/run?id=" + url.QueryEscape(id) }
func jsonText(value any) (string, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "", fmt.Errorf("format snapshot: %w", err)
	}
	return string(data), nil
}
func timestamp(ns int64) string {
	if ns == 0 {
		return "unknown"
	}
	return timeValue(ns)
}

// Filter/cutoff zero is the Unix epoch; telemetry zero can mean unknown.
func timeValue(ns int64) string   { return time.Unix(0, ns).UTC().Format(time.RFC3339Nano) }
func number(value float64) string { return fmt.Sprintf("%.3f", value) }

type eventView struct {
	Signal         model.Signal
	Anchor, Offset string
	X              float64
}
type spanView struct {
	Signal                           model.Signal
	Anchor, Indent, Duration, Offset string
	Level                            int
	X, Width                         float64
	Events                           []eventView
}
type evidenceLink struct{ ID, URL string }
type point struct {
	X, Y              float64
	RunID, Value, URL string
}
type plot struct {
	Points                  []point
	FailingX                float64
	Failing, Min, Max, Unit string
}
type differenceView struct {
	Difference model.Difference
	Links      []evidenceLink
	Plot       *plot
	Related    []differenceView
}
type panel struct {
	Title       string
	Result      model.DiffResult
	Differences []differenceView
	Message     string
}
type runView struct {
	Title, DiffAction, Before string
	Run                       model.Run
	Spans                     []spanView
	Orphans                   []eventView
	Panels                    []panel
}
type runRow struct {
	Run model.Run
	URL string
}
type listView struct {
	Title, Workload, Outcome, From, To, Next, First string
	Limit                                           int
	Rows                                            []runRow
}

func clamp(value, low, high float64) float64 { return math.Max(low, math.Min(high, value)) }

func timeline(detail model.RunDetail) ([]spanView, []eventView) {
	signals := append([]model.Signal(nil), detail.Signals...)
	sort.Slice(signals, func(i, j int) bool {
		if signals[i].TS == signals[j].TS {
			return signals[i].ID < signals[j].ID
		}
		return signals[i].TS < signals[j].TS
	})
	start, end := float64(detail.Run.StartedAt), float64(detail.Run.EndedAt)
	for _, s := range signals {
		if s.TS > 0 && (start == 0 || float64(s.TS) < start) {
			start = float64(s.TS)
		}
		if float64(s.End) > end {
			end = float64(s.End)
		}
	}
	width := math.Max(1, end-start)
	x := func(ts int64) float64 { return 20 + 600*clamp((float64(ts)-start)/width, 0, 1) }
	offset := func(ts int64) string {
		if ts == 0 {
			return "unknown"
		}
		return number((float64(ts)-start)/float64(time.Millisecond)) + " ms"
	}
	key := func(s model.Signal, id string) string { return s.TraceID + "\x00" + id }
	var spans []spanView
	byID := map[string]int{}
	for _, s := range signals {
		if s.Kind != "span" {
			continue
		}
		byID[key(s, s.SpanID)] = len(spans)
		spans = append(spans, spanView{Signal: s, Anchor: anchor(s.ID), Duration: number(math.Max(0, float64(s.End)-float64(s.TS))/float64(time.Millisecond)) + " ms", Offset: offset(s.TS), X: x(s.TS), Width: clamp(x(s.End)-x(s.TS), 1, 600)})
	}
	var roots []int
	children := map[int][]int{}
	for i, s := range spans {
		parent, ok := byID[key(s.Signal, s.Signal.ParentID)]
		if s.Signal.ParentID == "" || !ok || parent == i {
			roots = append(roots, i)
		} else {
			children[parent] = append(children[parent], i)
		}
	}
	var orphans []eventView
	for _, s := range signals {
		if s.Kind != "event" {
			continue
		}
		event := eventView{Signal: s, Anchor: anchor(s.ID), Offset: offset(s.TS), X: x(s.TS)}
		if parent, ok := byID[key(s, s.SpanID)]; ok {
			spans[parent].Events = append(spans[parent].Events, event)
		} else {
			orphans = append(orphans, event)
		}
	}
	type entry struct{ index, depth int }
	var ordered []spanView
	seen := map[int]bool{}
	walk := func(root int) {
		stack := []entry{{root, 0}}
		for len(stack) > 0 {
			current := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if seen[current.index] {
				continue
			}
			seen[current.index] = true
			s := spans[current.index]
			s.Level = current.depth + 1
			s.Indent = strings.Repeat("· ", min(current.depth, 32))
			ordered = append(ordered, s)
			list := children[current.index]
			for i := len(list) - 1; i >= 0; i-- {
				stack = append(stack, entry{list[i], current.depth + 1})
			}
		}
	}
	for _, root := range roots {
		walk(root)
	}
	// Missing parents and cycles remain visible; visited nodes prevent recursion
	// or duplicated anchors even when an incomplete trace is not a valid tree.
	for i := range spans {
		if !seen[i] {
			walk(i)
		}
	}
	return ordered, orphans
}

func makePlot(distribution diff.NumericDistribution) *plot {
	if len(distribution.Values) == 0 {
		return nil
	}
	low, high := distribution.Failing, distribution.Failing
	for _, value := range distribution.Values {
		low = math.Min(low, value)
		high = math.Max(high, value)
	}
	if high == low {
		low -= .5
		high += .5
	}
	position := func(value float64) float64 { return 20 + 600*(value-low)/(high-low) }
	result := &plot{FailingX: position(distribution.Failing), Failing: number(distribution.Failing), Min: number(low), Max: number(high), Unit: distribution.Unit}
	for i, value := range distribution.Values {
		result.Points = append(result.Points, point{X: position(value), Y: 35 + float64(i%4)*8, RunID: distribution.RunIDs[i], Value: number(value), URL: runURL(distribution.RunIDs[i])})
	}
	return result
}

func panelView(title string, input model.Comparison, result model.DiffResult) (panel, error) {
	p := panel{Title: title, Result: result, Message: result.Message}
	locations := map[string]string{}
	runs := append([]model.RunDetail{input.Failing}, input.Baseline...)
	for _, run := range runs {
		for _, s := range run.Signals {
			locations[s.ID] = runURL(run.Run.ID) + "#" + anchor(s.ID)
		}
	}
	location := func(id string) string {
		if target, ok := locations[id]; ok {
			return target
		}
		bestPrefix := ""
		target := ""
		for _, run := range runs {
			for _, kind := range []string{"config", "env"} {
				prefix := kind + ":" + run.Run.ID + "/"
				if strings.HasPrefix(id, prefix) && len(prefix) > len(bestPrefix) {
					section := "config"
					if kind == "env" {
						section = "environment"
					}
					bestPrefix = prefix
					target = runURL(run.Run.ID) + "#" + section
				}
			}
		}
		return target
	}
	distributions := diff.NewDistributions(input)
	var display func(model.Difference) (differenceView, error)
	display = func(d model.Difference) (differenceView, error) {
		view := differenceView{Difference: d}
		for _, id := range d.EvidenceIDs {
			view.Links = append(view.Links, evidenceLink{ID: id, URL: location(id)})
		}
		if d.Type == "latency" || d.Type == "count" {
			values, err := distributions.ForDifference(d)
			if err != nil {
				return view, err
			}
			view.Plot = makePlot(values)
		}
		for _, related := range d.Related {
			child, err := display(related)
			if err != nil {
				return view, err
			}
			view.Related = append(view.Related, child)
		}
		return view, nil
	}
	for _, d := range result.Differences {
		view, err := display(d)
		if err != nil {
			return p, err
		}
		p.Differences = append(p.Differences, view)
	}
	return p, nil
}
