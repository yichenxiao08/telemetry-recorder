"""Synthetic lane-01 target: plain Python, no instrumentation imports."""


def decide(gap, previous_lane, previous_frame, frame, noise):
    use_previous = gap < .55
    lane = previous_lane if use_previous else "right"
    return {"lane": lane, "previous_used": use_previous,
            "previous_version": previous_frame, "frame": frame}
