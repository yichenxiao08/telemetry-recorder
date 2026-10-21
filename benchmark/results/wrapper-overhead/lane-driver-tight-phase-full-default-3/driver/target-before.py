"""Plain synthetic two-snapshot target. No instrumentation dependencies."""


def decide(gap, current_value, history, previous_frame, frame, noise):
    previous_used = gap < 0.55
    if previous_used:
        chosen_value = history[str(previous_frame)]
    else:
        chosen_value = current_value
    lane = "right" if chosen_value >= 0 else "left"
    return {"lane": lane, "chosen_value": chosen_value,
            "previous_used": previous_used, "previous_version": previous_frame,
            "frame": frame}
