package main

import (
	"context"
	"github.com/signalscout/signalscout/internal/store"
	"testing"
	"time"
)

func TestCustomRatesVerifyUnequalPersistedCounts(t *testing.T) {
	counts := map[string]int{"span": 11, "log": 7, "metric": 5}
	m, err := combinedTrialRates(context.Background(), counts, time.Second, map[string]int{"span": 1000, "log": 1000, "metric": 1000}, 256, 1, store.Options{QueueSize: 64, IdleTimeout: time.Hour, SweepInterval: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	for k, n := range counts {
		if m.Counts[k] != int64(n) || m.CommittedInWindow[k] != n {
			t.Fatalf("%s: %+v", k, m)
		}
	}
}

func TestProfileValidation(t *testing.T) {
	for _, tc := range []struct {
		rates map[string]int
		ok    bool
	}{{map[string]int{"span": 2304, "log": 1212, "metric": 3000}, true}, {map[string]int{"span": 1, "log": 1}, false}, {map[string]int{"span": -1, "log": 1, "metric": 1}, false}} {
		if (validateProfile(ingestProfile{Targets: tc.rates}) == nil) != tc.ok {
			t.Fatal(tc)
		}
	}
}
