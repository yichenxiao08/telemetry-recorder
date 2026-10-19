package diff

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/signalscout/signalscout/internal/model"
	"github.com/signalscout/signalscout/internal/stats"
)

func Compute(input model.Comparison) (model.DiffResult, error) {
	result := model.DiffResult{FailingRunID: input.Failing.Run.ID, BaselineSpec: input.Spec, Status: "ok", Differences: []model.Difference{}}
	if !input.Failing.Run.Closed || input.Failing.Run.Outcome != "fail" {
		return result, fmt.Errorf("diff requires a closed failing run")
	}
	minimum := input.Spec.Minimum
	if minimum <= 0 {
		minimum = model.MinimumBaseline
		result.BaselineSpec.Minimum = minimum
	}
	if len(input.Baseline) < minimum {
		result.Status = "insufficient_baseline"
		result.Message = fmt.Sprintf("insufficient baseline: found %d closed passing runs; require %d", len(input.Baseline), minimum)
		return finish(result)
	}
	failing := extract(input.Failing)
	baseline := make([]features, 0, len(input.Baseline))
	seen := map[string]bool{}
	for _, b := range input.Baseline {
		if seen[b.Run.ID] || !b.Run.Closed || b.Run.Outcome != "pass" || b.Run.Workload != input.Failing.Run.Workload || b.Run.StartedAt >= input.Failing.Run.StartedAt {
			return result, fmt.Errorf("invalid baseline run %q", b.Run.ID)
		}
		seen[b.Run.ID] = true
		baseline = append(baseline, extract(b))
	}
	var differences []model.Difference
	for _, key := range sortedKeys(failing.paths) {
		f := failing.paths[key]
		var durations []float64
		for _, b := range baseline {
			if observed, ok := b.paths[key]; ok && observed.maxID != "" {
				durations = append(durations, observed.maximum)
			}
		}
		if len(durations) < minimum || f.maxID == "" {
			continue
		}
		score := numericScore(f.maximum, durations, latencyFloorMS, latencyScaleMS, false)
		if score == 0 {
			continue
		}
		middle := stats.Median(durations)
		exceed := directionalExceedances(f.maximum, middle, durations, false)
		p99 := stats.Quantile(durations, .99)
		differences = append(differences, model.Difference{Type: "latency", Feature: "path:" + key, Name: f.name, Path: f.path, Score: score, Timestamp: f.first, EvidenceIDs: []string{f.maxID}, BaselineRate: float64(exceed) / float64(len(durations)), Observations: len(durations), Value: f.maximum, Median: middle, P99: p99,
			Summary: fmt.Sprintf("%s: %.3f ms vs baseline median %.3f ms, p99 %.3f ms; %d/%d baseline runs with this path at or above the failing duration", strings.Join(f.path, " > "), f.maximum, middle, p99, exceed, len(durations))})
	}
	items := map[string]feature{}
	paths := map[string]feature{}
	for key, f := range failing.items {
		items[key] = f
	}
	for key, f := range failing.paths {
		paths[key] = f
	}
	for _, b := range baseline {
		for key, f := range b.items {
			if _, ok := items[key]; !ok {
				items[key] = f
			}
		}
		for key, f := range b.paths {
			if _, ok := paths[key]; !ok {
				paths[key] = f
			}
		}
	}
	for _, key := range sortedKeys(items) {
		f, present := failing.items[key]
		if !present {
			f = items[key]
			f.first = input.Failing.Run.StartedAt
		}
		count := 0
		var exampleIDs []string
		for _, b := range baseline {
			if observed, ok := b.items[key]; ok {
				count++
				exampleIDs = append(exampleIDs, observed.ids[0])
			}
		}
		score := presenceScore(present, count, len(baseline))
		if score > 0 {
			ids := append([]string(nil), f.ids...)
			state := "present"
			if !present {
				state = "absent"
				sort.Strings(exampleIDs)
				ids = []string{exampleIDs[0]}
				if failing.anchor != "" {
					ids = append(ids, failing.anchor)
				}
			}
			differences = append(differences, model.Difference{Type: "presence", Feature: key, Name: f.name, Score: score, Timestamp: f.first, EvidenceIDs: ids, BaselineRate: float64(count) / float64(len(baseline)), Observations: len(baseline), Summary: fmt.Sprintf("%s %s in failing run; present in %d/%d baseline runs (%.1f%%)", key, state, count, len(baseline), 100*float64(count)/float64(len(baseline)))})
		}
		if strings.HasPrefix(key, "event:") {
			differences = appendCount(differences, key, failing.items[key], f, baseline, false, failing.anchor, input.Failing.Run.StartedAt)
		}
	}
	for _, key := range sortedKeys(paths) {
		differences = appendCount(differences, "path:"+key, failing.paths[key], paths[key], baseline, true, failing.anchor, input.Failing.Run.StartedAt)
	}
	config, err := snapshotDifferences(input)
	if err != nil {
		return result, err
	}
	differences = append(differences, config...)
	differences = append(differences, contextDifferences(input)...)
	result.Differences = collapse(differences, failing)
	return finish(result)
}

func appendCount(differences []model.Difference, key string, f, example feature, baseline []features, path bool, anchor string, start int64) []model.Difference {
	values := make([]float64, 0, len(baseline))
	var baselineIDs []string
	for _, b := range baseline {
		observed := b.items[key]
		if path {
			observed = b.paths[strings.TrimPrefix(key, "path:")]
		}
		values = append(values, float64(observed.count))
		if len(observed.ids) > 0 {
			baselineIDs = append(baselineIDs, observed.ids[0])
		}
	}
	x := float64(f.count)
	score := numericScore(x, values, 1, countScale, true)
	if score == 0 {
		return differences
	}
	middle := stats.Median(values)
	exceed := directionalExceedances(x, middle, values, true)
	ids := append([]string(nil), f.ids...)
	if f.count == 0 {
		f = example
		f.first = start
		sort.Strings(baselineIDs)
		ids = append(ids, baselineIDs[0])
		if anchor != "" {
			ids = append(ids, anchor)
		}
	}
	direction := "at or above"
	if x < middle {
		direction = "at or below"
	}
	return append(differences, model.Difference{Type: "count", Feature: key, Name: f.name, Path: f.path, Score: score, Timestamp: f.first, EvidenceIDs: ids, BaselineRate: float64(exceed) / float64(len(values)), Observations: len(values), Value: x, Median: middle,
		Summary: fmt.Sprintf("%s: count %.0f vs baseline median %.1f; %d/%d baseline counts %s failing count", featureLabel(f), x, middle, exceed, len(values), direction)})
}

func finish(result model.DiffResult) (model.DiffResult, error) {
	// Timing and creation time are excluded from the content identity.
	data, err := json.Marshal(result)
	if err != nil {
		return result, fmt.Errorf("encode diff result: %w", err)
	}
	hash := sha256.Sum256(data)
	result.ID = "diff:" + hex.EncodeToString(hash[:])
	result.CreatedAt = time.Now().UTC()
	return result, nil
}
