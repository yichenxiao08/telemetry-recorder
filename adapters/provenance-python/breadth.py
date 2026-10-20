"""Synthetic S0b replay and HTTP assertions. Only Go opens SQLite, read-only."""
import argparse
import collections
import hashlib
import json
import subprocess
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path

import yaml
from count_instrumentation import count
from spike_support import Serving


def post(endpoint, path, body):
    request = urllib.request.Request(endpoint+path, json.dumps(body).encode(),
                                     {"Content-Type": "application/json"})
    with urllib.request.urlopen(request, timeout=20) as response:
        return response.read()


def inspect(repo, db):
    return json.loads(subprocess.check_output([str(repo/"bin/provenancespike.exe"), "-db", str(db)]))


def decoded(snapshot):
    return [json.loads(row["data"]) for row in snapshot["signals"]]


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--out", type=Path, required=True)
    args = parser.parse_args()
    root = Path(__file__).resolve().parent
    repo = root.parents[1]
    args.out.mkdir(parents=True, exist_ok=True)
    report = {"synthetic_only": True, "engine_changed": False, "fixtures": {}}
    source = repo/".cache/shop-demo/run-73122371/signalscout.db"
    source_snapshot = inspect(repo, source)
    source_signals = decoded(source_snapshot)
    source_resources = {v["resource"]["resource_id"] for v in source_signals}
    source_counts = dict(collections.Counter(s["kind"] for s in source_snapshot["signals"]))
    exported = subprocess.check_output([str(repo/"bin/provenancespike.exe"), "-db", str(source), "-export"])
    (args.out/"original-c.ndjson").write_bytes(exported)
    with Serving(args.out/"fresh.db") as serving:
        report["endpoint"] = serving.endpoint
        for name, module, config in [("A", "frame_target", "signalscout.yaml"),
                                      ("B", "pipeline_target", "pipeline.yaml"),
                                      ("B_optional", "pipeline_enriched", "pipeline-enriched.yaml")]:
            target = root/"fixtures"/(module+".py")
            before = target.read_bytes()
            baseline = args.out/(module+"-before.py")
            baseline.write_bytes(before)
            cfg = yaml.safe_load((root/"fixtures"/config).read_text())
            cfg["endpoint"] = serving.endpoint
            config_path = args.out/(name+".yaml")
            config_path.write_text(yaml.safe_dump(cfg), encoding="utf-8")
            doctor = args.out/(name+"-doctor.json")
            result = subprocess.run([sys.executable,"-m","scout_probe","--config",str(config_path),
                "--doctor",str(doctor),"--path",str(root/"fixtures"),module+":main"],
                check=True, capture_output=True, text=True)
            (args.out/(name+"-application.json")).write_text(result.stdout, encoding="utf-8")
            assert before == target.read_bytes(), "injector changed source"
            facts = json.loads(doctor.read_text())
            assert all(p["status"] == "attached" for p in facts["probes"]), facts
            assert not any(v for k,v in facts["counters"].items() if k.startswith("drop") or k=="export_failed"), facts
            report["fixtures"][name] = {"doctor": facts, "application":json.loads(result.stdout),
                "instrumentation":count(target, baseline, root/"fixtures"/config)}
        report["fixtures"]["B_optional"]["added_instrumentation"] = count(
            root/"fixtures/pipeline_enriched.py",root/"fixtures/pipeline_target.py",root/"fixtures/pipeline-enriched.yaml")
        # Original C records only. Six requests per 100ms <=60 signal records/s;
        # span events are embedded in their original spans, never replayed twice.
        replayed = collections.Counter()
        for index,line in enumerate(exported.splitlines()):
            item=json.loads(line)
            post(serving.endpoint,item["path"],item["body"])
            replayed[item["path"]] += 1
            if index % 6 == 5:
                time.sleep(.1)
        time.sleep(7)  # Default idle finalization, no shortened production timeout.
        snapshot = inspect(repo, serving.database)
        signals = decoded(snapshot)
        c_signals = [s for s in signals if s["resource"]["resource_id"] in source_resources]
        assert dict(collections.Counter(s["kind"] for s in c_signals)) == source_counts
        def original_signature(signal):
            return (signal["kind"], signal["resource"]["resource_id"],
                    json.dumps(signal.get("scope"), sort_keys=True),
                    json.dumps(signal.get("raw"), sort_keys=True))
        assert collections.Counter(map(original_signature, c_signals)) == collections.Counter(map(original_signature, source_signals)), "original OTLP changed on replay"
        source_ids = {s["id"] for s in source_signals}
        replay_ids = {s["id"] for s in c_signals}
        report["C"]={"source_counts":source_counts,"replay_counts":dict(collections.Counter(s["kind"] for s in c_signals)),
            "ordinary_otlp_requests":dict(replayed),"native_id_overlap":len(source_ids & replay_ids),"source_ids":len(source_ids),
            "saved_diff_results_copied":0}
        report["http"] = []
        for name,workload in [("A","frame_target.choose_lane"),("B","pipeline_target.turn"),("C","POST /checkout")]:
            runs=[r for r in snapshot["runs"] if r["workload"]==workload]
            assert runs, (name, sorted({r["workload"] for r in snapshot["runs"]}))
            selected = runs[0]
            detail = serving.show(selected["run_id"])
            evidence = next(s for s in detail["signals"] if s["kind"]=="span")
            anchor = "e-"+hashlib.sha256(evidence["id"].encode()).hexdigest()
            paths=["/runs", "/run?"+urllib.parse.urlencode({"id":selected["run_id"]}),
                "/telemetry?"+urllib.parse.urlencode({"mode":"spans","trace":selected["run_id"],"from":"2026-10-01T00:00:00Z","to":"2026-10-31T00:00:00Z","limit":100})]
            for path in paths:
                with urllib.request.urlopen(serving.endpoint+path,timeout=20) as response:
                    body=response.read().decode()
                    assert response.status==200
                    if path.startswith("/run?"):
                        assert 'id="'+anchor+'"' in body
                    report["http"].append({"behavior":name,"path":path,"status":200,"evidence":evidence["id"],"anchor":anchor})
        # One later simulated review targets a real HTTP-200 success, without relabeling outcome.
        b_runs=[r for r in snapshot["runs"] if r["workload"]=="pipeline_target.turn"]
        chosen=b_runs[3]
        before_scopes=[s for s in snapshot["scopes"] if s["run_id"]==chosen["run_id"]]
        detail=serving.show(chosen["run_id"])
        owned=next(s for s in detail["signals"] if s["kind"]=="span")
        resource=json.loads(json.dumps(owned["resource"]["raw"]))
        stamp=chosen["ended_at"]+60_000_000_000
        label={"version":1,"kind":"label","synthetic":True,"reviewer":"simulated-reviewer", "rubric":"synthetic-coaching-v1", "label":"too-strict", "target":owned["id"],"created_at_ns":str(stamp)}
        record={"timeUnixNano":str(stamp),"traceId":chosen["run_id"],"spanId":owned["span_id"],"severityNumber":9,
            "body":{"stringValue":"synthetic review"},"attributes":[{"key":"signalscout.record.kind","value":{"stringValue":"label"}},
            {"key":"signalscout.record.json","value":{"stringValue":json.dumps(label)}}]}
        post(serving.endpoint,"/v1/logs",{"resourceLogs":[{"resource":resource["resource"],"scopeLogs":[{"logRecords":[record]}]}]})
        after=inspect(repo,serving.database)
        after_run=next(r for r in after["runs"] if r["run_id"]==chosen["run_id"])
        after_scopes=[s for s in after["scopes"] if s["run_id"]==chosen["run_id"]]
        report["late_label"]={"label":label,"before_run":chosen,"after_run":after_run,"before_scopes":before_scopes,"after_scopes":after_scopes}
        assert after_run["outcome"]==chosen["outcome"]=="pass"
        assert after_run["ended_at"]==stamp
        assert max(s["end_ns"] for s in after_scopes)-max(s["end_ns"] for s in before_scopes)==60_000_000_000
        try:
            urllib.request.urlopen(urllib.request.Request(serving.endpoint+"/run/diff?"+urllib.parse.urlencode({"id":chosen["run_id"]}),b""),timeout=20)
            raise AssertionError("success accepted by failing-only diff")
        except urllib.error.HTTPError as error:
            assert error.code==409
            report["successful_mark_diff"]={"status":error.code,"body":error.read().decode()}
        report["run_counts"] = dict(collections.Counter(r["workload"] for r in snapshot["runs"]))
        (args.out/"snapshot.json").write_text(json.dumps(after,indent=2),encoding="utf-8")
        (args.out/"report.json").write_text(json.dumps(report,indent=2),encoding="utf-8")
        print(json.dumps({"report":str(args.out/"report.json"),"C":report["C"],"run_counts":report["run_counts"]}))


if __name__=="__main__":
    main()
