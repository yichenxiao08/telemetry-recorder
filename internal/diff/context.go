package diff

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/signalscout/signalscout/internal/model"
	"github.com/signalscout/signalscout/internal/stats"
	"github.com/signalscout/signalscout/internal/telemetry"
)

type metricWindow struct {
	name   string
	values []float64
	ids    []string
	lastTS int64
	last   float64
}
type logWindow struct {
	name  string
	ids   []string
	first int64
}
type contextWindow struct {
	metrics map[string]metricWindow
	logs    map[string]logWindow
}

func compatibleContext(run model.Run, s model.Signal) bool {
	for key, value := range s.Resource.Attrs {
		if strings.HasPrefix(key, "signalscout.config.") {
			want, ok := run.Config[strings.TrimPrefix(key, "signalscout.config.")]
			if !ok || fmt.Sprint(want) != fmt.Sprint(value) {
				return false
			}
		}
	}
	return true
}

func contextFeatures(d model.RunDetail, lookback int64) contextWindow {
	out := contextWindow{metrics: map[string]metricWindow{}, logs: map[string]logWindow{}}
	signals := append([]model.Signal(nil), d.Context...)
	sort.Slice(signals, func(i, j int) bool {
		if signals[i].TS == signals[j].TS {
			return signals[i].ID < signals[j].ID
		}
		return signals[i].TS < signals[j].TS
	})
	// Reads are time-bounded; recheck here for pure Compute callers and fixtures.
	for _, s := range signals {
		if !compatibleContext(d.Run, s) || s.TS > d.Run.EndedAt {
			continue
		}
		service := fmt.Sprint(s.Resource.Attrs["service.name"])
		host := fmt.Sprint(s.Resource.Attrs["host.name"])
		if s.Kind == "metric" && s.TS >= d.Run.StartedAt-lookback && s.Value != nil && !math.IsNaN(*s.Value) && !math.IsInf(*s.Value, 0) {
			var raw struct {
				Metric struct {
					Type, Unit  string
					IsMonotonic bool `json:"is_monotonic"`
				} `json:"metric"`
			}
			if json.Unmarshal(s.Raw, &raw) != nil {
				continue
			}
			if raw.Metric.Type != "gauge" && raw.Metric.Type != "sum" {
				continue
			}
			if raw.Metric.Type == "sum" && raw.Metric.IsMonotonic {
				continue
			}
			identity, _ := json.Marshal([]any{service, host, s.Name, s.Attrs, raw.Metric.Unit, raw.Metric.Type})
			key := string(identity)
			m := out.metrics[key]
			m.name = s.Name
			m.values = append(m.values, *s.Value)
			m.ids = append(m.ids, s.ID)
			if s.TS >= m.lastTS {
				m.lastTS = s.TS
				m.last = *s.Value
			}
			out.metrics[key] = m
		}
		if s.Kind == "log" && s.TS >= d.Run.StartedAt {
			identity, _ := json.Marshal([]any{service, host, s.Template, telemetry.Severity(s)})
			key := string(identity)
			l := out.logs[key]
			l.name = s.Template
			l.ids = append(l.ids, s.ID)
			if l.first == 0 || s.TS < l.first {
				l.first = s.TS
			}
			out.logs[key] = l
		}
	}
	return out
}
func metricSummary(m metricWindow, stat string) float64 {
	if stat == "last" {
		return m.last
	}
	if stat == "max" {
		v := m.values[0]
		for _, x := range m.values {
			v = math.Max(v, x)
		}
		return v
	}
	v := 0.0
	for _, x := range m.values {
		v += x / float64(len(m.values))
	}
	return v
}

