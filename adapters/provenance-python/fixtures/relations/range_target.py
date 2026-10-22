"""Synthetic admission/bounds fixture, with plain object and sequence returns."""
import time


def arguments(frame):
    return frame - 3, (11 if frame % 8 == 0 else 0), 10, frame % 3 != 0


def admit(value, low, high, enabled):
    accepted = enabled and low <= value <= high
    if value == 2:  # A deliberately authored boolean mismatch at frame five.
        accepted = not accepted
    return {"accepted": accepted, "clipped": min(max(value, low), high),
            "limits": (low, high)}


def emit():
    for frame in range(16):
        admit(*arguments(frame))
        time.sleep(0.005)
