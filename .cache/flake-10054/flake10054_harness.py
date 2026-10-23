"""Synthetic wrapper to run exactly the privacy integration test under scout_probe."""
from pathlib import Path
from types import SimpleNamespace
import socket
import sys
import urllib.error as urllib_error
import urllib.request as urllib_request

ADAPTER = Path(r"C:\Users\yiche\telemetry-project\adapters\provenance-python")
TESTS = ADAPTER / "tests"
sys.path.insert(0, str(ADAPTER))
sys.path.insert(0, str(TESTS))

import test_privacy

_ORIGINAL_URLOPEN = urllib_request.urlopen


def test_urlopen(url, data=None, timeout=socket._GLOBAL_DEFAULT_TIMEOUT, *, context=None):
    """Observe only test-owned HTTP calls; leave recorder exports untouched."""
    return _ORIGINAL_URLOPEN(url, data=data, timeout=timeout, context=context)


# The test receives a private urllib facade so only its calls use test_urlopen.
test_privacy.urllib = SimpleNamespace(
    request=SimpleNamespace(Request=urllib_request.Request, urlopen=test_urlopen),
    error=urllib_error,
)


def main():
    # Refresh after the import hook replaces this module's test_urlopen binding.
    test_privacy.urllib.request.urlopen = test_urlopen
    if test_privacy.urllib.request.urlopen is not test_urlopen:
        raise RuntimeError("test-local urlopen probe was not bound")
    case = test_privacy.Privacy("test_real_probe_intake_and_log_raw_copies")
    case.test_real_probe_intake_and_log_raw_copies()
    print("SCOUT_TARGET_TESTS status=PASS count=1")
