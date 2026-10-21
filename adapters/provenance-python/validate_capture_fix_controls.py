"""Repeat two unchanged Checkpoint A toy controls after serial validation."""
import argparse
from pathlib import Path

from capture_loss_conditions import trial


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--out", type=Path, required=True)
    args = parser.parse_args()
    args.out.mkdir(parents=True, exist_ok=False)
    trial(args.out, "post-commit-slow-ack", ack_delay=.65)
    trial(args.out, "burst-capacity-256", calls=1000, paced=False, ack_delay=.30)


if __name__ == "__main__":
    main()
