// Package completeness classifies capture completeness of runs from the
// opt-in manifests the Python recorder emits as ordinary telemetry.
//
// A run is complete only when a final marker proves every selected item was
// serialized, admitted and acknowledged. A missing or unreadable marker can
// never produce complete. Pure functions only: no I/O, no storage.
package completeness

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const (
	AttrKind = "signalscout.record.kind"
	AttrJSON = "signalscout.manifest.json"

	KindManifest = "capture-manifest"
	KindScope    = "capture-scope"
	// ScopeWorkload is the reserved root-span name of capture-scope traces. A
	// scope trace is never a cohort candidate and never counted as a run.
	ScopeWorkload = "signalscout.capture.scope"
	// ManifestSpanID is the deterministic child span carrying a run's manifest.
	ManifestSpanID = "ffffffffffffff01"

	Version        = 1
	MaxJSONBytes   = 32768
	MaxScopeChunks = 64  // chunks read per scope; more makes the population unknown
	MaxChunkIDs    = 256 // IDs accepted per scope record
	maxCount       = int64(1) << 40
)

type Status string

const (
	Complete   Status = "complete"
	Partial    Status = "partial"
	Unknown    Status = "unknown"
	Absent     Status = "absent"
	Unassessed Status = "unassessed"
)

// Bucket is one probe's (or the root/direct bucket's) counters for one run.
type Bucket struct {
	Seen       int64 `json:"seen"`
	SampledOut int64 `json:"sampled_out"`
	Selected   int64 `json:"selected"`
	Serialized int64 `json:"serialized"`
	Admitted   int64 `json:"admitted"`
	Rejected   int64 `json:"rejected"`
	Dropped    int64 `json:"dropped"`
	Ack        int64 `json:"ack"`
	Failed     int64 `json:"failed"`
	Unknown    int64 `json:"unknown"`
	DetailLoss int64 `json:"detail_loss"`
}

type Manifest struct {
	V       int               `json:"v"`
	Epoch   string            `json:"epoch"`
	Seq     int               `json:"seq"`
	Final   bool              `json:"final"`
	Buckets map[string]Bucket `json:"buckets"`
}

type ScopeRecord struct {
	V                  int      `json:"v"`
	Epoch              string   `json:"epoch"`
	Chunk              int      `json:"chunk"`
	IDs                []string `json:"ids"`
	Final              bool     `json:"final"`
	Chunks             int      `json:"chunks"`
	Expected           int      `json:"expected"`
	LedgerOverflow     int      `json:"ledger_overflow"`
	PopulationOverflow int      `json:"population_overflow"`
}

func payload(attrs map[string]any, kind string) (string, bool, error) {
	if attrs[AttrKind] != kind {
		return "", false, nil
	}
	body, ok := attrs[AttrJSON].(string)
	if !ok || len(body) > MaxJSONBytes {
		return "", true, fmt.Errorf("%s payload missing or oversized", kind)
	}
	return body, true, nil
}

// ParseManifest returns (nil, false, nil) for any span that is not a manifest.
func ParseManifest(attrs map[string]any) (*Manifest, bool, error) {
	body, is, err := payload(attrs, KindManifest)
	if !is || err != nil {
		return nil, is, err
	}
	var m Manifest
	if err = json.Unmarshal([]byte(body), &m); err != nil {
		return nil, true, fmt.Errorf("manifest JSON: %w", err)
	}
	return &m, true, nil
}

// ParseScope returns (nil, false, nil) for any span that is not a scope record.
func ParseScope(attrs map[string]any) (*ScopeRecord, bool, error) {
	body, is, err := payload(attrs, KindScope)
	if !is || err != nil {
		return nil, is, err
	}
	var r ScopeRecord
	if err = json.Unmarshal([]byte(body), &r); err != nil {
		return nil, true, fmt.Errorf("scope JSON: %w", err)
	}
	r.Epoch = strings.ToLower(r.Epoch)
	for i, id := range r.IDs {
		r.IDs[i] = strings.ToLower(id)
	}
	return &r, true, nil
}

// Scope is the assembled capture-scope inventory of one producer epoch.
type Scope struct {
	Epoch string
	IDs   map[string]struct{}
	// Expected is the producer's own count, meaningful only when Final.
	Expected        int
	Final           bool
	PopulationKnown bool
	Reason          string
}

// AssembleScope merges chunk records. The population is known only when the
// tail record is present, every chunk 0..chunks-1 arrived, the listed IDs match
// the producer's expected count, and neither overflow counter is set.
func AssembleScope(epoch string, records []*ScopeRecord, truncated bool) Scope {
	s := Scope{Epoch: epoch, IDs: map[string]struct{}{}}
	chunks := map[int]*ScopeRecord{}
	var tail *ScopeRecord
	for _, r := range records {
		if r == nil || r.V != Version || r.Epoch != epoch || r.Chunk < 0 || r.Chunk >= MaxScopeChunks || len(r.IDs) > MaxChunkIDs {
			s.Reason = "invalid or foreign scope record ignored"
			continue
		}
		chunks[r.Chunk] = r
		if r.Final {
			tail = r
		}
	}
	for _, r := range chunks {
		for _, id := range r.IDs {
			s.IDs[id] = struct{}{}
		}
	}
	switch {
	case truncated:
		s.Reason = "scope has more records than the read bound"
	case tail == nil:
		s.Reason = "scope tail record missing"
	default:
		s.Final, s.Expected = true, tail.Expected
		missing := tail.Chunks < 1 || tail.Chunks > MaxScopeChunks
		for i := 0; i < tail.Chunks && !missing; i++ {
			_, ok := chunks[i]
			missing = !ok
		}
		switch {
		case missing:
			s.Reason = "scope chunk missing"
		case tail.LedgerOverflow > 0 || tail.PopulationOverflow > 0:
			s.Reason = "producer ledger or scope capacity exhausted"
		case len(s.IDs) != tail.Expected:
			s.Reason = "listed run IDs differ from the producer's expected count"
		default:
			s.PopulationKnown = true
		}
	}
	return s
}

