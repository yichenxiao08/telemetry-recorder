"""Native process support for the synthetic spike, not a production recorder CLI."""
import socket
import subprocess
import time
import urllib.request
from pathlib import Path


class Serving:
    def __init__(self, database):
        self.database = Path(database)
        self.repo = Path(__file__).resolve().parents[2]

    def __enter__(self):
        self.database.parent.mkdir(parents=True, exist_ok=True)
        if self.database.exists():
            raise ValueError("fresh database required")
        with socket.socket() as listener:
            listener.bind(("127.0.0.1", 0))
            self.port = listener.getsockname()[1]
        self.endpoint = f"http://127.0.0.1:{self.port}"
        self.binary = self.repo/"bin"/"signalscout.exe"
        self.log = open(str(self.database)+".log", "w", encoding="utf-8")
        self.process = subprocess.Popen([str(self.binary), "--db", str(self.database), "serve",
            "--listen", f"127.0.0.1:{self.port}", "--retention", "0"],
            stdout=self.log, stderr=self.log)
        deadline = time.monotonic()+15
        while True:
            try:
                with urllib.request.urlopen(self.endpoint+"/runs", timeout=.1):
                    return self
            except OSError:
                if time.monotonic()>deadline or self.process.poll() is not None:
                    self.__exit__(None,None,None)
                    raise RuntimeError("fresh real intake did not start")
                time.sleep(.02)

    def show(self, run_id):
        import json
        return json.loads(subprocess.check_output([str(self.binary), "--db", str(self.database),
                                                  "runs", "show", run_id]))

    def __exit__(self, *args):
        self.process.terminate()
        self.process.wait(timeout=5)
        self.log.close()
