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

type Distributions struct {
	baseline []features
	ids      []string
}

// NewDistributions extracts each baseline once for all charts in a diff panel.
func NewDistributions(input model.Comparison) Distributions {
	result := Distributions{}
	for _, run := range input.Baseline {
		result.baseline = append(result.baseline, extract(run))
		result.ids = append(result.ids, run.Run.ID)
	}
	return result
}

// Distribution uses the exact selected snapshot and feature aggregation used by
// Compute. Latency excludes absent paths; counts include absent runs as zeros.
// It is display-only and does not change scoring or persisted result identity.
func Distribution(input model.Comparison, d model.Difference) (NumericDistribution, error) {
	return NewDistributions(input).ForDifference(d)
}

func (samples Distributions) ForDifference(d model.Difference) (NumericDistribution, error) {
	result := NumericDistribution{Failing: d.Value, Unit: "occurrences"}
	if d.Type != "latency" && d.Type != "count" {
		return result, fmt.Errorf("difference is not numeric")
	}
	if d.Type == "latency" {
		result.Unit = "ms"
	}
	for index, f := range samples.baseline {
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
		result.RunIDs = append(result.RunIDs, samples.ids[index])
	}
	return result, nil
}
