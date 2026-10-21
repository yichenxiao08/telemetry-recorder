package completeness

import (
	"encoding/json"
	"strings"
	"testing"
)

func ok(n int64) Bucket {
	return Bucket{Seen: n, Selected: n, Serialized: n, Admitted: n, Ack: n}
}

func manifest(buckets map[string]Bucket) *Manifest {
	return &Manifest{V: Version, Epoch: "e", Seq: 1, Final: true, Buckets: buckets}
}

func TestClassifyEveryStatus(t *testing.T) {
	lost := ok(3)
	lost.Admitted, lost.Ack, lost.Rejected, lost.Dropped = 2, 2, 1, 1
	unserialized := ok(3)
	unserialized.Serialized, unserialized.Admitted, unserialized.Ack = 2, 2, 2 // one selected call never became an item
	failed := ok(2)
	failed.Ack, failed.Failed = 1, 1
	unresolved := ok(2)
	unresolved.Ack, unresolved.Unknown = 1, 1
	unsettled := ok(2)
	unsettled.Ack = 1 // admitted 2 but only 1 settled
	sampled := ok(2)
	sampled.Seen, sampled.SampledOut = 10, 8 // sampling is an omission, not a loss
	for _, tc := range []struct {
		name string
		e    Evidence
		want Status
	}{
		{"complete", Evidence{Seen: true, Manifest: manifest(map[string]Bucket{"@root": ok(1), "p": ok(2)}), StoredSpans: 3}, Complete},
		{"sampling is not loss", Evidence{Seen: true, Manifest: manifest(map[string]Bucket{"p": sampled}), StoredSpans: 2}, Complete},
		{"queue rejection", Evidence{Seen: true, Manifest: manifest(map[string]Bucket{"p": lost}), StoredSpans: 2}, Partial},
		{"selected call never serialized", Evidence{Seen: true, Manifest: manifest(map[string]Bucket{"p": unserialized}), StoredSpans: 2}, Partial},
		{"confirmed failed delivery", Evidence{Seen: true, Manifest: manifest(map[string]Bucket{"p": failed}), StoredSpans: 1}, Partial},
		{"ambiguous delivery", Evidence{Seen: true, Manifest: manifest(map[string]Bucket{"p": unresolved}), StoredSpans: 1}, Unknown},
		{"items unsettled", Evidence{Seen: true, Manifest: manifest(map[string]Bucket{"p": unsettled}), StoredSpans: 1}, Unknown},
		{"acknowledged spans missing from store", Evidence{Seen: true, Manifest: manifest(map[string]Bucket{"p": ok(2)}), StoredSpans: 1}, Unknown},
		{"marker missing, run listed", Evidence{Seen: true, InScope: true}, Unknown},
		{"marker unreadable", Evidence{Seen: true, InScope: true, ManifestError: "x"}, Unknown},
		{"not final", Evidence{Seen: true, Manifest: &Manifest{V: Version, Buckets: map[string]Bucket{"p": ok(1)}}, StoredSpans: 1}, Unknown},
		{"unsupported version", Evidence{Seen: true, Manifest: &Manifest{V: 9, Final: true, Buckets: map[string]Bucket{"p": ok(1)}}, StoredSpans: 1}, Unknown},
		{"empty manifest", Evidence{Seen: true, Manifest: manifest(nil)}, Unknown},
		{"expected but never seen", Evidence{InScope: true}, Absent},
		{"legacy run", Evidence{Seen: true, StoredSpans: 5}, Unassessed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := Classify(tc.e); got.Status != tc.want {
				t.Fatalf("got %s (%s), want %s", got.Status, got.Reason, tc.want)
			}
		})
	}
}

func TestPartialWinsOverUnknownButKeepsTheUnknownVisible(t *testing.T) {
	b := ok(3)
	b.Admitted, b.Ack, b.Unknown, b.Rejected, b.Dropped = 2, 1, 1, 1, 1
	v := Classify(Evidence{Seen: true, Manifest: manifest(map[string]Bucket{"p": b}), StoredSpans: 1})
	if v.Status != Partial || !v.MixedUnknown || v.UnknownItems != 1 || v.LostItems != 1 {
		t.Fatalf("%+v", v)
	}
}

func TestInconsistentCountersAreUnknownNeverComplete(t *testing.T) {
	mutations := map[string]func(*Bucket){
		"seen":       func(b *Bucket) { b.Seen++ },
		"serialized": func(b *Bucket) { b.Serialized++ },
		"admitted":   func(b *Bucket) { b.Admitted++ },
		"dropped":    func(b *Bucket) { b.Dropped++ },
		"negative":   func(b *Bucket) { b.Ack = -1; b.Admitted = -1; b.Serialized = -1; b.Selected = -1; b.Seen = -1 },
		"huge":       func(b *Bucket) { b.Ack = 1 << 50 },
	}
	for name, mutate := range mutations {
		b := ok(2)
		mutate(&b)
		if got := Classify(Evidence{Seen: true, Manifest: manifest(map[string]Bucket{"p": b}), StoredSpans: 1 << 51}); got.Status == Complete {
			t.Errorf("%s: inconsistent manifest classified complete", name)
		}
	}
}

