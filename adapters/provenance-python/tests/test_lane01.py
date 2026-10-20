"""Development seeds only; evaluation outcomes must remain unseen until freeze."""
import importlib
import sys
import unittest
from pathlib import Path
import yaml
from lane01 import rows


class LaneDevelopment(unittest.TestCase):
    def test_development_generation_and_noise_independence(self):
        repo=Path(__file__).resolve().parents[3]
        spec=yaml.safe_load((repo/"benchmark/problems/lane-01/problem.yaml").read_text())
        sys.path.insert(0,str(Path(__file__).resolve().parents[1]/"fixtures"))
        app=importlib.import_module("lane_target")
        for seed in spec["development_seeds"]:
            for variant in spec["variants"]:
                data=rows(seed,variant,spec)
                self.assertEqual(data,rows(seed,variant,spec))
                self.assertEqual(len(data),200)
                for row in data[:5]:
                    a=app.decide(row["gap"],row["previous_lane"],8,10,row["noise"])
                    b=app.decide(row["gap"],row["previous_lane"],8,10,[999]*7)
                    self.assertEqual(a,b)
