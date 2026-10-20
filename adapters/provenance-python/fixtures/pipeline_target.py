"""Entirely synthetic coaching-like pipeline; no recorder imports."""
import json
import time
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import requests


def context_select(history, limit):
    return history[-limit:]


def retrieve(question):
    return [] if question == "unrelated" else ["synthetic-doc-relevant", "synthetic-doc-decoy"]


def select_evidence(candidates):
    selected = candidates[:1]
    return selected


def draft(evidence, context, strict):
    return "decline" if not evidence or strict else "synthetic answer using " + ",".join(context)


def review(reply):
    return "repair" if reply == "decline" else "accept"


def repair(reply, evidence):
    return "synthetic repaired answer" if evidence else reply


def turn(index):
    history = ["synthetic-prior-question", "synthetic-relevant-detail", "synthetic-followup"]
    context = context_select(history, 1 if index % 4 == 2 else 3)
    evidence = select_evidence(retrieve("unrelated" if index % 4 == 0 else "related"))
    reply = draft(evidence, context, index % 4 in (1, 3))
    if review(reply) == "repair" and evidence and index % 4 == 1:
        reply = repair(reply, evidence)
        review(reply)
    return {"index": index, "status": 200, "reply": reply}


def main():
    class Handler(BaseHTTPRequestHandler):
        def do_GET(self):
            body = json.dumps(turn(int(self.path[1:]))).encode()
            self.send_response(200)
            self.end_headers()
            self.wfile.write(body)

        def log_message(self, *args):
            pass

    server = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
    worker = threading.Thread(target=server.serve_forever, daemon=True)
    worker.start()
    results = []
    try:
        for index in range(12):
            response = requests.get(f"http://127.0.0.1:{server.server_port}/{index}", timeout=2)
            assert response.status_code == 200
            results.append(response.json())
            time.sleep(.1)
    finally:
        server.shutdown()
        server.server_close()
        worker.join(timeout=2)
    print(json.dumps(results))
