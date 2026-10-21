"""Fresh synthetic RL fixture; no external data or numeric dependencies."""
from collections import namedtuple

Step = namedtuple("Step", "state reward done")


class Environment:
    def __init__(self):
        self.state = [0]
        self.invocations = 0

    def step(self, action):
        self.state[0] += action
        return self.state, float(action), self.state[0] >= 10

    def named(self, action):
        return Step(*self.step(action))

    def echo(self, value):
        return value

    def raising(self, error):
        self.invocations += 1
        raise error


def loop(count=100):
    env = Environment()
    for _ in range(count):
        env.step(1)


def main():
    loop()


def invalid_main():
    Environment().echo((object(), 1))
