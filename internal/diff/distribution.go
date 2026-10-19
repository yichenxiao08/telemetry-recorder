package diff

import (
	"fmt"
	"github.com/signalscout/signalscout/internal/model"
	"strings"
)

type NumericDistribution struct {
	Values  []float64
	RunIDs  []string
	Failing float64
	Unit    string
}

// Distribution uses the exact selected snapshot and feature aggregation used by
// Compute. Latency excludes absent paths; counts include absent runs as zeros.
// It is display-only and does not change scoring or persisted result identity.
func Distribution(input model.Comparison, d model.Difference) (NumericDistribution, error) {
	result := NumericDistribution{Failing: d.Value, Unit: "occurrences"}
	if d.Type != "latency" && d.Type != "count" {
		return result, fmt.Errorf("difference is not numeric")
	}
	if d.Type == "latency" {
		result.Unit = "ms"
	}
	for _, run := range input.Baseline {
		f := extract(run)
		observed := f.items[d.Feature]
		if strings.HasPrefix(d.Feature, "path:") {
			observed = f.paths[strings.TrimPrefix(d.Feature, "path:")]
		}
		value := float64(observed.count)
		if d.Type == "latency" {
			if observed.maxID == "" {
				continue
			}
			value = observed.maximum
		}
		result.Values = append(result.Values, value)
		result.RunIDs = append(result.RunIDs, run.Run.ID)
	}
	return result, nil
}
