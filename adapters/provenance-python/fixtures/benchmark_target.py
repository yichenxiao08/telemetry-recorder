"""Plain synthetic cost target. Two alternatives are RETURNED, not inferred."""
def calculate(gap, previous_lane, previous_frame, frame):
    return {"value": gap + frame,
            "alternatives": [{"option": "left", "score": 0.4},
                             {"option": "right", "score": 0.6}]}
