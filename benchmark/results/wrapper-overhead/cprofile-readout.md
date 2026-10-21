# Separate cProfile readout

One separate process per payload/mode; these are not five-trial budget measurements.
Cumulative children overlap their parents. Do not sum the whole table.
Raw profiles also contain background network activity; the caller-only wrapper is the denominator here.
Function fractions are diagnostic proxies, not promised optimization savings.

| Profile | Caller-only function | Calls | Self ms | Cumulative ms | Fraction of probe wrapper cumulative |
|---|---|---:|---:|---:|---:|
| fixed-60hz-cprofile-full-default-1 | scout_probe.py:wrapped | 1800 | 169.66 | 1366.03 | 100.0% |
| fixed-60hz-cprofile-full-default-1 | recorder.py:record_decision | 1800 | 151.54 | 739.68 | 54.1% |
| fixed-60hz-cprofile-full-default-1 | __init__.py:dumps | 9000 | 42.04 | 275.17 | 20.1% |
| fixed-60hz-cprofile-full-default-1 | recorder.py:bounded | 66600 | 164.69 | 232.70 | 17.0% |
| fixed-60hz-cprofile-full-default-1 | encoder.py:iterencode | 9000 | 177.25 | 177.39 | 13.0% |
| fixed-60hz-cprofile-full-default-1 | recorder.py:_enqueue | 3600 | 10.11 | 120.22 | 8.8% |
| fixed-60hz-cprofile-full-default-1 | recorder.py:__enter__ | 1800 | 32.53 | 118.33 | 8.7% |
| fixed-60hz-cprofile-full-default-1 | inspect.py:bind | 1800 | 7.17 | 103.72 | 7.6% |
| fixed-60hz-cprofile-full-default-1 | recorder.py:__exit__ | 1800 | 9.35 | 54.91 | 4.0% |
| fixed-60hz-cprofile-full-default-1 | inspect.py:apply_defaults | 1800 | 18.92 | 26.51 | 1.9% |
| fixed-60hz-cprofile-full-default-1 | recorder.py:_span | 3600 | 7.09 | 7.09 | 0.5% |
| fixed-tight-cprofile-full-default-1 | scout_probe.py:wrapped | 10001 | 136.63 | 1832.12 | 100.0% |
| fixed-tight-cprofile-full-default-1 | recorder.py:record_decision | 10001 | 183.74 | 1197.22 | 65.3% |
| fixed-tight-cprofile-full-default-1 | __init__.py:dumps | 50001 | 45.86 | 404.44 | 22.1% |
| fixed-tight-cprofile-full-default-1 | recorder.py:bounded | 370000 | 234.74 | 368.00 | 20.1% |
| fixed-tight-cprofile-full-default-1 | recorder.py:_enqueue | 20000 | 20.41 | 253.63 | 13.8% |
| fixed-tight-cprofile-full-default-1 | encoder.py:iterencode | 50001 | 168.74 | 168.89 | 9.2% |
| fixed-tight-cprofile-full-default-1 | recorder.py:__exit__ | 10000 | 13.64 | 131.70 | 7.2% |
| fixed-tight-cprofile-full-default-1 | inspect.py:bind | 10000 | 4.61 | 95.49 | 5.2% |
| fixed-tight-cprofile-full-default-1 | recorder.py:__enter__ | 10000 | 23.36 | 76.41 | 4.2% |
| fixed-tight-cprofile-full-default-1 | inspect.py:apply_defaults | 10000 | 18.88 | 26.46 | 1.4% |
| fixed-tight-cprofile-full-default-1 | recorder.py:_span | 20000 | 6.54 | 6.54 | 0.4% |
| lane-inputs-60hz-cprofile-full-default-1 | scout_probe.py:wrapped | 1800 | 165.26 | 1455.54 | 100.0% |
| lane-inputs-60hz-cprofile-full-default-1 | recorder.py:record_decision | 1800 | 166.46 | 809.78 | 55.6% |
| lane-inputs-60hz-cprofile-full-default-1 | __init__.py:dumps | 12600 | 46.12 | 314.64 | 21.6% |
| lane-inputs-60hz-cprofile-full-default-1 | recorder.py:bounded | 108000 | 222.13 | 303.89 | 20.9% |
| lane-inputs-60hz-cprofile-full-default-1 | encoder.py:iterencode | 12600 | 202.36 | 202.56 | 13.9% |
| lane-inputs-60hz-cprofile-full-default-1 | recorder.py:_enqueue | 3600 | 9.28 | 117.08 | 8.0% |
| lane-inputs-60hz-cprofile-full-default-1 | recorder.py:__enter__ | 1800 | 27.30 | 98.74 | 6.8% |
| lane-inputs-60hz-cprofile-full-default-1 | inspect.py:bind | 1800 | 6.17 | 95.81 | 6.6% |
| lane-inputs-60hz-cprofile-full-default-1 | recorder.py:__exit__ | 1800 | 9.35 | 55.14 | 3.8% |
| lane-inputs-60hz-cprofile-full-default-1 | inspect.py:apply_defaults | 1800 | 19.20 | 26.58 | 1.8% |
| lane-inputs-60hz-cprofile-full-default-1 | recorder.py:_span | 3600 | 6.25 | 6.25 | 0.4% |
| lane-inputs-tight-cprofile-full-default-1 | scout_probe.py:wrapped | 10001 | 156.99 | 2431.71 | 100.0% |
| lane-inputs-tight-cprofile-full-default-1 | recorder.py:record_decision | 10001 | 220.03 | 1515.56 | 62.3% |
| lane-inputs-tight-cprofile-full-default-1 | __init__.py:dumps | 70001 | 61.38 | 674.24 | 27.7% |
| lane-inputs-tight-cprofile-full-default-1 | recorder.py:bounded | 600000 | 348.01 | 477.56 | 19.6% |
| lane-inputs-tight-cprofile-full-default-1 | encoder.py:iterencode | 70001 | 234.47 | 234.66 | 9.6% |
| lane-inputs-tight-cprofile-full-default-1 | inspect.py:bind | 10000 | 4.78 | 105.72 | 4.3% |
| lane-inputs-tight-cprofile-full-default-1 | recorder.py:_enqueue | 20000 | 19.46 | 104.39 | 4.3% |
| lane-inputs-tight-cprofile-full-default-1 | recorder.py:__enter__ | 10000 | 23.82 | 77.38 | 3.2% |
| lane-inputs-tight-cprofile-full-default-1 | recorder.py:__exit__ | 10000 | 13.73 | 65.75 | 2.7% |
| lane-inputs-tight-cprofile-full-default-1 | inspect.py:apply_defaults | 10000 | 22.22 | 32.09 | 1.3% |
| lane-inputs-tight-cprofile-full-default-1 | recorder.py:_span | 20000 | 6.53 | 6.53 | 0.3% |
