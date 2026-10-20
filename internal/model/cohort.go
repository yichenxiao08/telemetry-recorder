package model

// Annotation is an immutable assertion, separate from execution telemetry.
type Annotation struct {
	ID            string   `json:"id"`
	RunID         string   `json:"run_id"`
	OutputRef     string   `json:"output_ref,omitempty"`
	Dimension     string   `json:"dimension"`
	Value         string   `json:"value"`
	Source        string   `json:"source"`
	Reviewer      string   `json:"reviewer"`
	Synthetic     bool     `json:"synthetic"`
	RubricID      string   `json:"rubric_id"`
	RubricVersion string   `json:"rubric_version"`
	CreatedAt     int64    `json:"created_at"`
	RecordedAt    int64    `json:"recorded_at"`
	EvidenceIDs   []string `json:"evidence_ids"`
	Supersedes    string   `json:"supersedes,omitempty"`
	Retracts      string   `json:"retracts,omitempty"`
}

type NumericFeature struct {
	Name     string  `json:"name"`
	Selector string  `json:"selector"`
	Subtract string  `json:"subtract,omitempty"`
	Unit     string  `json:"unit"`
	Epsilon  float64 `json:"epsilon"`
	Floor    float64 `json:"floor"`
}
type CategoryFeature struct {
	Name     string `json:"name"`
	Selector string `json:"selector"`
}
type VersionInvariant struct {
	Declared string  `json:"declared"`
	Current  string  `json:"current"`
	Used     string  `json:"used"`
	Delta    float64 `json:"delta"`
}
type AnalysisProfile struct {
	ID              string            `json:"id"`
	Unit            string            `json:"unit"`
	Workload        string            `json:"workload"`
	Step            string            `json:"step"`
	EnvelopeProfile string            `json:"envelope_profile"`
	EnvelopeVersion string            `json:"envelope_version"`
	Numeric         []NumericFeature  `json:"numeric"`
	Categories      []CategoryFeature `json:"categories"`
	Invariant       VersionInvariant  `json:"invariant"`
}
type CohortSelection struct {
	From      int64  `json:"from"`
	To        int64  `json:"to"`
	Limit     int    `json:"limit"`
	Dimension string `json:"dimension,omitempty"`
	Value     string `json:"value,omitempty"`
	Predicate string `json:"predicate,omitempty"`
	Equals    string `json:"equals,omitempty"`
}
type Cohort struct {
	ID          string            `json:"id"`
	Profile     AnalysisProfile   `json:"profile"`
	Selection   CohortSelection   `json:"selection"`
	CreatedAt   int64             `json:"created_at"`
	RunIDs      []string          `json:"run_ids"`
	LabelIDs    []string          `json:"label_ids"`
	DecisionIDs map[string]string `json:"decision_ids"`
	Sessions    map[string]string `json:"sessions"`
	Excluded    map[string]int    `json:"excluded"`
}

type DecisionObservation struct {
	RunID      string         `json:"run_id"`
	EvidenceID string         `json:"evidence_id"`
	TS         int64          `json:"ts_ns"`
	Session    string         `json:"session"`
	Values     map[string]any `json:"values"`
}
