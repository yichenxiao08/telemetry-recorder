package model

import "encoding/json"

// Resource identifies the complete OTLP resource, including its schema URL.
type Resource struct {
	ID    string          `json:"resource_id"`
	Attrs map[string]any  `json:"attrs,omitempty"`
	Raw   json.RawMessage `json:"raw,omitempty"`
}

// Signal retains the original OTLP record as JSON as well as searchable fields.
// Non-finite numbers are represented in ValueText, never forced into JSON numbers.
type Signal struct {
	ID        string          `json:"id"`
	Kind      string          `json:"kind"`
	RunID     string          `json:"run_id,omitempty"`
	TraceID   string          `json:"trace_id,omitempty"`
	SpanID    string          `json:"span_id,omitempty"`
	ParentID  string          `json:"parent_span_id,omitempty"`
	Name      string          `json:"name"`
	TS        int64           `json:"ts_ns"`
	End       int64           `json:"end_ns"`
	Status    string          `json:"status,omitempty"`
	Attrs     map[string]any  `json:"attrs"`
	Resource  Resource        `json:"resource"`
	Scope     json.RawMessage `json:"scope"`
	Raw       json.RawMessage `json:"raw"`
	Body      string          `json:"body,omitempty"`
	Template  string          `json:"template,omitempty"`
	Value     *float64        `json:"value,omitempty"`
	ValueText string          `json:"value_text,omitempty"`
}

type Batch struct{ Signals []Signal }

type Run struct {
	ID            string         `json:"run_id"`
	Workload      string         `json:"workload"`
	Domain        string         `json:"domain"`
	StartedAt     int64          `json:"started_at"`
	EndedAt       int64          `json:"ended_at"`
	Outcome       string         `json:"outcome"`
	FailureMarker string         `json:"failure_marker,omitempty"`
	Env           map[string]any `json:"env"`
	Config        map[string]any `json:"config"`
	Closed        bool           `json:"closed"`
}

type RunDetail struct {
	Run     Run      `json:"run"`
	Signals []Signal `json:"signals"`
	Context []Signal `json:"context"`
}
