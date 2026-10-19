package diff

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/signalscout/signalscout/internal/model"
)

func environmentKey(key string) bool {
	if strings.HasPrefix(key, "signalscout.") || strings.HasPrefix(key, "telemetry.sdk.") {
		return false
	}
	switch key {
	case "service.name", "service.instance.id", "process.pid", "process.command", "process.command_line":
		return false
	default:
		return true
	}
}

func snapshotValue(values map[string]any, key string) (string, error) {
	value, ok := values[key]
	if !ok {
		return "<missing>", nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode snapshot key %s: %w", key, err)
	}
	return string(data), nil
}

func snapshotDifferences(input model.Comparison) ([]model.Difference, error) {
	var result []model.Difference
	commitChanged := false
	for _, kind := range []string{"env", "config"} {
		current := input.Failing.Run.Env
		if kind == "config" {
			current = input.Failing.Run.Config
		}
		keys := map[string]bool{}
		for key := range current {
			keys[key] = true
		}
		for _, b := range input.Baseline {
			values := b.Run.Env
			if kind == "config" {
				values = b.Run.Config
			}
			for key := range values {
				keys[key] = true
			}
		}
		ordered := make([]string, 0, len(keys))
		for key := range keys {
			ordered = append(ordered, key)
		}
		sort.Strings(ordered)
		for _, key := range ordered {
			if kind == "env" && !environmentKey(key) {
				continue
			}
			value, err := snapshotValue(current, key)
			if err != nil {
				return nil, err
			}
			counts := map[string]int{}
			for _, b := range input.Baseline {
				values := b.Run.Env
				if kind == "config" {
					values = b.Run.Config
				}
				v, err := snapshotValue(values, key)
				if err != nil {
					return nil, err
				}
				counts[v]++
			}
			mode := ""
			support := 0
			for candidate, count := range counts {
				if count > support || count == support && candidate < mode {
					mode = candidate
					support = count
				}
			}
			if value == mode {
				continue
			}
			if kind == "env" && key == "vcs.revision" {
				commitChanged = true
			}
			var example string
			for _, b := range input.Baseline {
				values := b.Run.Env
				if kind == "config" {
					values = b.Run.Config
				}
				v, _ := snapshotValue(values, key)
				if v == mode && (example == "" || b.Run.ID < example) {
					example = b.Run.ID
				}
			}
			result = append(result, model.Difference{Type: kind, Feature: kind + ":" + key, Name: key, Score: float64(support) / float64(len(input.Baseline)), Timestamp: input.Failing.Run.StartedAt, EvidenceIDs: []string{kind + ":" + input.Failing.Run.ID + "/" + key, kind + ":" + example + "/" + key}, BaselineRate: float64(support) / float64(len(input.Baseline)), Observations: len(input.Baseline), Summary: fmt.Sprintf("%s %s: %s vs baseline mode %s (%d/%d runs)", kind, key, value, mode, support, len(input.Baseline))})
		}
	}
	if commitChanged {
		for i := range result {
			if result[i].Type == "config" {
				result[i].Summary += "; coincides with a commit change (correlation only)"
				result[i].EvidenceIDs = append(result[i].EvidenceIDs, "env:"+input.Failing.Run.ID+"/vcs.revision")
			}
		}
	}
	return result, nil
}
