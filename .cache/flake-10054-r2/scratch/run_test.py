import sys,unittest
from pathlib import Path
sys.path.insert(0,'C:\\Users\\yiche\\telemetry-project\\adapters\\provenance-python')
import test_privacy_scratch
unittest.main(module=test_privacy_scratch,defaultTest="Privacy.test_real_probe_intake_and_log_raw_copies",verbosity=2)
