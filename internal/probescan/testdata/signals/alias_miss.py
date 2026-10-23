# Synthetic known miss: alias-mediated mutation defeats direct-name resolution.
ORIGINAL = {}
ALIAS = ORIGINAL

def missed(key, value):
    ALIAS[key] = value
    return ALIAS[key]
