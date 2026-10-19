package main

import (
	"path/filepath"
	"testing"

	"github.com/signalscout/signalscout/internal/model"
)

func TestRanksAndAcceptanceKeepMisses(t *testing.T) {
	result := model.DiffResult{Status: "ok", BaselineSpec: model.BaselineSpec{RunIDs: make([]string, 40)}, ElapsedNS: 1000000, Differences: []model.Difference{
		{Type: "config", Name: "DB_POOL_SIZE"}, {Type: "presence", Name: "retry"}, {Type: "latency", Name: "db.pool.acquire"}, {Type: "presence", Name: "cache.miss"},
	}}
	r := findRanks(result.Differences)
	if r.Pool != 1 || r.Retry != 2 || r.Latency != 3 || r.Decoy != 4 || !windowAccepted(result, r, 16) {
		t.Fatalf("ranks %+v", r)
	}
	result.Differences[0], result.Differences[3] = result.Differences[3], result.Differences[0]
	if windowAccepted(result, findRanks(result.Differences), 16) {
		t.Fatal("decoy above cause accepted")
	}
	result.Differences = result.Differences[:2]
	if windowAccepted(result, findRanks(result.Differences), 16) {
		t.Fatal("missing causes accepted")
	}
	result.Differences = []model.Difference{{Type: "config", Name: "DB_POOL_SIZE"}, {Type: "presence", Name: "retry"}, {Type: "latency", Name: "db.pool.acquire"}, {Type: "presence", Name: "cache.miss"}}
	if windowAccepted(result, findRanks(result.Differences), 0) {
		t.Fatal("absent baseline decoy accepted")
	}
	result.ElapsedNS = 1000000000
	if windowAccepted(result, findRanks(result.Differences), 16) {
		t.Fatal("one-second miss accepted")
	}
}

func TestPortablePaths(t *testing.T) {
	got, err := portable(filepath.Join(".cache", "shop-demo", "signalscout.db"))
	if err != nil || got != ".cache/shop-demo/signalscout.db" || filepath.IsAbs(got) {
		t.Fatalf("%q %v", got, err)
	}
	if _, err := portable(filepath.Join("..", "outside.db")); err == nil {
		t.Fatal("outside repository accepted")
	}
}
