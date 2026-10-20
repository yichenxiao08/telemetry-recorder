"""Fixed S0a five-trial probe cost harness; no engine/performance tuning."""
import argparse
import hashlib
import json
import os
import platform
import socket
import sys
import time
from pathlib import Path

from recorder import Recorder
from scout_probe import Injector, Probe
from spike_support import Serving

FIXTURES = Path(__file__).parent/"fixtures"
sys.path.insert(0,str(FIXTURES))
from benchmark_target import calculate


def quantile(values, q):
    x = sorted(v/1000 for v in values)
    index = q*(len(x)-1)
    lower = int(index)
    return x[lower] + (x[min(lower+1,len(x)-1)]-x[lower])*(index-lower)


def trial(endpoint, condition, mode, seed):
    fields = ("gap","previous_lane","previous_frame","frame","output")
    r = Recorder(endpoint, service="synthetic.cost", capture_fields=fields,
                 capacity=1 if condition=="saturated" else 256,
                 _start_worker=condition!="saturated")
    spec={"module":"benchmark_target","function":"calculate",
          "capture":{"args":list(fields[:-1]),"return":True}}
    owner=Injector(r,[spec])
    f=Probe(owner,calculate,spec).wrap()
    p=f.__signalscout_probe__
    times, entry, exit_times = [], [], []
    calls=1800 if mode=="60hz" else 10000
    started=time.perf_counter()
    for i in range(calls):
        if mode=="60hz":
            time.sleep(max(0,started+i/60-time.perf_counter()))
        t=time.perf_counter_ns()
        result=f(.51,0,seed,i)
        elapsed=time.perf_counter_ns()-t
        if result["value"] != .51+i:
            raise AssertionError("probe changed return")
        times.append(max(0,elapsed-p.last["body_ns"]))
        entry.append(p.last["context_enter_ns"])
        exit_times.append(p.last["context_exit_ns"])
    if mode=="60hz":
        time.sleep(max(0,started+30-time.perf_counter()))
    seconds=time.perf_counter()-started
    t=time.perf_counter()
    flushed=r.flush(.01 if condition=="saturated" else 10)
    flush_seconds=time.perf_counter()-t
    counts=r.snapshot()
    r.close(0)
    p50,p99=quantile(times,.5),quantile(times,.99)
    serialized=counts.get("serialized",0)
    return {"condition":condition,"mode":mode,"trial":seed,"calls":calls,"seconds":seconds,
            "p50_us":p50,"p99_us":p99,"budget_met":p50<=100 and p99<=250,
            "context_enter_p50_us":quantile(entry,.5),"context_enter_p99_us":quantile(entry,.99),
            "context_exit_p50_us":quantile(exit_times,.5),"context_exit_p99_us":quantile(exit_times,.99),
            "flush_seconds":flush_seconds,"drained":flushed,"counts":counts,
            "mean_envelope_bytes":counts.get("envelope_bytes",0)/serialized if serialized else None,
            "expected_saturation_flush_timeout":condition=="saturated"}


def main():
    parser=argparse.ArgumentParser()
    parser.add_argument("--out",required=True)
    parser.add_argument("--db",required=True)
    args=parser.parse_args()
    report={"synthetic":True,"python":sys.version,"platform":platform.platform(),
            "cpu":os.environ.get("PROCESSOR_IDENTIFIER",""),"trials":[],
            "method":"Caller perf_counter wall time minus target body only; includes automatic run context, argument snapshot, serialization, enqueue, context cleanup. Four scalar inputs and two scored alternatives returned by the benchmark target (not inferred). No network/capacity waits on host. 1800 calls/30s at 60 Hz and 10000 tight-loop calls, five trials each. Native real Go intake; no concurrent measurement workload. Saturated exporter deliberately not started, capacity1; circuit/no-op costs remain separate.",
            "payload_source_sha256":hashlib.sha256((FIXTURES/"benchmark_target.py").read_bytes()).hexdigest()}
    out=Path(args.out)
    out.parent.mkdir(parents=True,exist_ok=True)
    with Serving(args.db) as server, socket.socket() as down:
        down.bind(("127.0.0.1",0))  # Reserved but not listening: unreachable throughout.
        unreachable=f"http://127.0.0.1:{down.getsockname()[1]}"
        for condition in ("reachable","unreachable","saturated"):
            for mode in ("60hz","tight"):
                for seed in range(1,6):
                    result=trial(unreachable if condition=="unreachable" else server.endpoint,
                                 condition,mode,seed)
                    report["trials"].append(result)
                    out.write_text(json.dumps(report,indent=2)+"\n",encoding="utf-8")
                    print(condition,mode,seed,round(result["p50_us"],2),
                          round(result["p99_us"],2),result["budget_met"],flush=True)


if __name__=="__main__":
    main()
