# Synthetic false-positive control: never executed by the scanner.
LOOKUP = {"one": 1, "two": 2}

def read_only(key):
    return LOOKUP[key]
