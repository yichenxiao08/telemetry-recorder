"""New entirely synthetic plain fixture. No Signal Scout imports."""
import argparse
import json
import sys


def decision(value):
    return value


def payload(a, b, c, d):
    return {"choice": a + b + c + d, "alternatives": [{"option": "left", "score": a}, {"option": "right", "score": b}]}


class Environment:
    family = "synthetic"

    def __init__(self):
        self.score, self.steps, self.state = 0, 0, []

    def reset(self, seed):
        self.score, self.steps, self.state = seed, 0, []
        return self.state

    def step(self, action):
        self.steps += 1
        self.score += action
        self.state.append(action)
        return self.state

    def reward(self):
        return self.score

    def payload(self, a, b, c, d):
        return {"choice": a + b + c + d, "alternatives": [{"option": "left", "score": a}, {"option": "right", "score": b}]}

    def raising(self, error):
        raise error

    @classmethod
    def identity(cls, value):
        return value

    @staticmethod
    def policy(value):
        return value

    @property
    def unsafe(self):
        raise AssertionError("property must never run during attachment/capture")

    def generating(self):
        yield 1

    async def asynchronous(self):
        return 1


class Base:
    def step(self, value):
        return value + 1

    def raising(self, error):
        self.invocations = self.__dict__.get("invocations", 0) + 1
        raise error


class Child(Base):
    def step(self, value):
        return super().step(value) + 1

    def raising(self, error):
        return super().raising(error)


class Opaque:
    def __get__(self, instance, owner):
        raise AssertionError("descriptor must not execute during attachment")


class Unsupported:
    opaque = Opaque()


class CustomAccess:
    def __getattribute__(self, name):
        return object.__getattribute__(self, name)

    def step(self, value):
        return value


def snapshot_argv():
    return list(sys.argv)


def argv_main():
    print(json.dumps(snapshot_argv()))


def unused_main():
    print("synthetic target ran without calling configured probe")


def zero_exit_main():
    raise SystemExit(0)


def main():
    parser = argparse.ArgumentParser(allow_abbrev=False)
    commands = parser.add_subparsers(dest="command", required=True)
    run = commands.add_parser("run", allow_abbrev=False)
    run.add_argument("--episodes", type=int, required=True)
    run.add_argument("--seed", type=int, default=1)
    args = parser.parse_args()
    environment = Environment()
    rewards = []
    for _ in range(args.episodes):
        environment.reset(args.seed)
        for value in (1, 2, 3):
            environment.step(Environment.policy(value))
        rewards.append(environment.reward())
    print(json.dumps({"argv": list(sys.argv), "rewards": rewards}))


if __name__ == "__main__":
    main()
