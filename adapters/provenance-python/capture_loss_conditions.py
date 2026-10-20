"""Toy transport controls. Configuration comparisons are experiments, not fixes."""
import argparse
import hashlib
import http.server
import json
import os
from pathlib import Path
import socket
import subprocess
import sys
import threading
import time
import urllib.request

from recorder import Recorder
from scout_probe import Injector, Probe
from spike_support import Serving

sys.path.insert(0, str(Path(__file__).parent/"fixtures"))
from benchmark_target import calculate


def quantile(values, q):
    if not values:
        return None
    ordered = sorted(values)
    at = (len(ordered)-1)*q
    low = int(at)
    return ordered[low]+(ordered[min(low+1, len(ordered)-1)]-ordered[low])*(at-low)


def load_worker(path, io):
    end = time.monotonic()+180
    if io:
        # A fixed, bounded 8 MiB file in this experiment's fresh directory.
        block = b"synthetic-load"*4096
        with Path(path).open("w+b") as stream:
            while time.monotonic() < end:
                stream.seek(0)
                for _ in range(128):
                    stream.write(block)
                stream.flush()
                os.fsync(stream.fileno())
    else:
        value = b"synthetic-background"
        while time.monotonic() < end:
            value = hashlib.sha256(value).digest()


def trial(out, name, *, calls=1800, paced=True, capacity=256, ack_delay=0, loaded=False, down=False):
    directory = out/name
    directory.mkdir()
    loads, relay_events, caller_events, worker_batches = [], [], [], []
    samples_stopped = threading.Event()
    with Serving(directory/"fresh.db") as real:
        class Handler(http.server.BaseHTTPRequestHandler):
            def log_message(self, *a):
                pass
            def do_POST(self):
                begin = time.time_ns()
                body = self.rfile.read(int(self.headers["Content-Length"]))
                entry = {"started_unix_ns": begin}
                # Bound all retained samples; no body/header content is recorded.
                if len(relay_events) < 2000:
                    relay_events.append(entry)
                try:
                    request = urllib.request.Request(real.endpoint+"/v1/traces", body,
                                                      {"Content-Type": "application/x-protobuf"})
                    with urllib.request.urlopen(request, timeout=10) as response:
                        payload, status = response.read(), response.status
                    entry.update(upstream_ack_unix_ns=time.time_ns(), status=status)
                    if ack_delay and entry is relay_events[0]:
                        time.sleep(ack_delay)  # Delay only after the real commit acknowledgement.
                    entry["reply_unix_ns"] = time.time_ns()
                    self.send_response(status)
                    self.end_headers()
                    self.wfile.write(payload)
                except (BrokenPipeError, ConnectionResetError):
                    entry["client_disconnected"] = True
        relay = http.server.ThreadingHTTPServer(("127.0.0.1", 0), Handler)
        serving = threading.Thread(target=relay.serve_forever, daemon=True)
        serving.start()
        closed = socket.socket()
        closed.bind(("127.0.0.1", 0))
        endpoint = f"http://127.0.0.1:{closed.getsockname()[1] if down else relay.server_port}"
        r = Recorder(endpoint, service="synthetic.capture-loss", capacity=capacity,
                     capture_fields={"gap", "previous_lane", "previous_frame", "frame", "output"})
        def sample_batches():
            seen = set()
            while not samples_stopped.is_set():
                recent = r.snapshot()["diagnostics"]["last_worker_batches"]
                for batch in recent:
                    key = (batch["attempt_unix_ns"], batch["elapsed_ns"])
                    if key not in seen and len(worker_batches) < 4096:
                        worker_batches.append(batch)
                    seen.add(key)
                seen.intersection_update((b["attempt_unix_ns"], b["elapsed_ns"]) for b in recent)
                samples_stopped.wait(.01)
        sampler = threading.Thread(target=sample_batches, daemon=True)
        sampler.start()
        spec = {"module": "benchmark_target", "function": "calculate",
                "capture": {"args": ["gap", "previous_lane", "previous_frame", "frame"], "return": True}}
        probe = Probe(Injector(r, [spec]), calculate, spec)
        f = probe.wrap()
        try:
            if loaded:
                for index in range(3):
                    loads.append(subprocess.Popen([sys.executable, __file__, "--load-worker",
                        str(directory/f"load-{index}.bin"), *( ["--io"] if index == 2 else [])]))
            start_ns, start = time.time_ns(), time.perf_counter()
            for index in range(calls):
                if paced:
                    time.sleep(max(0, start+index/60-time.perf_counter()))
                before = time.perf_counter_ns()
                result = f(.51, 0, 101, index)
                elapsed = time.perf_counter_ns()-before
                if result["value"] != .51+index:
                    raise AssertionError("host result changed")
                caller_events.append({"unix_ns": time.time_ns(), "wrapper_ns": max(0, elapsed-probe.last["body_ns"]),
                                      "context_enter_ns": probe.last["context_enter_ns"],
                                      "context_exit_ns": probe.last["context_exit_ns"]})
            wall = time.perf_counter()-start
            begin = time.perf_counter_ns()
            drained = r.flush(10)
            flush_ns = time.perf_counter_ns()-begin
            snapshot = r.snapshot()
            # Drain the final bounded source ring without dropping early stalls.
            samples_stopped.set()
            sampler.join(5)
            seen = {(b["attempt_unix_ns"], b["elapsed_ns"]) for b in worker_batches}
            for batch in snapshot["diagnostics"]["last_worker_batches"]:
                if (batch["attempt_unix_ns"], batch["elapsed_ns"]) not in seen:
                    worker_batches.append(batch)
            report = {"synthetic_only": True, "condition": name, "calls": calls, "paced": paced,
                      "capacity": capacity, "timeout": r.timeout, "cooldown": r.cooldown,
                      "ack_delay_seconds_first_request_only": ack_delay,
                      "background": "two CPU processes + bounded-file fsync process" if loaded else "no added load",
                      "server_pid": real.process.pid, "driver_pid": os.getpid(),
                      "started_unix_ns": start_ns, "wall_seconds": wall, "achieved_calls_per_second": calls/wall,
                      "drained": drained, "flush_ns": flush_ns, "snapshot": snapshot,
                      "wrapper_p50_us": quantile([e["wrapper_ns"]/1000 for e in caller_events], .5),
                      "wrapper_p99_us": quantile([e["wrapper_ns"]/1000 for e in caller_events], .99),
                      "caller_events": caller_events, "relay_events": relay_events,
                      "worker_batches": worker_batches, "worker_batch_capture_limit": 4096,
                      "sampler_stopped": not sampler.is_alive()}
            (directory/"report.json").write_text(json.dumps(report, indent=2)+"\n")
            print(name, snapshot.get("drop_cooldown", 0), snapshot.get("drop_full", 0),
                  snapshot.get("export_failed", 0), round(calls/wall, 3), flush=True)
        finally:
            r.close(0)
            samples_stopped.set()
            sampler.join(5)
            for load in loads:
                load.terminate()
                load.wait(timeout=5)
            closed.close()
            relay.shutdown()
            relay.server_close()
            serving.join(5)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--out", type=Path)
    parser.add_argument("--load-worker", type=Path)
    parser.add_argument("--io", action="store_true")
    args = parser.parse_args()
    if args.load_worker:
        load_worker(args.load_worker, args.io)
        return
    if args.out is None:
        parser.error("--out required")
    args.out.mkdir(parents=True, exist_ok=False)
    trial(args.out, "real-fast")
    trial(args.out, "real-controlled-load", loaded=True)
    trial(args.out, "post-commit-slow-ack", ack_delay=.65)
    trial(args.out, "burst-capacity-256", calls=1000, paced=False, ack_delay=.30)
    trial(args.out, "burst-capacity-4096", calls=1000, paced=False, capacity=4096, ack_delay=.30)
    trial(args.out, "unreachable", down=True)


if __name__ == "__main__":
    main()
