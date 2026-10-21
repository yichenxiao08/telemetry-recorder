"""Synthetic trace acceptance only. Does not read SQLite or generate reserved seeds."""
import argparse
import hashlib
import importlib
import json
import subprocess
import sys
import urllib.request
from pathlib import Path

from recorder import Recorder
from scout_probe import Injector, load_config
from spike_support import Serving


def exercise(out):
    out.mkdir(parents=True, exist_ok=True)
    root = Path(__file__).resolve().parent
    repo = root.parents[1]
    paths = [root / "fixtures/lane_target.py", root / "fixtures/pipeline_target.py"]
    before = {p: hashlib.sha256(p.read_bytes()).hexdigest() for p in paths}
    profile = repo / "benchmark/problems/lane-01/profile.json"
    bounds = ["--from", "2026-01-01T00:00:00Z", "--to", "2027-01-01T00:00:00Z", "--limit", "200", "--nodes", "100"]
    traces, drops = {}, {}
    with Serving(out / "fresh.db") as server:
        sys.path.insert(0, str(root / "fixtures"))
        for module_name, config, calls in [
            ("lane_target", repo / "benchmark/problems/lane-01/signalscout.yaml", [("ab" * 16, 8), ("ac" * 16, 9)]),
            ("pipeline_target", root / "fixtures/pipeline.yaml", [("bd" * 16, 3)]),
        ]:
            cfg = load_config(config)
            names = {name for p in cfg["probes"] for name in p.get("capture", {}).get("args", [])}
            names.add("output")
            recorder = Recorder(server.endpoint, capture_fields=names, service="synthetic.trace.acceptance")
            injector = Injector(recorder, cfg["probes"])
            injector.install()
            try:
                module = importlib.import_module(module_name)
                for run_id, value in calls:
                    with recorder.run("synthetic.trace." + module_name, run_id=run_id):
                        if module_name == "lane_target":
                            module.decide(.50, "left", value, 10, [0] * 7)
                        else:
                            module.turn(value)
                assert recorder.close(10), recorder.snapshot()
                facts = recorder.snapshot()
                drops.update({module_name + ":" + k: v for k, v in facts.items() if (k.startswith("drop") or k == "export_failed") and v})
                assert not drops, drops
                for run_id, _ in calls:
                    detail = server.show(run_id)
                    decisions = [s for s in detail["signals"] if s.get("attrs", {}).get("signalscout.record.kind") == "decision"]
                    assert decisions
                    for signal in decisions:
                        envelope = json.loads(signal["attrs"]["signalscout.record.json"])
                        ref = envelope["output"]["ref"]
                        command = [str(server.binary), "--db", str(server.database), "trace", ref, "--json", *bounds]
                        if module_name == "lane_target":
                            command += ["--profile", str(profile)]
                        report = json.loads(subprocess.check_output(command))
                        assert report["trace"]["visited"] == 1, report
                        assert report["scope"]["truncated"] is False
                        assert report["trace"]["nodes"][0]["evidence"]["id"] == signal["id"]
                        traces[ref] = report
            finally:
                injector.uninstall()
                recorder.close(2)
        checked = 0
        for report in traces.values():
            for node in report["trace"]["nodes"]:
                for evidence in [node["evidence"], *node["edge_evidence"]]:
                    path, anchor = evidence["url"].split("#")
                    with urllib.request.urlopen(server.endpoint + path, timeout=10) as response:
                        assert response.status == 200
                        assert 'id="' + anchor + '"' in response.read().decode(), evidence
                    checked += 1
        lane = [r for r in traces.values() if r["trace"]["nodes"][0]["step"]["id"] == "lane_target.decide"]
        stale, fresh = lane
        command = [str(server.binary), "--db", str(server.database), "trace", stale["trace"]["target"],
                   "--profile", str(profile), "--compare-to", fresh["trace"]["target"], *bounds]
        text = subprocess.check_output(command, text=True)
        assert "invariant: violated (coverage 3/3)" in text
        assert "Descriptive comparison; no ranking or score" in text
        assert "derived from" not in text and "caused by" not in text
        (out / "stale-trace.txt").write_text(text, encoding="utf-8")
        # Native evidence ID and run + step-name selectors resolve the same output.
        for selector in [[stale["trace"]["nodes"][0]["evidence"]["id"]], ["ab" * 16, "lane_target.decide"]]:
            result = json.loads(subprocess.check_output([str(server.binary), "--db", str(server.database), "trace", *selector, "--json", *bounds]))
            assert result["trace"]["nodes"][0]["id"] == stale["trace"]["nodes"][0]["id"]
        assert all(hashlib.sha256(p.read_bytes()).hexdigest() == digest for p, digest in before.items())
        result = {"synthetic_only": True, "target_source_lines_changed": 0,
                  "stale_invariant": stale["trace"]["nodes"][0]["invariant"]["status"],
                  "fresh_invariant": fresh["trace"]["nodes"][0]["invariant"]["status"],
                  "anchors_checked": checked, "pipeline_decisions": len(traces) - len(lane), "drops": drops,
                  "target_sha256": {p.name: digest for p, digest in before.items()}, "traces": list(traces.values())}
        (out / "report.json").write_text(json.dumps(result, indent=2) + "\n", encoding="utf-8")
        return result


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--out", type=Path, required=True, help="New synthetic output directory")
    result = exercise(parser.parse_args().out)
    print(json.dumps({k: v for k, v in result.items() if k != "traces"}, indent=2))
