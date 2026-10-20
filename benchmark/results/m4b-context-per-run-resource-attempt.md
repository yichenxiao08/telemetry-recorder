# Unsuccessful context reader attempt: per-run-resource

Review-only diff; tabs are expanded and trailing whitespace is removed, so this is not an apply-ready patch.
The associated failed race logs are retained alongside this file.

```diff
diff --git a/internal/store/telemetry.go b/internal/store/telemetry.go
index bb743ae..342d2e4 100644
--- a/internal/store/telemetry.go
+++ b/internal/store/telemetry.go
@@ -3,6 +3,7 @@ package store
 import (
     "context"
     "database/sql"
+    "encoding/json"
     "fmt"
     "sort"

@@ -104,47 +105,158 @@ func (r *Reader) ServiceOverview(ctx context.Context, f telemetry.Filter) ([]tel
     return out, nil
 }

-const contextQuery = `SELECT s.data,r.attrs,r.raw FROM signals s JOIN resources r ON r.resource_id=s.resource_id WHERE s.evidence_id IN (SELECT c.evidence_id FROM context_matches c JOIN signals e ON e.evidence_id=c.evidence_id WHERE c.run_id=? AND e.ts_ns>=? AND e.ts_ns<=?) AND s.ts_ns>=? AND s.ts_ns<=? ORDER BY s.ts_ns,s.evidence_id LIMIT ?`
+const contextResourcesQuery = `SELECT a.resource_id,a.attrs,a.raw,c.rule,c.matched_resource_id,c.start_ns,c.end_ns
+ FROM context_scopes c CROSS JOIN resources a ON a.resource_id=c.match_value
+ WHERE c.run_id=? AND c.rule='exact_resource' AND c.start_ns<=? AND c.end_ns>=?
+ UNION ALL
+ SELECT a.resource_id,a.attrs,a.raw,c.rule,c.matched_resource_id,c.start_ns,c.end_ns
+ FROM context_scopes c CROSS JOIN resources a ON json_extract(a.attrs,'$."host.name"')=c.match_value
+ WHERE c.run_id=? AND c.rule='host.name' AND json_type(a.attrs,'$."host.name"')='text' AND c.start_ns<=? AND c.end_ns>=?
+ UNION ALL
+ SELECT a.resource_id,a.attrs,a.raw,c.rule,c.matched_resource_id,c.start_ns,c.end_ns
+ FROM context_scopes c CROSS JOIN resources a ON json_extract(a.attrs,'$."service.instance.id"')=c.match_value
+ WHERE c.run_id=? AND c.rule='service.instance.id' AND json_type(a.attrs,'$."service.instance.id"')='text' AND c.start_ns<=? AND c.end_ns>=?
+ ORDER BY 1,4,5 LIMIT 50001`

+const contextQuery = `SELECT s.data,s.ts_ns FROM signals s INDEXED BY signals_context
+ WHERE s.resource_id=? AND s.run_id='' AND s.kind IN ('log','metric')
+ AND s.ts_ns>=? AND s.ts_ns<=? ORDER BY s.ts_ns,s.evidence_id LIMIT ?`
+
+type contextRule struct {
+    match    model.ContextMatch
+    from, to int64
+}
+type contextResource struct {
+    resource model.Resource
+    rules    []contextRule
+    from, to int64
+}
+
+// Resource/rule matching does not depend on individual points. Resolve it once
+// on the read transaction's snapshot, then read each resource's points once.
+// Resource metadata is shared read-only within this returned snapshot; the
+// writer never touches these decoded maps. Per-point provenance stays separate.
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
+    rows, err := tx.QueryContext(ctx, contextResourcesQuery, id, to, from, id, to, from, id, to, from)
     if err != nil {
-        return nil, err
+        return nil, fmt.Errorf("resolve context resources: %w", err)
     }
-    defer func() { _ = rows.Close() }()
-    byID := map[string][]model.ContextMatch{}
-    count := 0
+    resources := map[string]*contextResource{}
+    metadataCount := 0
     for rows.Next() {
-        var evidence string
-        var match model.ContextMatch
-        if err = rows.Scan(&evidence, &match.Rule, &match.ResourceID); err != nil {
-            return nil, err
+        var resourceID, attrs, raw string
+        var rule contextRule
+        if err = rows.Scan(&resourceID, &attrs, &raw, &rule.match.Rule, &rule.match.ResourceID, &rule.from, &rule.to); err != nil {
+            break
+        }
+        metadataCount++
+        if metadataCount > 50000 {
+            err = EvidenceLimit
+            break
         }
-        count++
-        if count > 50000 {
-            return nil, EvidenceLimit
+        rule.from = max(rule.from, from)
+        rule.to = min(rule.to, to)
+        resource, ok := resources[resourceID]
+        if !ok {
+            resource = &contextResource{resource: model.Resource{ID: resourceID, Raw: json.RawMessage(raw)}, from: rule.from, to: rule.to}
+            if err = json.Unmarshal([]byte(attrs), &resource.resource.Attrs); err != nil {
+                break
+            }
+            resources[resourceID] = resource
         }
-        byID[evidence] = append(byID[evidence], match)
+        resource.from = min(resource.from, rule.from)
+        resource.to = max(resource.to, rule.to)
+        resource.rules = append(resource.rules, rule)
+    }
+    if err == nil {
+        err = rows.Err()
     }
-    if err = rows.Err(); err != nil {
+    closeErr := rows.Close()
+    if err != nil {
         return nil, err
     }
-    for i := range signals {
-        signals[i].ContextMatches = byID[signals[i].ID]
+    if closeErr != nil {
+        return nil, closeErr
+    }
+    keys := []string{}
+    for key := range resources {
+        keys = append(keys, key)
+    }
+    sort.Strings(keys)
+    signals := []model.Signal{}
+    scanned, matches := 0, 0
+    for _, key := range keys {
+        resource := resources[key]
+        records, e := tx.QueryContext(ctx, contextQuery, key, resource.from, resource.to, 50001-scanned)
+        if e != nil {
+            return nil, e
+        }
+        for records.Next() {
+            var data string
+            var timestamp int64
+            var signal model.Signal
+            if e = records.Scan(&data, &timestamp); e != nil {
+                break
+            }
+            scanned++
+            if scanned > 50000 {
+                e = EvidenceLimit
+                break
+            }
+            if e = json.Unmarshal([]byte(data), &signal); e != nil {
+                break
+            }
+            signal.Resource = resource.resource
+            signal.ContextMatches = nil
+            for _, rule := range resource.rules {
+                if timestamp >= rule.from && timestamp <= rule.to {
+                    signal.ContextMatches = append(signal.ContextMatches, rule.match)
+                }
+            }
+            if len(signal.ContextMatches) == 0 {
+                continue
+            }
+            sort.Slice(signal.ContextMatches, func(i, j int) bool {
+                a, b := signal.ContextMatches[i], signal.ContextMatches[j]
+                if a.Rule == b.Rule {
+                    return a.ResourceID < b.ResourceID
+                }
+                return a.Rule < b.Rule
+            })
+            matches += len(signal.ContextMatches)
+            if matches > 50000 {
+                e = EvidenceLimit
+                break
+            }
+            signals = append(signals, signal)
+            if len(signals) > limit {
+                e = EvidenceLimit
+                break
+            }
+        }
+        if e == nil {
+            e = records.Err()
+        }
+        closeErr = records.Close()
+        if e != nil {
+            return nil, e
+        }
+        if closeErr != nil {
+            return nil, closeErr
+        }
     }
+    sort.Slice(signals, func(i, j int) bool {
+        if signals[i].TS == signals[j].TS {
+            return signals[i].ID < signals[j].ID
+        }
+        return signals[i].TS < signals[j].TS
+    })
     return signals, nil
 }
+
 func (r *Reader) RunContext(ctx context.Context, id string, from, to int64, limit int) ([]model.Signal, error) {
     tx, err := r.reader.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
     if err != nil {
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
index 8487317..5e25242 100644
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
@@ -91,7 +91,7 @@ func TestRunContextQueryPlan(t *testing.T) {
     if err = rows.Err(); err != nil {
         t.Fatal(err)
     }
-    if bounded != 3 || strings.Contains(plan.String(), "SCAN s") {
+    if bounded != 1 || strings.Contains(plan.String(), "SCAN s") {
         t.Fatal(plan.String())
     }
 }
```
