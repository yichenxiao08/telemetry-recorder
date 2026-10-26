"""Public Snake example: supplied scalar measurements, not inferred causality."""
import hashlib
import json
import os
from pathlib import Path
import sys
import time

WORK = json.loads(Path(os.environ["SIGNALSCOUT_REWARD_WORKLOAD"]).read_text(encoding="utf-8"))
sys.path.insert(0, WORK["project_root"])
import torch
import study.harness as harness
from value.agent.dqn import Agent

ACTIVE = None
TRANSITION = harness.transition


def step_observation(apple, death, repeated_configuration, actual_reward):
    """Explicit supplied measurements; pacing is an acknowledged recording cost."""
    time.sleep(WORK["pace_ms"] / 1000)


def observed_transition(env, action):
    state, row = TRANSITION(env, action)
    key = json.dumps([row["after"]["body"], row["after"]["direction"], row["after"]["apple"]])
    repeated = not row["death"] and key in ACTIVE["configurations"]
    if not row["death"]:
        ACTIVE["configurations"].add(key)
    reward = row["original_reward"] if WORK["policy"]["condition"] == "original" else row["variant_reward"]
    step_observation(bool(row["apple"]), bool(row["death"]), repeated, float(reward))
    return state, row


harness.transition = observed_transition


def episode_run(policy_id, training_seed, evaluation_seed, agent):
    global ACTIVE
    previous = harness.SPEC["evaluation"]["heldout_seeds"]
    ACTIVE = {"configurations": set()}
    harness.SPEC["evaluation"]["heldout_seeds"] = [evaluation_seed]
    try:
        return harness.evaluate(agent)[0]
    finally:
        ACTIVE = None
        harness.SPEC["evaluation"]["heldout_seeds"] = previous


def main():
    policy = WORK["policy"]
    checkpoint = Path(policy["checkpoint"])
    if hashlib.sha256(checkpoint.read_bytes()).hexdigest() != policy["checkpoint_sha256"]:
        raise ValueError("checkpoint changed")
    harness.setup_seed(policy["seed"])
    torch.set_num_threads(1)
    torch.use_deterministic_algorithms(True)
    saved = torch.load(checkpoint, map_location="cpu", weights_only=True)
    agent = Agent(11, 3)
    agent.training_network.load_state_dict(saved["training_network"])
    agent.frozen_network.load_state_dict(saved["frozen_network"])
    result_path = Path(policy["result"])
    if hashlib.sha256(result_path.read_bytes()).hexdigest() != policy["result_sha256"]:
        raise ValueError("saved evaluation changed")
    expected = json.loads(result_path.read_text(encoding="utf-8"))
    if expected["condition"] != policy["condition"] or expected["seed"] != policy["seed"]:
        raise ValueError("saved policy identity differs")
    by_seed = {row["seed"]: row for row in expected["evaluation"]}
    rows = []
    for seed in WORK["heldout_seeds"]:
        observed = episode_run(policy["id"], policy["seed"], seed, agent)
        if observed != by_seed[seed]:
            raise ValueError(f"frozen evaluation differs: {policy['id']}/{seed}")
        rows.append(observed)
    Path(WORK["parity_output"]).write_text(json.dumps({"policy": policy["id"], "episodes": rows,
        "all_exact_matches": True}, indent=2) + "\n", encoding="utf-8")
