package telemetry

import "testing"

func TestRequiredBounds(t *testing.T) {
	f := Filter{Kind: "log", Service: "shop", From: 1, To: 10, Limit: 2}
	if f.Validate() != nil {
		t.Fatal("valid")
	}
	for _, bad := range []Filter{{Kind: "log", Service: "shop", Limit: 2}, {Kind: "span", From: 1, To: 2, Limit: 2}, {Kind: "metric", TraceID: "t", From: 1, To: 2, Limit: 2}, {Kind: "log", Service: "shop", From: 1, To: 2, Limit: 1001}} {
		if bad.Validate() == nil {
			t.Fatal(bad)
		}
	}
}
