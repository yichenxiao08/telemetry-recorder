package diff

import (
	"github.com/signalscout/signalscout/internal/model"
	"testing"
)

func TestDistributionsMatchScoredFeatures(t *testing.T) {
	input := comparisonFixture()
	result, err := Compute(input)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range result.Differences {
		if d.Type != "latency" && d.Type != "count" {
			continue
		}
		values, err := Distribution(input, d)
		if err != nil {
			t.Fatal(err)
		}
		if len(values.Values) != d.Observations || values.Failing != d.Value || len(values.RunIDs) != len(values.Values) {
			t.Fatalf("distribution %+v for %+v", values, d)
		}
	}
	missing := model.Difference{Type: "count", Feature: "event:unknown"}
	got, err := Distribution(input, missing)
	if err != nil || len(got.Values) != 40 {
		t.Fatal("absent count must contribute zeros")
	}
	for _, v := range got.Values {
		if v != 0 {
			t.Fatal("invented occurrence")
		}
	}
}
