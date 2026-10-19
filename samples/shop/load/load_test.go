package load

import (
	"context"
	"encoding/json"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/signalscout/signalscout/samples/shop/workload"
)

func TestSeededCartsAndBulkPolicy(t *testing.T) {
	a := rand.New(rand.NewSource(42))
	b := rand.New(rand.NewSource(42))
	for index := 0; index < 20; index++ {
		first := cart(a, "x", 0, true)
		second := cart(b, "x", 0, true)
		if !reflect.DeepEqual(first, second) {
			t.Fatal("unseeded cart variation")
		}
		if err := first.Validate(); err != nil {
			t.Fatal(err)
		}
		if workload.CacheHit(first) {
			t.Fatal("burst victim bypasses the database")
		}
	}
}

func TestReportsActualResponsesAndBoundsConcurrency(t *testing.T) {
	var active, peak atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`{"in_use":5}`))
			return
		}
		now := active.Add(1)
		defer active.Add(-1)
		for old := peak.Load(); now > old; old = peak.Load() {
			if peak.CompareAndSwap(old, now) {
				break
			}
		}
		var c workload.Cart
		if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
			t.Error(err)
			return
		}
		time.Sleep(time.Millisecond)
		w.Header().Set("X-Trace-ID", "00000000000000000000000000000001")
		if strings.HasSuffix(c.ID, "0005") || strings.HasSuffix(c.ID, "0006") || strings.HasSuffix(c.ID, "0007") {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	}))
	defer server.Close()
	result, err := Run(context.Background(), Options{URL: server.URL, InventoryHealthURL: server.URL, Count: 100, Concurrency: 8, Seed: 1, Phase: "test", Bursts: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.Requests != 100 || result.Failing != 3 || result.Passing != 97 || peak.Load() > 8 {
		t.Fatalf("bad load accounting: %+v peak %d", result, peak.Load())
	}
	for _, record := range result.Records {
		if record.TraceID == "" {
			t.Fatal("trace ID was discarded")
		}
	}
}
