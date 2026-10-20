"""Execute ONLY synthetic pre-registered lane-01 through unchanged probes/OTLP."""
import argparse
import datetime
import hashlib
import importlib
import json
import random
import subprocess
import sys
import time
import urllib.request
from pathlib import Path

import yaml
from count_instrumentation import count
from recorder import Recorder
from scout_probe import Injector, load_config
from spike_support import Serving


def rows(seed, variant, spec):
    rng=random.Random(seed)
    settings=spec["variants"][variant]
    n=spec["population"]["problem_frames"]
    stale=set(rng.sample(range(n), round(n*settings["problem_stale_fraction"])))
    output=[]
    for i in range(n):
        for problem in (True,False):
            low,high=settings["problem_gap_range" if problem else "reference_gap_range"]
            noise=[rng.uniform(-1,1)+(settings["decoy_problem_offset"] if problem else 0) for _ in range(7)]
            output.append({"gap":rng.uniform(low,high),"previous_lane":"left" if problem else "right",
                "age":2 if problem and i in stale else 1,"noise":noise,"expected":"right"})
    rng.shuffle(output)
    return output


def utc(ns):
    return datetime.datetime.fromtimestamp(ns/1e9,datetime.timezone.utc).isoformat().replace("+00:00","Z")


def post(endpoint,path,payload):
    request=urllib.request.Request(endpoint+path,json.dumps(payload).encode(),{"Content-Type":"application/json"})
    with urllib.request.urlopen(request,timeout=20) as response:
        return json.loads(response.read())