func contextDifferences(input model.Comparison) []model.Difference {
	if input.Spec.ContextPolicy == "" {
		return nil
	}
	f := contextFeatures(input.Failing, input.Spec.ContextLookbackNS)
	baseline := []contextWindow{}
	for _, b := range input.Baseline {
		baseline = append(baseline, contextFeatures(b, input.Spec.ContextLookbackNS))
	}
	minimum := max(10, input.Spec.Minimum)
	out := []model.Difference{}
	metricKeys := []string{}
	for key := range f.metrics {
		metricKeys = append(metricKeys, key)
	}
	sort.Strings(metricKeys)
	for _, key := range metricKeys {
		m := f.metrics[key]
		if len(m.values) < 2 {
			continue
		}
		var best *model.Difference
		for _, stat := range []string{"max", "mean", "last"} {
			values := []float64{}
			for _, b := range baseline {
				if v, ok := b.metrics[key]; ok && len(v.values) > 0 {
					values = append(values, metricSummary(v, stat))
				}
			}
			if len(values) < minimum {
				continue
			}
			x := metricSummary(m, stat)
			mid := stats.Median(values)
			floor := .01 * math.Max(math.Abs(mid), 1)
			score := numericScore(x, values, floor, floor, true)
			if score <= 0 {
				continue
			}
			rate := float64(directionalExceedances(x, mid, values, true)) / float64(len(values))
			d := model.Difference{Type: "context_metric", Feature: "context-metric:" + key + ":" + stat, Name: m.name, Score: score, Timestamp: m.lastTS, EvidenceIDs: append([]string(nil), m.ids...), BaselineRate: rate, Observations: len(values), Value: x, Median: mid, P99: stats.Quantile(values, .99), Summary: fmt.Sprintf("%s context %s %.4g vs baseline median %.4g (%d retrospective windows, raw directional exceedance %.3f); config-compatible, no future lookahead; windows may overlap", m.name, stat, x, mid, len(values), rate)}
			if best == nil {
				best = &d
			} else if d.Score > best.Score {
				d.Related = append(d.Related, *best)
				best = &d
			} else {
				best.Related = append(best.Related, d)
			}
		}
		if best != nil {
			out = append(out, *best)
		}
	}
	keys := map[string]logWindow{}
	for k, v := range f.logs {
		keys[k] = v
	}
	for _, b := range baseline {
		for k, v := range b.logs {
			keys[k] = v
		}
	}
	logKeys := []string{}
	for key := range keys {
		logKeys = append(logKeys, key)
	}
	sort.Strings(logKeys)
	for _, key := range logKeys {
		l := f.logs[key]
		count := 0
		values := []float64{}
		baseIDs := []string{}
		for _, b := range baseline {
			v := b.logs[key]
			values = append(values, float64(len(v.ids)))
			if len(v.ids) > 0 {
				count++
				baseIDs = append(baseIDs, v.ids[0])
			}
		}
		present := len(l.ids) > 0
		presence := presenceScore(present, count, len(baseline))
		x := float64(len(l.ids))
		numeric := numericScore(x, values, 1, 1, true)
		score := math.Max(presence, numeric)
		if score <= 0 {
			continue
		}
		ids := append([]string(nil), l.ids...)
		if !present {
			sort.Strings(baseIDs)
			ids = baseIDs[:1]
			l = keys[key]
			l.first = input.Failing.Run.StartedAt
		}
		method := "presence"
		rate := float64(count) / float64(len(baseline))
		if numeric > presence {
			method = "count"
			rate = float64(directionalExceedances(x, stats.Median(values), values, true)) / float64(len(values))
		}
		out = append(out, model.Difference{Type: "context_log", Feature: "context-log:" + key, Name: l.name, Score: score, Timestamp: l.first, EvidenceIDs: ids, BaselineRate: rate, Observations: len(baseline), Value: x, Median: stats.Median(values), Summary: fmt.Sprintf("Context template %q: %s difference; failing count %.0f, baseline median %.1f, present in %d/%d actual run windows; shared context does not establish causality", l.name, method, x, stats.Median(values), count, len(baseline))})
	}
	return out
}
