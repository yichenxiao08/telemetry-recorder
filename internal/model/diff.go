package model

import "time"

const DiffScoringVersion = "v1"
const MinimumBaseline = 10

type BaselineOptions struct {
	N         int
	ExplicitN bool
	Before    string
}

type BaselineSpec struct {
	ContextPolicy     string   `json:"context_policy,omitempty"`
	ContextLookbackNS int64    `json:"context_lookback_ns,omitempty"`
	Mode              string   `json:"mode"`
	N                 int      `json:"n"`
	ExplicitN         bool     `json:"explicit_n"`
	BeforeInput       string   `json:"before_input,omitempty"`
	BeforeNS          int64    `json:"before_ns"`
	Workload          string   `json:"workload"`
	Minimum           int      `json:"minimum"`
	RunIDs            []string `json:"run_ids"`
	ScoringVersion    string   `json:"scoring_version"`
}

type Comparison struct {
	Failing  RunDetail
	Baseline []RunDetail
	Spec     BaselineSpec
}

// Related differences keep their statistics and evidence after redundancy
// collapsing. They are not independently ranked a second time.
type Difference struct {
	Type         string       `json:"type"`
	Feature      string       `json:"feature"`
	Name         string       `json:"name"`
	Path         []string     `json:"path,omitempty"`
	Score        float64      `json:"score"`
	Summary      string       `json:"summary"`
	EvidenceIDs  []string     `json:"evidence_ids"`
	BaselineRate float64      `json:"baseline_rate"`
	Observations int          `json:"observations"`
	Timestamp    int64        `json:"timestamp_ns"`
	Value        float64      `json:"value,omitempty"`
	Median       float64      `json:"baseline_median,omitempty"`
	P99          float64      `json:"baseline_p99,omitempty"`
	Related      []Difference `json:"related,omitempty"`
}

type DiffResult struct {
	ID           string       `json:"diff_id"`
	FailingRunID string       `json:"failing_run_id"`
	BaselineSpec BaselineSpec `json:"baseline_spec"`
	CreatedAt    time.Time    `json:"created_at"`
	Status       string       `json:"status"`
	Message      string       `json:"message,omitempty"`
	Differences  []Difference `json:"differences"`
	ElapsedNS    int64        `json:"elapsed_ns"`
}
