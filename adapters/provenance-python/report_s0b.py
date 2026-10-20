"""Extract manually reviewable examples from synthetic replay evidence."""
import argparse
import json
from pathlib import Path


def examples(snapshot):
    decisions=[]
    for row in snapshot["signals"]:
        signal=json.loads(row["data"])
        if signal.get("attrs",{}).get("signalscout.record.kind")=="decision":
            envelope=json.loads(signal["attrs"]["signalscout.record.json"])
            decisions.append({"native_id":signal["id"],"run_id":row["run_id"],"envelope":envelope})
    frames={}
    turns={}
    for item in decisions:
        env=item["envelope"]
        inputs={v["name"]:v.get("inline") for v in env["inputs"]}
        if env["step"]["id"]=="frame_target.choose_lane":
            frames[inputs["frame"]]=item
        if env["step"]["id"]=="pipeline_target.turn":
            turns[inputs["index"]]=item
    def stages(index):
        run_id=turns[index]["run_id"]
        return [d for d in decisions if d["run_id"]==run_id]
    runs=snapshot["runs"]
    c_fail=next(r for r in runs if r["workload"]=="POST /checkout" and r["outcome"]=="fail")
    c_reference=next(r for r in runs if r["workload"]=="POST /checkout" and str(json.loads(r["config"]).get("DB_POOL_SIZE"))=="20")
    return {"C_before_after":{"failing":c_fail,"reference":c_reference},
        "frame_absence":frames[2],"frame_stale_previous":frames[42],
        "frame_actual_previous":frames[41],"stale_version_without_lane_value_change":frames[7],
        "near_threshold_before":frames[4],"near_threshold_after":frames[5],
        "legitimate_decline_stages":stages(0),"successful_repair_stages":stages(1),
        "deliberate_truncation_stages":stages(2),"unrepaired_decline_stages":stages(3),
        "optional_semantics":[d for d in decisions if d["envelope"]["step"]["id"]=="selection-semantics"]}


if __name__=="__main__":
    parser=argparse.ArgumentParser()
    parser.add_argument("--input",type=Path,required=True)
    parser.add_argument("--out",type=Path,required=True)
    args=parser.parse_args()
    args.out.write_text(json.dumps(examples(json.loads(args.input.read_text())),indent=2)+"\n",encoding="utf-8")
