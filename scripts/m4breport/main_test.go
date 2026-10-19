package main

import (
	"github.com/signalscout/signalscout/internal/model"
	"testing"
)

func TestRanksNeverInventMissingEvidence(t *testing.T) {
	ds := []model.Difference{{Type: "context_metric", Name: "db.pool.capacity"}}
	if rank(ds, "context_metric", "db.pool.in_use") != 0 || rankText(0) != "absent" || rank(ds, "context_metric", "db.pool.capacity") != 1 {
		t.Fatal("invented or shifted rank")
	}
	if maximum([]float64{2, 4, 3}) != 4 {
		t.Fatal("query maximum")
	}
}
