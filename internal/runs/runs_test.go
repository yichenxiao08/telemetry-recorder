package runs

import (
	"math"
	"testing"
	"time"

	"github.com/signalscout/signalscout/internal/model"
)

func TestOutcome(t *testing.T) {
	cases := []struct {
		name string
		in   OutcomeInput
		want string
	}{
		{"web pass", OutcomeInput{RootSeen: true}, "pass"},
		{"root error", OutcomeInput{RootSeen: true, RootError: true}, "fail"},
		{"http error", OutcomeInput{RootSeen: true, HTTPStatus: 503}, "fail"},
		{"slow", OutcomeInput{RootSeen: true, DurationMS: 11, ThresholdMS: 10}, "degraded"},
		{"boundary", OutcomeInput{RootSeen: true, DurationMS: 10, ThresholdMS: 10}, "pass"},
		{"missing root", OutcomeInput{}, "degraded"},
		{"ml pass", OutcomeInput{Domain: "ml"}, "pass"},
		{"ml exception", OutcomeInput{Domain: "ml", Failed: true}, "fail"},
		{"ml NaN", OutcomeInput{Domain: "ml", Loss: math.NaN()}, "fail"},
		{"ml infinity", OutcomeInput{Domain: "ml", Loss: math.Inf(1)}, "fail"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Outcome(tc.in); got != tc.want {
				t.Fatalf("got %s, want %s", got, tc.want)
			}
		})
	}
}

func TestGroupingAndLateData(t *testing.T) {
	now := time.Unix(10, 0)
	root := model.Signal{ID: "span:root", Kind: "span", TraceID: "trace", Name: "request", TS: 1, End: 20, Attrs: map[string]any{"http.route": "/checkout", "http.method": "POST"}, Resource: model.Resource{Attrs: map[string]any{"service.name": "shop"}}}
	child := root
	child.ID = "span:child"
	child.ParentID = "root"
	child.TS = 2
	child.End = 10
	state := Observe(State{}, child, now, 0)
	state = Observe(state, root, now, 0)
	if state.Run.ID != "trace" || state.Run.Workload != "POST /checkout" || state.Run.Outcome != "pass" {
		t.Fatalf("wrong run: %+v", state.Run)
	}
	if Idle(state, now.Add(time.Second-time.Nanosecond), time.Second) {
		t.Fatal("closed early")
	}
	if !Idle(state, now.Add(time.Second), time.Second) {
		t.Fatal("not idle at boundary")
	}
	state.Run.Closed = true
	root.Status = "error"
	state = Observe(state, root, now.Add(2*time.Second), 0)
	if !state.Run.Closed || state.Run.Outcome != "fail" {
		t.Fatalf("late evidence: %+v", state)
	}
	ml := root
	ml.Resource.Attrs = map[string]any{"signalscout.run.id": "training", "signalscout.workload": "train/small"}
	ml.Name = "ml.loss"
	ml.Kind = "metric"
	ml.ValueText = "NaN"
	state = Observe(State{}, ml, now, 0)
	if state.Run.ID != "training" || state.Run.Domain != "ml" || state.Run.Outcome != "fail" {
		t.Fatalf("ML grouping: %+v", state)
	}
}

func TestSpanResourceConfigIsFlatWithoutMetrics(t *testing.T) {
	span := model.Signal{ID: "span:root", Kind: "span", TraceID: "run", Name: "work", TS: 1, End: 2, Resource: model.Resource{Attrs: map[string]any{"signalscout.config.DB_POOL_SIZE": 5, "vcs.revision": "commit"}}}
	got := Observe(State{}, span, time.Now(), 0)
	if got.Run.Config["DB_POOL_SIZE"] != 5 || got.Run.Env["vcs.revision"] != "commit" {
		t.Fatalf("snapshot %+v", got.Run)
	}
	if _, nested := got.Run.Config["signalscout.config.DB_POOL_SIZE"]; nested {
		t.Fatal("config prefix was not removed")
	}
}

func TestContextAttachment(t *testing.T) {
	run := model.Run{StartedAt: 10, EndedAt: 20}
	for _, tc := range []struct {
		name string
		s    model.Signal
		want bool
	}{
		{"ambient overlapping", model.Signal{Kind: "log", TS: 10, Resource: model.Resource{ID: "r"}}, true},
		{"metric end boundary", model.Signal{Kind: "metric", TS: 20, Resource: model.Resource{ID: "r"}}, true},
		{"outside", model.Signal{Kind: "log", TS: 21, Resource: model.Resource{ID: "r"}}, false},
		{"other resource", model.Signal{Kind: "log", TS: 15, Resource: model.Resource{ID: "other"}}, false},
		{"owned by another", model.Signal{Kind: "log", RunID: "other", TS: 15, Resource: model.Resource{ID: "r"}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := ContextMatches(run, map[string]bool{"r": true}, tc.s); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestTemplates(t *testing.T) {
	n := NewNormalizer()
	cases := map[string]string{`request 42 failed for "alice"`: `request {} failed for {}`, `peer 10.0.0.1 retry 3`: `peer {} retry {}`, `job 123e4567-e89b-12d3-a456-426614174000 done`: `job {} done`, `HTTP timeout`: `HTTP timeout`, `order_id=71 status=503`: `order_id={} status={}`}
	for input, want := range cases {
		got := n.Template(input)
		if got != want {
			t.Errorf("%q: got %q, want %q", input, got, want)
		}
		if n.Template(got) != got {
			t.Errorf("not idempotent: %q", input)
		}
	}
}

func TestRootSelectionIgnoresArrivalOrderAndChildErrors(t *testing.T) {
	root := model.Signal{ID: "span:earlier", Kind: "span", TraceID: "trace", Name: "earlier", TS: 1, End: 10, Resource: model.Resource{Attrs: map[string]any{}}, Attrs: map[string]any{}}
	later := root
	later.ID = "span:later"
	later.Name = "later"
	later.TS = 2
	later.Status = "error"
	for _, signals := range [][]model.Signal{{later, root}, {root, later}} {
		state := State{}
		for _, signal := range signals {
			state = Observe(state, signal, time.Now(), 0)
		}
		if state.Run.Workload != "earlier" || state.Run.Outcome != "pass" || state.Run.FailureMarker != "" {
			t.Fatalf("arrival order changed selected root: %+v", state)
		}
		child := later
		child.ParentID = "earlier"
		child.ID = "span:child"
		state = Observe(state, child, time.Now(), 0)
		if state.Run.Outcome != "pass" {
			t.Fatal("child error failed a successful web request")
		}
	}
}

func TestUnknownEventTimeDoesNotExpandRunToEpoch(t *testing.T) {
	root := model.Signal{ID: "s", Kind: "span", TraceID: "trace", TS: 100, End: 200, Resource: model.Resource{Attrs: map[string]any{}}}
	event := root
	event.ID = "event"
	event.Kind = "event"
	event.TS = 0
	event.End = 0
	for _, signals := range [][]model.Signal{{root, event}, {event, root}} {
		state := State{}
		for _, s := range signals {
			state = Observe(state, s, time.Now(), 0)
		}
		if state.Run.StartedAt != 100 {
			t.Fatalf("unknown time changed interval: %+v", state)
		}
	}
}
