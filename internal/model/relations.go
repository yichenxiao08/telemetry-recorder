package model

// Expression is a closed JSON tree, never source code. Each node has exactly
// one form: field, number, boolean, or an operator with bounded args.
type Expression struct {
	Op      string       `json:"op"`
	Field   string       `json:"field,omitempty"`
	Number  *float64     `json:"number,omitempty"`
	Boolean *bool        `json:"boolean,omitempty"`
	Args    []Expression `json:"args,omitempty"`
}
type RelationRule struct {
	ID        string      `json:"id"`
	Unit      string      `json:"unit"`
	Tolerance *float64    `json:"tolerance"`
	When      *Expression `json:"when,omitempty"`
	Assert    Expression  `json:"assert"`
}
type RelationPolicy struct {
	Version int            `json:"version"`
	Rules   []RelationRule `json:"rules"`
}

type RelationCounts struct {
	Total             int            `json:"recorded_decisions"`
	Satisfied         int            `json:"satisfied"`
	Violated          int            `json:"violated"`
	Unknown           int            `json:"unknown"`
	NotApplicable     int            `json:"not_applicable"`
	SampledRecords    int            `json:"sampled_records"`
	ViolationEvidence []string       `json:"violation_evidence"`
	ViolationURLs     []string       `json:"violation_urls"`
	EvidenceOmitted   int            `json:"evidence_omitted"`
	LastViolation     string         `json:"last_violation,omitempty"`
	UnknownReasons    map[string]int `json:"unknown_reasons"`
}
type RelationReport struct {
	Version             int                                   `json:"version"`
	Tier                string                                `json:"evidence_tier"`
	Step                string                                `json:"step"`
	Profile             AnalysisProfile                       `json:"profile"`
	From                int64                                 `json:"from"`
	To                  int64                                 `json:"to"`
	RunLimit            int                                   `json:"run_limit"`
	RecordLimit         int                                   `json:"record_limit"`
	CandidateRuns       int                                   `json:"candidate_runs"`
	RunsWithoutDecision []string                              `json:"runs_without_supported_decision"`
	Totals              map[string]*RelationCounts            `json:"step_counts"`
	Runs                map[string]map[string]*RelationCounts `json:"run_counts"`
	SamplingNote        string                                `json:"sampling_note"`
}
