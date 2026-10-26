"""Small 1D mobile robot model. All units and data are synthetic."""
import argparse
import json


def command(clearance, brake_at):
    """Return a forward command or stop at the specified clearance."""
    return 0.0 if clearance <= brake_at else 0.1


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--controller", choices=("reference", "regression"), required=True)
    args = parser.parse_args()
    threshold = 0.3 if args.controller == "reference" else 0.1
    clearance = 1.0
    steps = []
    for _ in range(12):
        speed = command(clearance, threshold)
        steps.append({"clearance": clearance, "speed": speed})
        clearance = round(clearance - speed, 6)
    print(json.dumps({"controller": args.controller, "steps": steps}))


if __name__ == "__main__":
    main()