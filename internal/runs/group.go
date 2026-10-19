package runs

import (
	"math"
	"strings"
	"time"

	"github.com/signalscout/signalscout/internal/model"
)

// State is persisted with the run, so closed runs can accept evidence after restart.
// Only the store's writer owns and changes State; readers receive value snapshots.
type State struct {
	Run        model.Run
	LastSeen   int64
	RootID     string
	RootTS     int64
	RootError  bool
	HTTPStatus int
	DurationMS float64
	Failed     bool
	FailureTS  int64
}

func Identity(s model.Signal) (id, domain string) {
	if id = text(s.Resource.Attrs["signalscout.run.id"]); id != "" {
		return id, "ml"
	}
	if s.TraceID != "" {
		return s.TraceID, "web"
	}
	return "", ""
}

func Observe(state State, s model.Signal, now time.Time, thresholdMS float64) State {
	id, domain := Identity(s)
	if state.Run.ID == "" {
		state.Run = model.Run{ID: id, Domain: domain, Workload: "unknown", StartedAt: s.TS, EndedAt: s.End, Env: map[string]any{}, Config: map[string]any{}}
	}
	state.LastSeen = now.UnixNano()
	if s.TS < state.Run.StartedAt {
		state.Run.StartedAt = s.TS
	}
	if s.End > state.Run.EndedAt {
		state.Run.EndedAt = s.End
	}
	for key, value := range s.Resource.Attrs {
		if strings.HasPrefix(key, "signalscout.config.") {
			state.Run.Config[strings.TrimPrefix(key, "signalscout.config.")] = value
		} else {
			state.Run.Env[key] = value
		}
	}
	if domain == "ml" {
		if workload := text(s.Resource.Attrs["signalscout.workload"]); workload != "" {
			state.Run.Workload = workload
		}
	} else if s.Kind == "span" && s.ParentID == "" && (state.RootID == "" || s.TS < state.RootTS || s.TS == state.RootTS && s.ID <= state.RootID) {
		state.RootID, state.RootTS = s.ID, s.TS
		state.Run.Workload = s.Name
		if route := text(s.Attrs["http.route"]); route != "" {
			method := text(s.Attrs["http.request.method"])
			if method == "" {
				method = text(s.Attrs["http.method"])
			}
			state.Run.Workload = strings.TrimSpace(method + " " + route)
		}
		state.RootError = s.Status == "error"
		status := s.Attrs["http.response.status_code"]
		if status == nil {
			status = s.Attrs["http.status_code"]
		}
		state.HTTPStatus = int(numeric(status))
		state.DurationMS = float64(s.End-s.TS) / float64(time.Millisecond)
	}
	failed := false
	if domain == "ml" {
		failed = truth(s.Resource.Attrs["signalscout.run.failed"]) || truth(s.Attrs["signalscout.run.failed"]) || truth(s.Attrs["signalscout.run.outcome"])
		failed = failed || s.Name == "exception" || s.Name == "job.failed" || s.Kind == "span" && s.Status == "error"
		if s.Name == "ml.loss" {
			failed = failed || s.ValueText == "NaN" || s.ValueText == "Infinity" || s.ValueText == "-Infinity"
		}
		if loss, ok := s.Attrs["ml.loss"]; ok {
			v := numeric(loss)
			failed = failed || math.IsNaN(v) || math.IsInf(v, 0)
		}
	} else {
		failed = s.ID == state.RootID && (state.RootError || state.HTTPStatus >= 500)
	}
	if failed {
		state.Failed = true
		if state.Run.FailureMarker == "" || s.TS < state.FailureTS || s.TS == state.FailureTS && s.ID < state.Run.FailureMarker {
			state.Run.FailureMarker = s.ID
			state.FailureTS = s.TS
		}
	}
	state.Run.Outcome = Outcome(OutcomeInput{Domain: domain, RootSeen: state.RootID != "", RootError: state.RootError, HTTPStatus: state.HTTPStatus, DurationMS: state.DurationMS, ThresholdMS: thresholdMS, Failed: state.Failed})
	return state
}

func Idle(state State, now time.Time, timeout time.Duration) bool {
	return now.UnixNano()-state.LastSeen >= int64(timeout)
}

// ContextMatches links ambient evidence to an interval without changing ownership.
func ContextMatches(run model.Run, resourceIDs map[string]bool, s model.Signal) bool {
	return s.RunID == "" && (s.Kind == "log" || s.Kind == "metric") && resourceIDs[s.Resource.ID] && s.TS >= run.StartedAt && s.TS <= run.EndedAt
}
