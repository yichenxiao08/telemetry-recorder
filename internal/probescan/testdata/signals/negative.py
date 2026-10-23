# Synthetic negative controls; no candidate expected.
def read_only():
    mapping = {"one": 1}
    return mapping["one"]

def complete(a, b):
    local = {}
    local[(a, b)] = a + b
    return local[(a, b)]

class Local:
    def ordinary(self):
        self.count = 2
        return self.count

def immutable_default(items=()):
    return len(items)

def off_by_one(x):
    return x + 2  # Synthetic defect outside the catalogue.
