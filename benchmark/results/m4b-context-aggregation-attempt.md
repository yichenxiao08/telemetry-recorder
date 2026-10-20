# Unsuccessful context reader attempt: aggregation

Review-only diff; tabs are expanded and trailing whitespace is removed, so this is not an apply-ready patch.
The associated failed race logs are retained alongside this file.

```diff
diff --git a/internal/store/telemetry.go b/internal/store/telemetry.go
index bb743ae..da80665 100644
--- a/internal/store/telemetry.go
+++ b/internal/store/telemetry.go
@@ -3,6 +3,7 @@ package store
 import (
     "context"
     "database/sql"
+    "encoding/json"
     "fmt"
     "sort"

@@ -104,45 +105,68 @@ func (r *Reader) ServiceOverview(ctx context.Context, f telemetry.Filter) ([]tel
     return out, nil
 }

-const contextQuery = `SELECT s.data,r.attrs,r.raw FROM signals s JOIN resources r ON r.resource_id=s.resource_id WHERE s.evidence_id IN (SELECT c.evidence_id FROM context_matches c JOIN signals e ON e.evidence_id=c.evidence_id WHERE c.run_id=? AND e.ts_ns>=? AND e.ts_ns<=?) AND s.ts_ns>=? AND s.ts_ns<=? ORDER BY s.ts_ns,s.evidence_id LIMIT ?`
+// Resolve the bounded rule set once, aggregate provenance before loading large
+// raw records, then fetch each evidence record once. The extra row in each cap
+// detects overflow; it is never returned as a partial successful comparison.
+const contextQuery = `WITH matches AS MATERIALIZED (
+ SELECT c.evidence_id,c.rule,c.matched_resource_id FROM context_matches c
+ JOIN signals e ON e.evidence_id=c.evidence_id
+ WHERE c.run_id=? AND e.ts_ns>=? AND e.ts_ns<=? LIMIT 50001
+), grouped AS (
+ SELECT evidence_id,json_group_array(json_object('rule',rule,'matched_resource_id',matched_resource_id)) AS provenance
+ FROM matches GROUP BY evidence_id
+)
+SELECT s.data,r.attrs,r.raw,c.provenance FROM grouped c
+ JOIN signals s ON s.evidence_id=c.evidence_id
+ JOIN resources r ON r.resource_id=s.resource_id
+ ORDER BY s.ts_ns,s.evidence_id LIMIT ?`

 func readContext(ctx context.Context, tx *sql.Tx, id string, from, to int64, limit int) ([]model.Signal, error) {
     if from > to || limit < 1 || limit > 5000 {
         return nil, fmt.Errorf("context requires valid bounds and limit 1..5000")
     }
-    signals, err := readSignals(ctx, tx, contextQuery, id, from, to, from, to, limit+1)
-    if err != nil {
-        return nil, err
-    }
-    if len(signals) > limit {
-        return nil, EvidenceLimit
-    }
-    // Fetch provenance once for the bounded run/window, not once per record.
-    rows, err := tx.QueryContext(ctx, `SELECT c.evidence_id,c.rule,c.matched_resource_id FROM context_matches c JOIN signals e ON e.evidence_id=c.evidence_id WHERE c.run_id=? AND e.ts_ns>=? AND e.ts_ns<=? ORDER BY c.evidence_id,c.rule,c.matched_resource_id LIMIT 50001`, id, from, to)
+    rows, err := tx.QueryContext(ctx, contextQuery, id, from, to, limit+1)
     if err != nil {
         return nil, err
     }
     defer func() { _ = rows.Close() }()
-    byID := map[string][]model.ContextMatch{}
+    signals := []model.Signal{}
     count := 0
     for rows.Next() {
-        var evidence string
-        var match model.ContextMatch
-        if err = rows.Scan(&evidence, &match.Rule, &match.ResourceID); err != nil {
+        var data, attrs, raw, provenance string
+        var signal model.Signal
+        if err = rows.Scan(&data, &attrs, &raw, &provenance); err != nil {
             return nil, err
         }
-        count++
+        if err = json.Unmarshal([]byte(data), &signal); err != nil {
+            return nil, fmt.Errorf("decode context: %w", err)
+        }
+        if err = json.Unmarshal([]byte(attrs), &signal.Resource.Attrs); err != nil {
+            return nil, fmt.Errorf("decode context resource: %w", err)
+        }
+        signal.Resource.Raw = json.RawMessage(raw)
+        if err = json.Unmarshal([]byte(provenance), &signal.ContextMatches); err != nil {
+            return nil, fmt.Errorf("decode context provenance: %w", err)
+        }
+        sort.Slice(signal.ContextMatches, func(i, j int) bool {
+            a, b := signal.ContextMatches[i], signal.ContextMatches[j]
+            if a.Rule == b.Rule {
+                return a.ResourceID < b.ResourceID
+            }
+            return a.Rule < b.Rule
+        })
+        count += len(signal.ContextMatches)
         if count > 50000 {
             return nil, EvidenceLimit
         }
-        byID[evidence] = append(byID[evidence], match)
+        signals = append(signals, signal)
+        if len(signals) > limit {
+            return nil, EvidenceLimit
+        }
     }
     if err = rows.Err(); err != nil {
         return nil, err
     }
-    for i := range signals {
-        signals[i].ContextMatches = byID[signals[i].ID]
-    }
     return signals, nil
 }
 func (r *Reader) RunContext(ctx context.Context, id string, from, to int64, limit int) ([]model.Signal, error) {
diff --git a/internal/store/telemetry_behavior_test.go b/internal/store/telemetry_behavior_test.go
index 3805c6a..51552ea 100644
--- a/internal/store/telemetry_behavior_test.go
+++ b/internal/store/telemetry_behavior_test.go
@@ -4,6 +4,7 @@ import (
     "context"
     "github.com/signalscout/signalscout/internal/model"
     "github.com/signalscout/signalscout/internal/telemetry"
+    "reflect"
     "testing"
     "time"
 )
@@ -56,7 +57,10 @@ func TestBoundedContextIncludesRulesAndRefusesTruncation(t *testing.T) {
     ctx := context.Background()
     s := testStore(t, 0)
     root := fixture("span:root", "span", "trace", "r", 10, 100)
+    root.Resource.Attrs["host.name"] = "host"
+    root.Resource.Attrs["service.instance.id"] = "instance"
     ambient := fixture("log:ambient", "log", "", "r", 20, 20)
+    ambient.Resource = root.Resource
     other := ambient
     other.ID = "log:other"
     if err := s.Submit(ctx, model.Batch{Signals: []model.Signal{root, ambient, other}}); err != nil {
@@ -69,6 +73,16 @@ func TestBoundedContextIncludesRulesAndRefusesTruncation(t *testing.T) {
     if err != nil || len(signals) != 2 || len(signals[0].ContextMatches) == 0 {
         t.Fatalf("%+v %v", signals, err)
     }
+    legacy, err := s.Show(ctx, "trace")
+    if err != nil {
+        t.Fatal(err)
+    }
+    if !reflect.DeepEqual(legacy.Context, signals) {
+        t.Fatalf("bounded context changed records or provenance: old=%+v new=%+v", legacy.Context, signals)
+    }
+    if len(signals[0].ContextMatches) != 3 {
+        t.Fatal("lost context rules")
+    }
     if _, err = s.RunContext(ctx, "trace", 1, 101, 1); err != EvidenceLimit {
         t.Fatal(err)
     }
diff --git a/internal/store/telemetry_test.go b/internal/store/telemetry_test.go
index 8487317..ddc3760 100644
--- a/internal/store/telemetry_test.go
+++ b/internal/store/telemetry_test.go
@@ -70,7 +70,7 @@ func TestExplorerQueryPlans(t *testing.T) {

 func TestRunContextQueryPlan(t *testing.T) {
     s := testStore(t, 0)
-    rows, err := s.reader.Query("EXPLAIN QUERY PLAN "+contextQuery, "r", int64(1), int64(1000), int64(1), int64(1000), 100)
+    rows, err := s.reader.Query("EXPLAIN QUERY PLAN "+contextQuery, "r", int64(1), int64(1000), 100)
     if err != nil {
         t.Fatal(err)
     }
```