def main():
    parser=argparse.ArgumentParser()
    parser.add_argument("--out",type=Path,required=True)
    args=parser.parse_args()
    root=Path(__file__).resolve().parent
    repo=root.parents[1]
    subprocess.run(["git","merge-base","--is-ancestor","b537ee154f004cc93a24571156fe1feb856b226f","HEAD"],cwd=repo,check=True)
    subprocess.run(["git","diff","--exit-code","b537ee154f004cc93a24571156fe1feb856b226f","--",
        "benchmark/problems/lane-01/problem.yaml","docs/slice1-scoring.md"],cwd=repo,check=True)
    subprocess.run(["git","diff","--exit-code","HEAD","--","internal/cohort","internal/provenance",
        "benchmark/problems/lane-01/profile.json","adapters/provenance-python/lane01.py",
        "adapters/provenance-python/fixtures/lane_target.py"],cwd=repo,check=True)
    spec=yaml.safe_load((repo/"benchmark/problems/lane-01/problem.yaml").read_text())
    profile=repo/"benchmark/problems/lane-01/profile.json"
    cfg=load_config(repo/"benchmark/problems/lane-01/signalscout.yaml")
    args.out.mkdir(parents=True,exist_ok=True)
    target=root/"fixtures/lane_target.py"
    before=target.read_bytes()
    snapshot=args.out/"lane-target-before.py"
    snapshot.write_bytes(before)
    report={"synthetic_only":True,"preregistration_sha":"b537ee154f004cc93a24571156fe1feb856b226f",
        "revision":subprocess.check_output(["git","rev-parse","HEAD"],cwd=repo,text=True).strip(),
        "profile_sha256":hashlib.sha256(profile.read_bytes()).hexdigest(),"cases":[]}
    with Serving(args.out/"fresh.db") as server:
        recorder=Recorder(server.endpoint,service=cfg["service"],capture_fields={"gap","previous_lane","previous_frame","frame","noise","output"})
        injector=Injector(recorder,cfg["probes"])
        sys.path.insert(0,str(root/"fixtures"))
        injector.install()
        app=importlib.import_module("lane_target")
        scenario=0
        for seed in spec["evaluation_seeds"]:
            for variant in spec["variants"]:
                base=1792540800000000000+scenario*60_000_000_000
                name=f"lane-01-{seed}-{variant}"
                manifest=[]
                started=time.perf_counter()
                for index,row in enumerate(rows(seed,variant,spec)):
                    time.sleep(max(0,started+index/60-time.perf_counter()))
                    n=1000+index
                    trace=hashlib.sha256(f"lane-01/{seed}/{variant}/{index}".encode()).hexdigest()[:32]
                    with recorder.run("lane_target.decide",run_id=trace,start_ns=base+index*16_666_667,
                                      attributes={"signalscout.session.id":name+f"-session-{index//20}"}):
                        output=app.decide(row["gap"],row["previous_lane"],n-row["age"],n,row["noise"])
                        run=recorder._active.get().trace_id
                    value="problem" if output["lane"]!=row["expected"] else "reference"
                    manifest.append({"run_id":run,"value":value,"age":row["age"],"frame":n,"output":output})
                assert recorder.flush(10), "delivery did not drain"
                for entry in manifest:
                    label={"id":"annotation:"+entry["run_id"]+"/oracle-1","run_id":entry["run_id"],"dimension":name,"value":entry["value"],
                        "source":"synthetic-oracle-v1","reviewer":"synthetic-reviewer","synthetic":True,"rubric_id":"lane-01-expected-right","rubric_version":"1",
                        "created_at":base+60_000_000_000,"evidence_ids":[]}
                    post(server.endpoint,"/api/annotations",label)
                selected={}
                for group in ("problem","reference"):
                    command=[str(server.binary),"cohort","create",name+"-"+group,"--endpoint",server.endpoint,"--profile",str(profile),
                        "--from",utc(base),"--to",utc(base+60_000_000_000),"--limit","200","--dimension",name,"--value",group]
                    selected[group]=json.loads(subprocess.check_output(command))
                command=[str(server.binary),"--db",str(server.database),"explain-cohort",selected["problem"]["id"],selected["reference"]["id"],"--json"]
                result=json.loads(subprocess.check_output(command))
                assert result["problem_runs"]==result["reference_runs"]==100
                assert result["status"]=="ok"
                ranked=[d for d in result["differences"] if d["status"]=="ok" and d["score"]>0]
                ranks={d["feature"]:i+1 for i,d in enumerate(ranked)}
                mechanisms=["previous_age","gap"]
                counts=result["invariant_counts"]
                case={"seed":seed,"variant":variant,"ranks":{v["feature"]:ranks.get(v["feature"]) for v in spec["labels"]},
                    "top_k_mechanism_recall":{str(k):sum(ranks.get(v,999)<=k for v in mechanisms)/len(mechanisms) for k in (1,3,5)},
                    "invariant_counts":counts,"capture":recorder.snapshot(),"manifest":manifest,"result":result}
                # Follow native evidence URLs, including event markers, over real HTTP.
                for d in ranked[:5]:
                    for link in d["evidence_urls"]:
                        path,anchor=link.split("#")
                        with urllib.request.urlopen(server.endpoint+path,timeout=20) as response:
                            assert ('id="'+anchor+'"') in response.read().decode(),link
                report["cases"].append(case)
                (args.out/"report.json").write_text(json.dumps(report,indent=2)+"\n",encoding="utf-8")
                print(seed,variant,case["ranks"],counts,flush=True)
                scenario+=1
        injector.uninstall()
        recorder.close(10)
        assert target.read_bytes()==before,"target source changed"
        report["instrumentation"]=count(target,snapshot,repo/"benchmark/problems/lane-01/signalscout.yaml")
        report["doctor"]=injector.doctor()
        counters=report["doctor"]["counters"]
        assert not any(value for key,value in counters.items() if key.startswith("drop") or key=="export_failed"),counters
        report["expected_decisions"]=3000
        report["captured_decisions"]=counters["serialized"]
        assert report["captured_decisions"]==3000
        (args.out/"report.json").write_text(json.dumps(report,indent=2)+"\n",encoding="utf-8")


if __name__=="__main__":
    main()
