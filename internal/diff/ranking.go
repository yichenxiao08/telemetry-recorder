package diff

import (
	"sort"

	"github.com/signalscout/signalscout/internal/model"
)

func typePriority(kind string) int {
	switch kind {
	case "latency":
		return 0
	case "presence":
		return 1
	case "count":
		return 2
	case "config":
		return 3
	case "env":
		return 4
	case "context_metric":
		return 5
	case "context_log":
		return 6
	default:
		return 7
	}
}

func rank(differences []model.Difference) {
	sort.SliceStable(differences, func(i, j int) bool {
		a, b := differences[i], differences[j]
		if a.Score != b.Score {
			return a.Score > b.Score
		}
		if typePriority(a.Type) != typePriority(b.Type) {
			return typePriority(a.Type) < typePriority(b.Type)
		}
		if a.Timestamp != b.Timestamp {
			return a.Timestamp < b.Timestamp
		}
		return a.Feature < b.Feature
	})
}

func pathPrefix(parent, child []string) bool {
	if len(parent) > len(child) {
		return false
	}
	for i := range parent {
		if parent[i] != child[i] {
			return false
		}
	}
	return true
}

func collapse(differences []model.Difference, failing features) []model.Difference {
	rank(differences)
	// A presence/count pair measures the same item twice. Preserve the weaker
	// observation as Related rather than occupying another rank.
	var grouped []model.Difference
	for _, d := range differences {
		merged := false
		if d.Type == "presence" || d.Type == "count" {
			for i := range grouped {
				g := &grouped[i]
				if g.Feature == d.Feature && (g.Type == "presence" || g.Type == "count") {
					g.Related = append(g.Related, d)
					merged = true
					break
				}
			}
		}
		if !merged {
			grouped = append(grouped, d)
		}
	}
	targets := make([]int, len(grouped))
	for i := range targets {
		targets[i] = -1
	}
	for i, d := range grouped {
		if d.Type != "latency" {
			continue
		}
		for j, child := range grouped {
			if child.Type != "latency" || len(child.Path) <= len(d.Path) || !pathPrefix(d.Path, child.Path) {
				continue
			}
			previous := targets[i]
			if previous < 0 || len(child.Path) > len(grouped[previous].Path) || len(child.Path) == len(grouped[previous].Path) && child.Feature < grouped[previous].Feature {
				targets[i] = j
			}
		}
	}
	for i, d := range grouped {
		if d.Feature != "event:exception" {
			continue
		}
		annotations := failing.items[d.Feature].eventPaths
		if len(annotations) == 0 {
			continue
		}
		for j, slow := range grouped {
			if slow.Type != "latency" || targets[j] >= 0 {
				continue
			}
			matching := true
			for _, path := range annotations {
				if !pathPrefix(path, slow.Path) {
					matching = false
					break
				}
			}
			if matching && (targets[i] < 0 || len(slow.Path) > len(grouped[targets[i]].Path)) {
				targets[i] = j
			}
		}
	}
	for i, target := range targets {
		if target >= 0 {
			grouped[target].Related = append(grouped[target].Related, grouped[i])
		}
	}
	result := make([]model.Difference, 0, len(grouped))
	for i, d := range grouped {
		if targets[i] < 0 {
			rank(d.Related)
			result = append(result, d)
		}
	}
	rank(result)
	return result
}
