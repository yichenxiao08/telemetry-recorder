# Synthetic syntax-only fixture. Never execute.
from functools import cache as memo
import functools as ft
STATE = {}
SHARED = []

def partial(a, b):
    STATE[a] = a + b
    return STATE[a]

@memo
def cached(a):
    return a + 1

@ft.lru_cache(maxsize=10)
def bounded_cache(a):
    return a

def defaulted(items=[]):
    return len(items)

def write_shared(value):
    SHARED.append(value)

def cold(value):
    return value * 2

for i in range(2):
    cold(i)

class Top:
    @staticmethod
    def method(items={}):
        return len(items)

async def asynchronous(items=[]):
    return items

def generator(items=[]):
    yield items

def outer():
    def inner(items=[]):
        return items
    return inner

class Property:
    @property
    def value(self):
        return 1

for i in range(2):
    unknown.step(i)
