"""Read-only regression of existing development reports. No data generation."""
import argparse
import hashlib
import json
import subprocess
from pathlib import Path


def digest(value):
    return hashlib.sha256(value).hexdigest()


def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(",", ":")).encode()


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--reference", type=Path, required=True)
    parser.add_argument("--out", type=Path, required=True)
    args = parser.parse_args()
    root = Path(__file__).resolve().parents[2]
    binary = root/"bin/signalscout.exe"
    saved = json.loads((args.reference/"report.json").read_text())
    records = []
    for case in saved["cases"]:
        value = case["result"]
        command = [str(binary), "--db", str(args.reference/"fresh.db"), "explain-cohort",
                   value["problem"]["id"], value["reference"]["id"]]
        actual = subprocess.check_output(command+["--json"])
        expected_json, actual_json = canonical(value), canonical(json.loads(actual))
        expected_text = (args.reference/f'lane-02-{case["seed"]}-{case["variant"]}.txt').read_bytes()
        actual_text = subprocess.check_output(command)
        records.append({"seed": case["seed"], "variant": case["variant"],
                        "full_json_bytes_equal": expected_json == actual_json,
                        "text_bytes_equal": expected_text == actual_text,
                        "expected_json_sha256": digest(expected_json), "actual_json_sha256": digest(actual_json),
                        "expected_text_sha256": digest(expected_text), "actual_text_sha256": digest(actual_text)})
    args.out.parent.mkdir(parents=True, exist_ok=True)
    with args.out.open("x", encoding="utf-8") as stream:
        json.dump({"method": "Existing development store, query-only engine path; full JSON canonical bytes including cohort creation times; original text CLI stdout bytes.",
                   "cases": records}, stream, indent=2)
        stream.write("\n")
    if not all(r["full_json_bytes_equal"] and r["text_bytes_equal"] for r in records):
        raise SystemExit("Legacy output changed; retained mismatches")
    print("12 existing development JSON/text reports identical")


if __name__ == "__main__":
    main()
