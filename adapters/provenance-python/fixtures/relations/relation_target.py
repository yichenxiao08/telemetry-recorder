"""Synthetic arithmetic fixture. Plain application; no Signal Scout imports."""

import argparse
import time
import range_target


class Calculator:
    def __init__(self):
        self.score = 0

    def calculate(self, x, y, frame, flag, enabled, private=7):
        # Four of 40 frames misreport the absolute difference. The flag is
        # balanced within both the affected and unaffected groups.
        state = (x, x + 1, x + 2, y)
        reported = abs(y - x) + (1 if frame % 10 == 0 else 0)
        return state, reported, flag, x <= y


def run_case(frame):
    return Calculator().calculate(frame, frame + 3, frame, frame % 4 < 2,
                                  frame % 7 != 0)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cases", type=int, default=40)
    args = parser.parse_args()
    for frame in range(args.cases):
        run_case(frame)
        time.sleep(0.005)
    range_target.emit()


if __name__ == "__main__":
    main()
