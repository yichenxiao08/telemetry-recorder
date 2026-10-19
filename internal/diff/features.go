package diff

import (
	"encoding/json"
	"sort"
	"strconv"
	"time"

	"github.com/signalscout/signalscout/internal/model"
)

type feature struct {
	name    string
	path    []string
	count   int
	maximum float64
	maxID   string
	first   int64
	ids     []string
	// Event paths identify the branches annotated by an aggregated event name.
	eventPaths [][]string
}

type features struct {
	paths  map[string]feature
	items  map[string]feature
	anchor string
}

func pathKey(path []string) string {
	data, _ := json.Marshal(path) // A slice of strings is always JSON encodable.
	return string(data)
}

func spanKey(trace, id string) string { return trace + "\x00" + id }

func spanPath(signal model.Signal, spans map[string]model.Signal) []string {
	var reversed []string
	seen := map[string]bool{}
	for {
		key := spanKey(signal.TraceID, signal.SpanID)
		if seen[key] {
			reversed = append(reversed, "[cycle]")
			break
		}
		seen[key] = true
		reversed = append(reversed, signal.Name)
		if signal.ParentID == "" {
			break
		}
		parent, ok := spans[spanKey(signal.TraceID, signal.ParentID)]
		if !ok {
			reversed = append(reversed, "[missing parent:"+signal.ParentID+"]")
			break
		}
		signal = parent
	}
	for i, j := 0, len(reversed)-1; i < j; i, j = i+1, j-1 {
		reversed[i], reversed[j] = reversed[j], reversed[i]
	}
	return reversed
}

func addOccurrence(f feature, s model.Signal, path []string) feature {
	if f.count == 0 {
		f.name = s.Name
		f.path = path
		f.first = s.TS
	}
	f.count++
	f.ids = append(f.ids, s.ID)
	return f
}

func extract(detail model.RunDetail) features {
	result := features{paths: map[string]feature{}, items: map[string]feature{}}
	signals := append([]model.Signal(nil), detail.Signals...)
	sort.Slice(signals, func(i, j int) bool {
		if signals[i].TS == signals[j].TS {
			return signals[i].ID < signals[j].ID
		}
		return signals[i].TS < signals[j].TS
	})
	spans := map[string]model.Signal{}
	for _, s := range signals {
		if s.Kind == "span" {
			spans[spanKey(s.TraceID, s.SpanID)] = s
		}
	}
	for _, s := range signals {
		if s.Kind != "span" && s.Kind != "event" {
			continue
		}
		var path []string
		if s.Kind == "span" {
			path = spanPath(s, spans)
			if result.anchor == "" {
				result.anchor = s.ID
			}
			key := pathKey(path)
			f := addOccurrence(result.paths[key], s, path)
			if s.End >= s.TS {
				duration := float64(s.End-s.TS) / float64(time.Millisecond)
				if f.maxID == "" || duration > f.maximum {
					f.maximum = duration
					f.maxID = s.ID
				}
			}
			result.paths[key] = f
		} else if parent, ok := spans[spanKey(s.TraceID, s.SpanID)]; ok {
			path = spanPath(parent, spans)
		} else {
			path = []string{"[missing event parent:" + s.SpanID + "]"}
		}
		key := s.Kind + ":" + s.Name
		f := addOccurrence(result.items[key], s, nil)
		if s.Kind == "event" {
			f.eventPaths = append(f.eventPaths, path)
		}
		result.items[key] = f
	}
	return result
}

// sortedKeys makes extraction/ranking independent of map iteration order.
func sortedKeys(values map[string]feature) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func featureLabel(f feature) string {
	if len(f.path) > 0 {
		return pathKey(f.path)
	}
	return strconv.Quote(f.name)
}