// A run without a readable final marker can never be complete, whatever else is known.
func TestNoMarkerNeverYieldsComplete(t *testing.T) {
	for _, seen := range []bool{false, true} {
		for _, scoped := range []bool{false, true} {
			for _, bad := range []string{"", "broken"} {
				for _, spans := range []int64{0, 1, 100} {
					v := Classify(Evidence{Seen: seen, InScope: scoped, ManifestError: bad, StoredSpans: spans})
					if v.Status == Complete || v.Status == Partial {
						t.Fatalf("seen=%v scoped=%v err=%q: %s", seen, scoped, bad, v.Status)
					}
				}
			}
		}
	}
}

func scopeJSON(r ScopeRecord) *ScopeRecord {
	r.V = Version
	if r.Epoch == "" {
		r.Epoch = "e"
	}
	return &r
}

func TestAssembleScope(t *testing.T) {
	tail := func(chunk, chunks, expected int, ids ...string) *ScopeRecord {
		return scopeJSON(ScopeRecord{Chunk: chunk, IDs: ids, Final: true, Chunks: chunks, Expected: expected})
	}
	body := func(chunk int, ids ...string) *ScopeRecord { return scopeJSON(ScopeRecord{Chunk: chunk, IDs: ids}) }
	for _, tc := range []struct {
		name      string
		records   []*ScopeRecord
		truncated bool
		known     bool
		ids       int
	}{
		{"single tail", []*ScopeRecord{tail(0, 1, 2, "a", "b")}, false, true, 2},
		{"chunks and tail", []*ScopeRecord{body(0, "a"), tail(1, 2, 2, "b")}, false, true, 2},
		{"duplicate delivery", []*ScopeRecord{body(0, "a"), body(0, "a"), tail(1, 2, 2, "b"), tail(1, 2, 2, "b")}, false, true, 2},
		{"tail missing", []*ScopeRecord{body(0, "a")}, false, false, 1},
		{"chunk missing", []*ScopeRecord{tail(1, 2, 2, "b")}, false, false, 1},
		{"count mismatch", []*ScopeRecord{tail(0, 1, 3, "a", "b")}, false, false, 2},
		{"truncated read", []*ScopeRecord{tail(0, 1, 1, "a")}, true, false, 1},
		{"foreign epoch ignored", []*ScopeRecord{scopeJSON(ScopeRecord{Epoch: "other", Final: true, Chunks: 1, Expected: 0})}, false, false, 0},
		{"overflow", []*ScopeRecord{scopeJSON(ScopeRecord{Chunk: 0, IDs: []string{"a"}, Final: true, Chunks: 1, Expected: 1, LedgerOverflow: 1})}, false, false, 1},
		{"population overflow", []*ScopeRecord{scopeJSON(ScopeRecord{Chunk: 0, IDs: []string{"a"}, Final: true, Chunks: 1, Expected: 1, PopulationOverflow: 2})}, false, false, 1},
		{"nothing", nil, false, false, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := AssembleScope("e", tc.records, tc.truncated)
			if s.PopulationKnown != tc.known || len(s.IDs) != tc.ids {
				t.Fatalf("known=%v ids=%d reason=%q", s.PopulationKnown, len(s.IDs), s.Reason)
			}
		})
	}
}

func TestParseRejectsOversizedAndIgnoresOrdinarySpans(t *testing.T) {
	if _, is, err := ParseManifest(map[string]any{"k": "v"}); is || err != nil {
		t.Fatal("ordinary span treated as a manifest")
	}
	if _, is, err := ParseManifest(map[string]any{AttrKind: KindManifest, AttrJSON: strings.Repeat("x", MaxJSONBytes+1)}); !is || err == nil {
		t.Fatal("oversized manifest accepted")
	}
	body, _ := json.Marshal(manifest(map[string]Bucket{"p": ok(1)}))
	m, is, err := ParseManifest(map[string]any{AttrKind: KindManifest, AttrJSON: string(body)})
	if !is || err != nil || m.Buckets["p"].Ack != 1 {
		t.Fatalf("%v %v %+v", is, err, m)
	}
	r, is, err := ParseScope(map[string]any{AttrKind: KindScope, AttrJSON: `{"v":1,"epoch":"AB","ids":["CD"]}`})
	if !is || err != nil || r.Epoch != "ab" || r.IDs[0] != "cd" {
		t.Fatalf("IDs are lower-cased to match stored trace IDs: %v %v %+v", is, err, r)
	}
}