// Evidence is everything known about one run when it is classified.
type Evidence struct {
	Seen          bool      // a run row exists (any workload)
	InScope       bool      // a relevant scope record lists the run
	Manifest      *Manifest // nil when none was stored
	ManifestError string    // a manifest span exists but is unreadable
	StoredSpans   int64     // stored spans of the run, excluding the manifest span
}

type Verdict struct {
	Status       Status
	Reason       string
	LostItems    int64 // confirmed lost selected items (unserialized + rejected + failed)
	UnknownItems int64 // terminal-unknown items
	DetailLoss   int64
	// MixedUnknown marks a partial run that also has unresolved unknown evidence.
	MixedUnknown bool
}

// Totals validates per-bucket conservation and returns sums.
func Totals(m *Manifest) (lost, unknown, ack, detail int64, err error) {
	if m.V != Version {
		return 0, 0, 0, 0, fmt.Errorf("unsupported manifest version %d", m.V)
	}
	if !m.Final {
		return 0, 0, 0, 0, fmt.Errorf("final marker missing")
	}
	if len(m.Buckets) == 0 {
		return 0, 0, 0, 0, fmt.Errorf("manifest has no buckets")
	}
	keys := make([]string, 0, len(m.Buckets))
	for k := range m.Buckets {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		b := m.Buckets[k]
		for _, v := range []int64{b.Seen, b.SampledOut, b.Selected, b.Serialized, b.Admitted, b.Rejected, b.Dropped, b.Ack, b.Failed, b.Unknown, b.DetailLoss} {
			if v < 0 || v > maxCount {
				return 0, 0, 0, 0, fmt.Errorf("bucket %q has an out-of-range counter", k)
			}
		}
		switch {
		case b.Seen != b.SampledOut+b.Selected:
			err = fmt.Errorf("bucket %q: seen != sampled_out + selected", k)
		case b.Serialized > b.Selected:
			err = fmt.Errorf("bucket %q: serialized > selected", k)
		case b.Serialized != b.Admitted+b.Rejected:
			err = fmt.Errorf("bucket %q: serialized != admitted + rejected", k)
		case b.Dropped != b.Rejected:
			err = fmt.Errorf("bucket %q: dropped != rejected", k)
		case b.Admitted != b.Ack+b.Failed+b.Unknown:
			err = fmt.Errorf("bucket %q: admitted != ack + failed + unknown (items unsettled)", k)
		}
		if err != nil {
			return 0, 0, 0, 0, err
		}
		lost += b.Selected - b.Serialized + b.Rejected + b.Failed
		unknown += b.Unknown
		ack += b.Ack
		detail += b.DetailLoss
	}
	return lost, unknown, ack, detail, nil
}

// Classify applies the precedence: absent, unassessed, unknown (untrusted or
// missing marker), partial (confirmed loss), unknown (unresolved terminal
// state), complete. Confirmed loss wins over unresolved unknown items.
func Classify(e Evidence) Verdict {
	switch {
	case !e.Seen && e.InScope:
		return Verdict{Status: Absent, Reason: "listed by the capture scope but never stored"}
	case !e.Seen:
		return Verdict{Status: Unassessed, Reason: "not stored and not expected by any scope"}
	case e.ManifestError != "":
		return Verdict{Status: Unknown, Reason: "manifest unreadable: " + e.ManifestError}
	case e.Manifest == nil && e.InScope:
		return Verdict{Status: Unknown, Reason: "final marker missing"}
	case e.Manifest == nil:
		return Verdict{Status: Unassessed, Reason: "no manifest (legacy or unlisted run)"}
	}
	lost, unknown, ack, detail, err := Totals(e.Manifest)
	if err != nil {
		return Verdict{Status: Unknown, Reason: "manifest inconsistent: " + err.Error()}
	}
	v := Verdict{LostItems: lost, UnknownItems: unknown, DetailLoss: detail}
	short := e.StoredSpans < ack
	switch {
	case lost > 0:
		v.Status, v.Reason = Partial, "confirmed lost selected items"
		v.MixedUnknown = unknown > 0 || short
	case unknown > 0:
		v.Status, v.Reason = Unknown, "terminal delivery state unresolved"
	case short:
		v.Status, v.Reason = Unknown, "acknowledged spans not found in the store"
	default:
		v.Status, v.Reason = Complete, "final marker; every selected item acknowledged"
	}
	return v
}
