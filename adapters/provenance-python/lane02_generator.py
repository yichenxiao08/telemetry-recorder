"""Executable translation of the synthetic lane-02 normative source. No I/O."""
import hashlib

DEVELOPMENT = (101, 103)
RESERVED = (710021, 710023, 710027, 710029, 710033)
VARIANTS = ("matched-negative-controls", "minority-stale-use", "benign-minority-tail",
            "within-range-subgroup", "moderate-decoy", "unrelated-nuisance")
ORIGIN_NS = 1767225600000000000


def guard(seed, evaluation_runner=False):
    if seed in RESERVED and evaluation_runner is not True:
        raise ValueError("reserved seed requires explicit evaluation-runner flag and owner GO")
    if seed not in DEVELOPMENT + RESERVED:
        raise ValueError("seed not pre-registered")


def digest(seed, variant, session, pair, field, arm, *, evaluation_runner=False):
    guard(seed, evaluation_runner)
    if variant not in VARIANTS:
        raise ValueError("unknown variant")
    message = f"lane-02/v1|{seed}|{variant}|{session}|{pair}|{field}|{arm}"
    return hashlib.sha256(message.encode("ascii")).digest()


def uniform(seed, variant, session, pair, field, arm, *, evaluation_runner=False):
    raw = digest(seed, variant, session, pair, field, arm,
                 evaluation_runner=evaluation_runner)
    return (int.from_bytes(raw[:8], "big") >> 11) / 2**53


def ordering(seed, variant, session, *, evaluation_runner=False):
    guard(seed, evaluation_runner)
    return sorted(((arm, pair) for arm in ("P", "R") for pair in range(20)),
                  key=lambda item: (digest(seed, variant, session, item[1], "order", item[0],
                                           evaluation_runner=evaluation_runner), item[0], item[1]))


def rows(seed, variant, *, evaluation_runner=False):
    # Validate BEFORE constructing any row; ordinary imports/tests cannot evaluate holdouts.
    guard(seed, evaluation_runner)
    if variant not in VARIANTS:
        raise ValueError("unknown variant")
    output = []
    for session in range(20):
        for slot, (arm, pair) in enumerate(ordering(seed, variant, session,
                                                   evaluation_runner=evaluation_runner)):
            def u(field, draw_arm):
                return uniform(seed, variant, session, pair, field, draw_arm,
                               evaluation_runner=evaluation_runner)
            previous = variant == "matched-negative-controls" or pair < 4
            gap = .50 + .04*u("gap", "shared") if previous else .65 + .30*u("gap", "shared")
            age = 2 if arm == "P" and previous else 1
            if variant == "within-range-subgroup" and arm == "R" and 4 <= pair < 10:
                age = 2
            noise = [-1+2*u(f"noise_{j}", arm if variant == "unrelated-nuisance" else "shared")
                     for j in range(7)]
            if variant == "benign-minority-tail" and pair < 4:
                noise[0] += 3.0
            if variant == "moderate-decoy" and arm == "P":
                noise[0] += .75
            frame = 100000 + 100*session + slot
            identity = f"lane-02/{seed}/{variant}/session/{session}/pair/{pair}/arm/{arm}"
            output.append({"run_id": hashlib.sha256(identity.encode("ascii")).hexdigest()[:32],
                           "session": f"lane-02/{seed}/{variant}/session/{session}",
                           "arm": arm, "pair": pair, "slot": slot, "session_index": session,
                           "inputs": {"gap": gap, "current_value": -1 if arm == "P" and not previous else 1,
                                      "history": {str(frame-2): -1, str(frame-1): 1},
                                      "previous_frame": frame-age, "frame": frame, "noise": noise}})
    return output
