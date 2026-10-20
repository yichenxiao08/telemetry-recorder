"""Entirely synthetic plain application. No Signal Scout imports or callbacks."""
import json


def choose_lane(gap, previous_lane, previous_frame, frame):
    lane = "right" if gap >= 0.5 else "left"
    maneuver = "keep" if lane == previous_lane else "change_" + lane
    return {"lane": lane, "maneuver": maneuver, "used_previous_frame": previous_frame}


def frames(seed=1, count=60):
    history = []
    for frame in range(count):
        previous_frame = frame - 1
        if frame >= 2 and frame % 7 == 0:
            previous_frame = frame - 2  # Synthetic stale-state defect.
        previous_lane = history[previous_frame]["lane"] if previous_frame >= 0 else "left"
        gap = 0.5 + (((frame * 17 + seed * 13) % 9) - 4) / 100
        history.append(choose_lane(gap, previous_lane, previous_frame, frame))
    return history


def main():
    print(json.dumps(frames()))


if __name__ == "__main__":
    main()
