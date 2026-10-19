package viewer

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/signalscout/signalscout/internal/model"
	"github.com/signalscout/signalscout/internal/telemetry"
)

type contextRow struct {
	Signal           model.Signal
	Anchor, TraceURL string
	X                float64
}
type seriesChart struct {
	Name, Resource, Line, SelectURL string
	From, To, Low, High             string
	FailureX                        float64
	HasFailure                      bool
	Points                          []point
}
type exploreView struct {
	Title, Mode, Next, TraceLogsURL string
	Query                           url.Values
	Rows                            []contextRow
	Charts                          []seriesChart
	Templates                       []telemetry.TemplateCount
	Services                        []telemetry.ServiceStats
	TraceSpans                      []spanView
	Queried, Trace                  bool
}

func explorerFilter(q url.Values, kind string) (telemetry.Filter, error) {
	f := telemetry.Filter{Kind: kind, Service: q.Get("service"), TraceID: q.Get("trace"), Name: q.Get("name"), Text: q.Get("text"), Template: q.Get("template"), Status: q.Get("status"), Attribute: q.Get("attribute"), AttributeValue: q.Get("value")}
	f.NoService = q.Get("no_service") == "1"
	var err error
	if f.From, err = parseTime(q.Get("from")); err != nil {
		return f, err
	}
	if f.To, err = parseTime(q.Get("to")); err != nil {
		return f, err
	}
	if f.Limit, err = strconv.Atoi(q.Get("limit")); err != nil {
		return f, fmt.Errorf("explicit numeric limit required")
	}
	for _, x := range []struct {
		name string
		dest **int
	}{{"severity_min", &f.SeverityMin}, {"severity_max", &f.SeverityMax}} {
		if text := q.Get(x.name); text != "" {
			n, e := strconv.Atoi(text)
			if e != nil {
				return f, fmt.Errorf("invalid severity")
			}
			*x.dest = &n
		}
	}
	for _, x := range []struct {
		name string
		dest **float64
	}{{"duration_min", &f.DurationMin}, {"duration_max", &f.DurationMax}} {
		if text := q.Get(x.name); text != "" {
			n, e := strconv.ParseFloat(text, 64)
			if e != nil {
				return f, fmt.Errorf("invalid duration")
			}
			*x.dest = &n
		}
	}
	if text := q.Get("cursor"); text != "" {
		data, e := base64.RawURLEncoding.DecodeString(text)
		if e != nil {
			return f, fmt.Errorf("invalid cursor")
		}
		var c telemetry.Cursor
		if json.Unmarshal(data, &c) != nil || c.ID == "" || c.TS < f.From || c.TS >= f.To {
			return f, fmt.Errorf("invalid cursor")
		}
		f.After = &c
	}
	return f, f.Validate()
}
func (v *viewer) explore(w http.ResponseWriter, r *http.Request) { v.explorer(w, r, false) }
func (v *viewer) trace(w http.ResponseWriter, r *http.Request)   { v.explorer(w, r, true) }
func (v *viewer) explorer(w http.ResponseWriter, r *http.Request, trace bool) {
	q := r.URL.Query()
	mode := q.Get("mode")
	if mode == "" {
		mode = "spans"
	}
	if trace {
		mode = "spans"
	}
	kind := "span"
	switch mode {
	case "spans", "services":
	case "logs", "templates":
		kind = "log"
	case "metrics":
		kind = "metric"
	default:
		v.problem(w, 400, "Unknown explorer view")
		return
	}
	view := exploreView{Title: "Telemetry explorer", Mode: mode, Query: q, Trace: trace}
	if !trace && len(q) == 0 {
		v.render(w, 200, "explorer.html", view)
		return
	}
	if trace && q.Get("trace") == "" {
		v.problem(w, 400, "Trace ID is required")
		return
	}
	f, err := explorerFilter(q, kind)
	if err != nil {
		v.problem(w, 400, err.Error())
		return
	}
	view.Queried = true
	if (mode == "templates" || mode == "services") && f.After != nil || mode == "services" && f.Service == "" {
		v.problem(w, 400, "Aggregates require a whole bounded window; service overview also requires a service")
		return
	}
	switch mode {
	case "templates":
		view.Templates, err = v.reader.LogTemplates(r.Context(), f)
	case "services":
		view.Services, err = v.reader.ServiceOverview(r.Context(), f)
	default:
		var page telemetry.Page
		page, err = v.reader.SearchTelemetry(r.Context(), f)
		view.Rows = contextRows(page.Signals, f.From, f.To)
		if mode == "metrics" {
			view.Charts = seriesCharts(page.Signals, f.From, f.To, 0)
			for i := range view.Charts {
				selectQuery := r.URL.Query()
				selectQuery.Set("name", view.Charts[i].Name)
				selectQuery.Del("cursor")
				view.Charts[i].SelectURL = "/telemetry?" + selectQuery.Encode()
			}
		}
		if trace {
			view.Title = "Trace " + f.TraceID
			view.TraceSpans, _ = timeline(model.RunDetail{Signals: page.Signals})
		}
		if page.Next != nil {
			data, e := json.Marshal(page.Next)
			if e != nil {
				v.failure(w, e)
				return
			}
			q.Set("cursor", base64.RawURLEncoding.EncodeToString(data))
			view.Next = r.URL.Path + "?" + q.Encode()
			q.Del("cursor")
		}
	}
	if err != nil {
		v.failure(w, err)
		return
	}
	if trace {
		logs := r.URL.Query()
		logs.Set("mode", "logs")
		logs.Del("cursor")
		view.TraceLogsURL = "/telemetry?" + logs.Encode()
		for i := range view.Rows {
			view.Rows[i].Anchor = "record-" + view.Rows[i].Anchor
		}
	}
	v.render(w, 200, "explorer.html", view)
}
func contextRows(signals []model.Signal, from, to int64) []contextRow {
	out := []contextRow{}
	for _, s := range signals {
		link := ""
		if s.TraceID != "" {
			q := url.Values{"trace": {s.TraceID}, "from": {timeValue(from)}, "to": {timeValue(to)}, "limit": {"1000"}}
			link = "/trace?" + q.Encode()
			if s.Kind == "span" {
				// Seek inclusively to the selected span; it may be beyond the
				// first trace page. Next-page cursors remain exclusive.
				data, _ := json.Marshal(telemetry.Cursor{TS: s.TS, ID: s.ID, Inclusive: true})
				q.Set("cursor", base64.RawURLEncoding.EncodeToString(data))
				link = "/trace?" + q.Encode()
				link += "#" + anchor(s.ID)
			}
		}
		out = append(out, contextRow{Signal: s, Anchor: anchor(s.ID), TraceURL: link, X: 20 + 600*clamp((float64(s.TS)-float64(from))/math.Max(1, float64(to)-float64(from)), 0, 1)})
	}
	return out
}
func seriesCharts(signals []model.Signal, from, to, marker int64) []seriesChart {
	groups := map[string][]model.Signal{}
	for _, s := range signals {
		if s.Kind != "metric" || s.Value == nil || math.IsNaN(*s.Value) || math.IsInf(*s.Value, 0) {
			continue
		}
		attrs, _ := json.Marshal(s.Attrs)
		key := s.Name + "/" + s.Resource.ID + "/" + string(attrs)
		groups[key] = append(groups[key], s)
	}
	keys := []string{}
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := []seriesChart{}
	for _, k := range keys {
		xs := groups[k]
		low, high := *xs[0].Value, *xs[0].Value
		for _, s := range xs {
			low = math.Min(low, *s.Value)
			high = math.Max(high, *s.Value)
		}
		resource, _ := jsonText(xs[0].Resource.Attrs)
		attrs, _ := jsonText(xs[0].Attrs)
		chart := seriesChart{Name: xs[0].Name, Resource: resource + "\nPoint attributes: " + attrs, From: timeValue(from), To: timeValue(to), Low: number(low), High: number(high), HasFailure: marker > 0, FailureX: 20 + 600*clamp((float64(marker)-float64(from))/math.Max(1, float64(to)-float64(from)), 0, 1)}
		var line strings.Builder
		for _, s := range xs {
			x := 20 + 600*clamp((float64(s.TS)-float64(from))/math.Max(1, float64(to)-float64(from)), 0, 1)
			scale := math.Max(1, math.Max(math.Abs(low), math.Abs(high)))
			y := 80 - 60*(*s.Value/scale-low/scale)/math.Max(1e-12, high/scale-low/scale)
			fmt.Fprintf(&line, "%.3f,%.3f ", x, y)
			chart.Points = append(chart.Points, point{X: x, Y: y, Value: number(*s.Value), URL: "#" + anchor(s.ID)})
		}
		chart.Line = line.String()
		out = append(out, chart)
	}
	return out
}
