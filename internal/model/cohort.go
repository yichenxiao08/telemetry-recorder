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
	Name      string  `json:"name"`
	Selector  string  `json:"selector"`
	Subtract  string  `json:"subtract,omitempty"`
	Unit      string  `json:"unit"`
	Epsilon   float64 `json:"epsilon"`
	Floor     float64 `json:"floor"`
	Role      string  `json:"role,omitempty"`
	Rationale string  `json:"rationale,omitempty"`
}
type CategoryFeature struct {
	Name      string `json:"name"`
	Selector  string `json:"selector"`
	Role      string `json:"role,omitempty"`
	Rationale string `json:"rationale,omitempty"`
}
type VersionInvariant struct {
	Declared string  `json:"declared"`
	Current  string  `json:"current"`
	Used     string  `json:"used"`
	Delta    float64 `json:"delta"`
}
type AnalysisProfile struct {
	SelectorSyntax  string            `json:"selector_syntax,omitempty"`
	Relations       *RelationPolicy   `json:"relations,omitempty"`
	ID              string            `json:"id"`
	Unit            string            `json:"unit"`
	Workload        string            `json:"workload"`
	Step            string            `json:"step"`
	EnvelopeProfile string            `json:"envelope_profile"`
	EnvelopeVersion string            `json:"envelope_version"`
	Numeric         []NumericFeature  `json:"numeric"`
	Categories      []CategoryFeature `json:"categories"`
	Invariant       VersionInvariant  `json:"invariant"`
	RoleAuthor      string            `json:"role_author,omitempty"`
	ReferenceTail   bool              `json:"reference_tail,omitempty"`
}
type CohortSelection struct {
	RecordLimit int    `json:"record_limit,omitempty"`
	Relation    string `json:"relation,omitempty"`
	From        int64  `json:"from"`
	To          int64  `json:"to"`
	Limit       int    `json:"limit"`
	Dimension   string `json:"dimension,omitempty"`
	Value       string `json:"value,omitempty"`
	Predicate   string `json:"predicate,omitempty"`
	Equals      string `json:"equals,omitempty"`
	// IncludePartial keeps runs with confirmed lost selected items (excluded by default).
	IncludePartial bool `json:"include_partial,omitempty"`
	// RequireComplete rejects the cohort unless every candidate run is demonstrably assessed.
	RequireComplete bool `json:"require_complete,omitempty"`
}

// CompletenessCounts are run counts by capture-completeness status. Items are
// summed over the assessed runs' manifests; they are never silently dropped.
type CompletenessCounts struct {
	Complete           int   `json:"complete"`
	Partial            int   `json:"partial"`
	Unknown            int   `json:"unknown"`
	Absent             int   `json:"absent"`
	Unassessed         int   `json:"unassessed"`
	PartialWithUnknown int   `json:"partial_with_unknown"`
	LostItems          int64 `json:"lost_items"`
	UnknownItems       int64 `json:"unknown_items"`
	DetailLossItems    int64 `json:"detail_loss_items"`
}

// CohortCompleteness is pinned at creation. It is nil (and omitted from the
// JSON) when no candidate run has manifest or capture-scope evidence, so
// cohorts built from legacy runs serialize exactly as before.
type CohortCompleteness struct {
	Version         int                `json:"version"`
	Mode            string             `json:"mode"`
	Candidates      CompletenessCounts `json:"candidates"`
	Selected        CompletenessCounts `json:"selected"`
	RunStatus       map[string]string  `json:"run_status"`
	PopulationKnown bool               `json:"population_known"`
	Scopes          []string           `json:"scopes,omitempty"`
}
type Cohort struct {
	Relations    *RelationReport     `json:"relations,omitempty"`
	ID           string              `json:"id"`
	Profile      AnalysisProfile     `json:"profile"`
	Selection    CohortSelection     `json:"selection"`
	CreatedAt    int64               `json:"created_at"`
	RunIDs       []string            `json:"run_ids"`
	LabelIDs     []string            `json:"label_ids"`
	DecisionIDs  map[string]string   `json:"decision_ids"`
	Sessions     map[string]string   `json:"sessions"`
	Excluded     map[string]int      `json:"excluded"`
	Completeness *CohortCompleteness `json:"completeness,omitempty"`
}

type DecisionObservation struct {
	Sampled    bool           `json:"sampled,omitempty"`
	RunID      string         `json:"run_id"`
	EvidenceID string         `json:"evidence_id"`
	TS         int64          `json:"ts_ns"`
	Session    string         `json:"session"`
	Values     map[string]any `json:"values"`
}
