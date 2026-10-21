# Lane-02 Stage 1 development results

Synthetic only; seeds 101/103. Reserved seeds ungenerated/unscored.
Both rankings are separate. Deterministic tail ranks are reproducibility checks, not diagnostic-performance evidence.

| Seed / variant | Median positives / mechanisms | Tail positives / mechanisms | Tail top 1 | Score/count mismatches |
|---|---|---|---|---|
| 101 / matched-negative-controls | 1 / {'previous_age': 1} | 1 / {'previous_age': 1} | previous_age | [] |
| 101 / minority-stale-use | 1 / {'previous_age': None, 'current_value': 1} | 2 / {'previous_age': 2, 'current_value': 1} | current_value | [] |
| 101 / benign-minority-tail | 1 / {'previous_age': None, 'current_value': 1} | 2 / {'previous_age': 2, 'current_value': 1} | current_value | [] |
| 101 / within-range-subgroup | 1 / {'previous_age': None, 'current_value': 1} | 1 / {'previous_age': None, 'current_value': 1} | current_value | [] |
| 101 / moderate-decoy | 2 / {'previous_age': None, 'current_value': 1} | 3 / {'previous_age': 2, 'current_value': 1} | current_value | [] |
| 101 / unrelated-nuisance | 1 / {'previous_age': None, 'current_value': 1} | 2 / {'previous_age': 2, 'current_value': 1} | current_value | [] |
| 103 / matched-negative-controls | 1 / {'previous_age': 1} | 1 / {'previous_age': 1} | previous_age | [] |
| 103 / minority-stale-use | 1 / {'previous_age': None, 'current_value': 1} | 2 / {'previous_age': 2, 'current_value': 1} | current_value | [] |
| 103 / benign-minority-tail | 1 / {'previous_age': None, 'current_value': 1} | 2 / {'previous_age': 2, 'current_value': 1} | current_value | [] |
| 103 / within-range-subgroup | 1 / {'previous_age': None, 'current_value': 1} | 1 / {'previous_age': None, 'current_value': 1} | current_value | [] |
| 103 / moderate-decoy | 2 / {'previous_age': None, 'current_value': 1} | 3 / {'previous_age': 2, 'current_value': 1} | current_value | [] |
| 103 / unrelated-nuisance | 2 / {'previous_age': None, 'current_value': 1} | 2 / {'previous_age': 2, 'current_value': 1} | current_value | [] |

## 101 / matched-negative-controls

Capture: `{'captured_args': 4800, 'serialized': 800, 'envelope_bytes': 1294501, 'attempted': 1600, 'accepted': 1600, 'network_requests': 688, 'exported': 1600}`. Wrapper/context timings ns: `{'wrapper_p50': 268100.0, 'wrapper_p99': 512635.99999999994, 'run_enter_p50': 27800.0, 'run_exit_p50': 9000.0}`.
Invariant counts by arm: `{'problem': {'violated': 400}, 'reference': {'satisfied': 400}}`.
Support-premise misses (scores unchanged): `[]`.

### legacy_median

Positive count 2; top 1 output.lane; recall@1/3/5 `{'1': 0.0, '3': 1.0, '5': 1.0}`.
Ranks/scores: `[{'feature': 'output.lane', 'score': 0.9950248756218905}, {'feature': 'previous_age', 'score': 0.2827635434326132}]`.
Ties including unranked zero observations: `[{'score': 0, 'features': ['current_value', 'gap', 'noise_0', 'noise_1', 'noise_2', 'noise_3', 'noise_4', 'noise_5', 'noise_6']}]`.
Nuisance above mechanism: `[]`.

### median

Positive count 1; top 1 previous_age; recall@1/3/5 `{'1': 1.0, '3': 1.0, '5': 1.0}`.
Ranks/scores: `[{'feature': 'previous_age', 'score': 0.2827635434326132}]`.
Ties including unranked zero observations: `[{'score': 0, 'features': ['current_value', 'gap', 'noise_0', 'noise_1', 'noise_2', 'noise_3', 'noise_4', 'noise_5', 'noise_6']}]`.
Nuisance above mechanism: `[]`.

### tail

Positive count 1; top 1 previous_age; recall@1/3/5 `{'1': 1.0, '3': 1.0, '5': 1.0}`.
Ranks/scores: `[{'feature': 'previous_age', 'score': 0.9950248756218905}]`.
Ties including unranked zero observations: `[{'score': 0, 'features': ['current_value', 'gap', 'noise_0', 'noise_1', 'noise_2', 'noise_3', 'noise_4', 'noise_5', 'noise_6']}]`.
Nuisance above mechanism: `[]`.

| Tail feature | Raw P outside / observed | Raw R outside / observed | Interval | Score |
|---|---|---|---|---|
| previous_age | 400/400 | 0/400 | [0.5, 1.5] | 0.995024876 |
| current_value | 0/400 | 0/400 | [0.5, 1.5] | 0 |
| gap | 0/400 | 0/400 | [0.497371918, 0.542813084] | 0 |
| noise_0 | 0/400 | 0/400 | [-1.39535463, 1.41976799] | 0 |
| noise_1 | 0/400 | 0/400 | [-1.40677279, 1.3812505] | 0 |
| noise_2 | 0/400 | 0/400 | [-1.43163565, 1.3876324] | 0 |
| noise_3 | 0/400 | 0/400 | [-1.42986994, 1.39974226] | 0 |
| noise_4 | 0/400 | 0/400 | [-1.41685561, 1.43252202] | 0 |
| noise_5 | 0/400 | 0/400 | [-1.41372838, 1.39421617] | 0 |
| noise_6 | 0/400 | 0/400 | [-1.42804409, 1.39837552] | 0 |

## 101 / minority-stale-use

Capture: `{'captured_args': 4800, 'serialized': 800, 'envelope_bytes': 1295711, 'attempted': 1600, 'accepted': 1600, 'network_requests': 651, 'exported': 1600}`. Wrapper/context timings ns: `{'wrapper_p50': 277650.0, 'wrapper_p99': 533523.0, 'run_enter_p50': 28250.0, 'run_exit_p50': 9500.0}`.
Invariant counts by arm: `{'problem': {'not_applicable': 320, 'violated': 80}, 'reference': {'not_applicable': 320, 'satisfied': 80}}`.
Support-premise misses (scores unchanged): `[]`.

### legacy_median

Positive count 2; top 1 output.lane; recall@1/3/5 `{'1': 0.0, '3': 0.5, '5': 0.5}`.
Ranks/scores: `[{'feature': 'output.lane', 'score': 0.9950248756218905}, {'feature': 'current_value', 'score': 0.4853724757908721}]`.
Ties including unranked zero observations: `[{'score': 0, 'features': ['gap', 'noise_0', 'noise_1', 'noise_2', 'noise_3', 'noise_4', 'noise_5', 'noise_6', 'previous_age']}]`.
Nuisance above mechanism: `[]`.

### median

Positive count 1; top 1 current_value; recall@1/3/5 `{'1': 0.5, '3': 0.5, '5': 0.5}`.
Ranks/scores: `[{'feature': 'current_value', 'score': 0.4853724757908721}]`.
Ties including unranked zero observations: `[{'score': 0, 'features': ['gap', 'noise_0', 'noise_1', 'noise_2', 'noise_3', 'noise_4', 'noise_5', 'noise_6', 'previous_age']}]`.
Nuisance above mechanism: `[]`.

### tail

Positive count 2; top 1 current_value; recall@1/3/5 `{'1': 0.5, '3': 1.0, '5': 1.0}`.
Ranks/scores: `[{'feature': 'current_value', 'score': 0.7960199004975124}, {'feature': 'previous_age', 'score': 0.19900497512437812}]`.
Ties including unranked zero observations: `[{'score': 0, 'features': ['gap', 'noise_0', 'noise_1', 'noise_2', 'noise_3', 'noise_4', 'noise_5', 'noise_6']}]`.
Nuisance above mechanism: `[]`.

| Tail feature | Raw P outside / observed | Raw R outside / observed | Interval | Score |
|---|---|---|---|---|
| current_value | 320/400 | 0/400 | [0.5, 1.5] | 0.7960199 |
| previous_age | 80/400 | 0/400 | [0.5, 1.5] | 0.199004975 |
| gap | 24/400 | 24/400 | [0.503687714, 0.940563001] | 0 |
| noise_0 | 0/400 | 0/400 | [-1.3612778, 1.37314377] | 0 |
| noise_1 | 0/400 | 0/400 | [-1.37240531, 1.44387919] | 0 |
| noise_2 | 0/400 | 0/400 | [-1.39800654, 1.40321825] | 0 |
| noise_3 | 0/400 | 0/400 | [-1.3799771, 1.36783752] | 0 |
| noise_4 | 0/400 | 0/400 | [-1.37648025, 1.41384633] | 0 |
| noise_5 | 0/400 | 0/400 | [-1.36884937, 1.39859629] | 0 |
| noise_6 | 0/400 | 0/400 | [-1.41553692, 1.41320832] | 0 |

## 101 / benign-minority-tail

Capture: `{'captured_args': 4800, 'serialized': 800, 'envelope_bytes': 1295357, 'attempted': 1600, 'accepted': 1600, 'network_requests': 670, 'exported': 1600}`. Wrapper/context timings ns: `{'wrapper_p50': 278350.0, 'wrapper_p99': 529834.0, 'run_enter_p50': 27100.0, 'run_exit_p50': 9600.0}`.
Invariant counts by arm: `{'problem': {'not_applicable': 320, 'violated': 80}, 'reference': {'not_applicable': 320, 'satisfied': 80}}`.
Support-premise misses (scores unchanged): `[{'feature': 'noise_0', 'problem_outside': 7, 'reference_outside': 7}]`.

### legacy_median

Positive count 2; top 1 output.lane; recall@1/3/5 `{'1': 0.0, '3': 0.5, '5': 0.5}`.
Ranks/scores: `[{'feature': 'output.lane', 'score': 0.9950248756218905}, {'feature': 'current_value', 'score': 0.4853724757908721}]`.
Ties including unranked zero observations: `[{'score': 0, 'features': ['gap', 'noise_0', 'noise_1', 'noise_2', 'noise_3', 'noise_4', 'noise_5', 'noise_6', 'previous_age']}]`.
Nuisance above mechanism: `[]`.

### median

Positive count 1; top 1 current_value; recall@1/3/5 `{'1': 0.5, '3': 0.5, '5': 0.5}`.
Ranks/scores: `[{'feature': 'current_value', 'score': 0.4853724757908721}]`.
Ties including unranked zero observations: `[{'score': 0, 'features': ['gap', 'noise_0', 'noise_1', 'noise_2', 'noise_3', 'noise_4', 'noise_5', 'noise_6', 'previous_age']}]`.
Nuisance above mechanism: `[]`.

### tail

Positive count 2; top 1 current_value; recall@1/3/5 `{'1': 0.5, '3': 1.0, '5': 1.0}`.
Ranks/scores: `[{'feature': 'current_value', 'score': 0.7960199004975124}, {'feature': 'previous_age', 'score': 0.19900497512437812}]`.
Ties including unranked zero observations: `[{'score': 0, 'features': ['gap', 'noise_0', 'noise_1', 'noise_2', 'noise_3', 'noise_4', 'noise_5', 'noise_6']}]`.
Nuisance above mechanism: `[]`.

| Tail feature | Raw P outside / observed | Raw R outside / observed | Interval | Score |
|---|---|---|---|---|
| current_value | 320/400 | 0/400 | [0.5, 1.5] | 0.7960199 |
| previous_age | 80/400 | 0/400 | [0.5, 1.5] | 0.199004975 |
| gap | 23/400 | 23/400 | [0.504674109, 0.93866937] | 0 |
| noise_0 | 7/400 | 7/400 | [-1.36764598, 3.7982302] | 0 |
| noise_1 | 0/400 | 0/400 | [-1.36749661, 1.42782661] | 0 |
| noise_2 | 0/400 | 0/400 | [-1.43496551, 1.39785935] | 0 |
| noise_3 | 0/400 | 0/400 | [-1.40770004, 1.38513438] | 0 |
| noise_4 | 0/400 | 0/400 | [-1.38248861, 1.36983979] | 0 |
| noise_5 | 0/400 | 0/400 | [-1.37896548, 1.39615298] | 0 |
| noise_6 | 0/400 | 0/400 | [-1.37216593, 1.41087105] | 0 |

## 101 / within-range-subgroup

Capture: `{'captured_args': 4800, 'serialized': 800, 'envelope_bytes': 1295641, 'attempted': 1600, 'accepted': 1600, 'network_requests': 657, 'exported': 1600}`. Wrapper/context timings ns: `{'wrapper_p50': 242600.0, 'wrapper_p99': 494845.9999999999, 'run_enter_p50': 26500.0, 'run_exit_p50': 8300.0}`.
Invariant counts by arm: `{'problem': {'not_applicable': 320, 'violated': 80}, 'reference': {'not_applicable': 320, 'satisfied': 80}}`.
Support-premise misses (scores unchanged): `[]`.

### legacy_median

Positive count 2; top 1 output.lane; recall@1/3/5 `{'1': 0.0, '3': 0.5, '5': 0.5}`.
Ranks/scores: `[{'feature': 'output.lane', 'score': 0.9950248756218905}, {'feature': 'current_value', 'score': 0.4853724757908721}]`.
Ties including unranked zero observations: `[{'score': 0, 'features': ['gap', 'noise_0', 'noise_1', 'noise_2', 'noise_3', 'noise_4', 'noise_5', 'noise_6', 'previous_age']}]`.
Nuisance above mechanism: `[]`.

### median

Positive count 1; top 1 current_value; recall@1/3/5 `{'1': 0.5, '3': 0.5, '5': 0.5}`.
Ranks/scores: `[{'feature': 'current_value', 'score': 0.4853724757908721}]`.
Ties including unranked zero observations: `[{'score': 0, 'features': ['gap', 'noise_0', 'noise_1', 'noise_2', 'noise_3', 'noise_4', 'noise_5', 'noise_6', 'previous_age']}]`.
Nuisance above mechanism: `[]`.

### tail

Positive count 1; top 1 current_value; recall@1/3/5 `{'1': 0.5, '3': 0.5, '5': 0.5}`.
Ranks/scores: `[{'feature': 'current_value', 'score': 0.7960199004975124}]`.
Ties including unranked zero observations: `[{'score': 0, 'features': ['gap', 'noise_0', 'noise_1', 'noise_2', 'noise_3', 'noise_4', 'noise_5', 'noise_6', 'previous_age']}]`.
Nuisance above mechanism: `[]`.

| Tail feature | Raw P outside / observed | Raw R outside / observed | Interval | Score |
|---|---|---|---|---|
| current_value | 320/400 | 0/400 | [0.5, 1.5] | 0.7960199 |
| gap | 21/400 | 21/400 | [0.501983219, 0.930379973] | 0 |
| noise_0 | 0/400 | 0/400 | [-1.40461472, 1.42421835] | 0 |
| noise_1 | 0/400 | 0/400 | [-1.40245278, 1.35992043] | 0 |
| noise_2 | 0/400 | 0/400 | [-1.38266673, 1.39315313] | 0 |
| noise_3 | 0/400 | 0/400 | [-1.40032107, 1.4214385] | 0 |
| noise_4 | 0/400 | 0/400 | [-1.39958397, 1.39084361] | 0 |
| noise_5 | 0/400 | 0/400 | [-1.40860813, 1.41168423] | 0 |
| noise_6 | 0/400 | 0/400 | [-1.40297091, 1.39574857] | 0 |
| previous_age | 0/400 | 0/400 | [0.5, 2.5] | 0 |

## 101 / moderate-decoy

Capture: `{'captured_args': 4800, 'serialized': 800, 'envelope_bytes': 1295460, 'attempted': 1600, 'accepted': 1600, 'network_requests': 672, 'exported': 1600}`. Wrapper/context timings ns: `{'wrapper_p50': 234150.0, 'wrapper_p99': 469142.99999999977, 'run_enter_p50': 27300.0, 'run_exit_p50': 8050.0}`.
Invariant counts by arm: `{'problem': {'not_applicable': 320, 'violated': 80}, 'reference': {'not_applicable': 320, 'satisfied': 80}}`.
Support-premise misses (scores unchanged): `[]`.

### legacy_median

Positive count 3; top 1 output.lane; recall@1/3/5 `{'1': 0.0, '3': 0.5, '5': 0.5}`.
Ranks/scores: `[{'feature': 'output.lane', 'score': 0.9950248756218905}, {'feature': 'current_value', 'score': 0.4853724757908721}, {'feature': 'noise_0', 'score': 0.1964381105559912}]`.
Ties including unranked zero observations: `[{'score': 0, 'features': ['gap', 'noise_1', 'noise_2', 'noise_3', 'noise_4', 'noise_5', 'noise_6', 'previous_age']}]`.
Nuisance above mechanism: `[{'nuisance': 'noise_0', 'mechanism': 'previous_age'}]`.

### median

Positive count 2; top 1 current_value; recall@1/3/5 `{'1': 0.5, '3': 0.5, '5': 0.5}`.
Ranks/scores: `[{'feature': 'current_value', 'score': 0.4853724757908721}, {'feature': 'noise_0', 'score': 0.1964381105559912}]`.
Ties including unranked zero observations: `[{'score': 0, 'features': ['gap', 'noise_1', 'noise_2', 'noise_3', 'noise_4', 'noise_5', 'noise_6', 'previous_age']}]`.
Nuisance above mechanism: `[{'nuisance': 'noise_0', 'mechanism': 'previous_age'}]`.

### tail

Positive count 3; top 1 current_value; recall@1/3/5 `{'1': 0.5, '3': 1.0, '5': 1.0}`.
Ranks/scores: `[{'feature': 'current_value', 'score': 0.7960199004975124}, {'feature': 'previous_age', 'score': 0.19900497512437812}, {'feature': 'noise_0', 'score': 0.14676616915422885}]`.
Ties including unranked zero observations: `[{'score': 0, 'features': ['gap', 'noise_1', 'noise_2', 'noise_3', 'noise_4', 'noise_5', 'noise_6']}]`.
Nuisance above mechanism: `[]`.

| Tail feature | Raw P outside / observed | Raw R outside / observed | Interval | Score |
|---|---|---|---|---|
| current_value | 320/400 | 0/400 | [0.5, 1.5] | 0.7960199 |
| previous_age | 80/400 | 0/400 | [0.5, 1.5] | 0.199004975 |
| noise_0 | 59/400 | 0/400 | [-1.38912233, 1.40503882] | 0.146766169 |
| gap | 27/400 | 27/400 | [0.507699364, 0.930101882] | 0 |
| noise_1 | 0/400 | 0/400 | [-1.42509592, 1.39783819] | 0 |
| noise_2 | 0/400 | 0/400 | [-1.36887051, 1.36512481] | 0 |
| noise_3 | 0/400 | 0/400 | [-1.39689569, 1.40582391] | 0 |
| noise_4 | 0/400 | 0/400 | [-1.39702609, 1.37660726] | 0 |
| noise_5 | 0/400 | 0/400 | [-1.43317686, 1.3628208] | 0 |
| noise_6 | 0/400 | 0/400 | [-1.36232003, 1.4296034] | 0 |

## 101 / unrelated-nuisance

Capture: `{'captured_args': 4800, 'serialized': 800, 'envelope_bytes': 1295554, 'attempted': 1600, 'accepted': 1600, 'network_requests': 633, 'exported': 1600}`. Wrapper/context timings ns: `{'wrapper_p50': 279700.0, 'wrapper_p99': 535609.0, 'run_enter_p50': 28050.0, 'run_exit_p50': 9750.0}`.
Invariant counts by arm: `{'problem': {'not_applicable': 320, 'violated': 80}, 'reference': {'not_applicable': 320, 'satisfied': 80}}`.
Support-premise misses (scores unchanged): `[]`.

### legacy_median

Positive count 2; top 1 output.lane; recall@1/3/5 `{'1': 0.0, '3': 0.5, '5': 0.5}`.
Ranks/scores: `[{'feature': 'output.lane', 'score': 0.9950248756218905}, {'feature': 'current_value', 'score': 0.4853724757908721}]`.
Ties including unranked zero observations: `[{'score': 0, 'features': ['gap', 'noise_0', 'noise_1', 'noise_2', 'noise_3', 'noise_4', 'noise_5', 'noise_6', 'previous_age']}]`.
Nuisance above mechanism: `[]`.

### median

Positive count 1; top 1 current_value; recall@1/3/5 `{'1': 0.5, '3': 0.5, '5': 0.5}`.
Ranks/scores: `[{'feature': 'current_value', 'score': 0.4853724757908721}]`.
Ties including unranked zero observations: `[{'score': 0, 'features': ['gap', 'noise_0', 'noise_1', 'noise_2', 'noise_3', 'noise_4', 'noise_5', 'noise_6', 'previous_age']}]`.
Nuisance above mechanism: `[]`.

### tail

Positive count 2; top 1 current_value; recall@1/3/5 `{'1': 0.5, '3': 1.0, '5': 1.0}`.
Ranks/scores: `[{'feature': 'current_value', 'score': 0.7960199004975124}, {'feature': 'previous_age', 'score': 0.19900497512437812}]`.
Ties including unranked zero observations: `[{'score': 0, 'features': ['gap', 'noise_0', 'noise_1', 'noise_2', 'noise_3', 'noise_4', 'noise_5', 'noise_6']}]`.
Nuisance above mechanism: `[]`.

| Tail feature | Raw P outside / observed | Raw R outside / observed | Interval | Score |
|---|---|---|---|---|
| current_value | 320/400 | 0/400 | [0.5, 1.5] | 0.7960199 |
| previous_age | 80/400 | 0/400 | [0.5, 1.5] | 0.199004975 |
| gap | 31/400 | 31/400 | [0.508206532, 0.928679389] | 0 |
| noise_0 | 0/400 | 0/400 | [-1.43975677, 1.36663651] | 0 |
| noise_1 | 0/400 | 0/400 | [-1.38461966, 1.40527185] | 0 |
| noise_2 | 0/400 | 0/400 | [-1.3809479, 1.39180719] | 0 |
| noise_3 | 0/400 | 0/400 | [-1.38825171, 1.37747181] | 0 |
| noise_4 | 0/400 | 0/400 | [-1.39401334, 1.36073867] | 0 |
| noise_5 | 0/400 | 0/400 | [-1.368308, 1.41429367] | 0 |
| noise_6 | 0/400 | 0/400 | [-1.380847, 1.38579121] | 0 |

## 103 / matched-negative-controls

Capture: `{'captured_args': 4800, 'serialized': 800, 'envelope_bytes': 1294615, 'attempted': 1600, 'accepted': 1600, 'network_requests': 617, 'exported': 1600}`. Wrapper/context timings ns: `{'wrapper_p50': 247750.0, 'wrapper_p99': 633872.9999999999, 'run_enter_p50': 26900.0, 'run_exit_p50': 8500.0}`.
Invariant counts by arm: `{'problem': {'violated': 400}, 'reference': {'satisfied': 400}}`.
Support-premise misses (scores unchanged): `[]`.

### legacy_median

Positive count 2; top 1 output.lane; recall@1/3/5 `{'1': 0.0, '3': 1.0, '5': 1.0}`.
Ranks/scores: `[{'feature': 'output.lane', 'score': 0.9950248756218905}, {'feature': 'previous_age', 'score': 0.2827635434326132}]`.
Ties including unranked zero observations: `[{'score': 0, 'features': ['current_value', 'gap', 'noise_0', 'noise_1', 'noise_2', 'noise_3', 'noise_4', 'noise_5', 'noise_6']}]`.
Nuisance above mechanism: `[]`.

### median

Positive count 1; top 1 previous_age; recall@1/3/5 `{'1': 1.0, '3': 1.0, '5': 1.0}`.
Ranks/scores: `[{'feature': 'previous_age', 'score': 0.2827635434326132}]`.
Ties including unranked zero observations: `[{'score': 0, 'features': ['current_value', 'gap', 'noise_0', 'noise_1', 'noise_2', 'noise_3', 'noise_4', 'noise_5', 'noise_6']}]`.
Nuisance above mechanism: `[]`.

### tail

Positive count 1; top 1 previous_age; recall@1/3/5 `{'1': 1.0, '3': 1.0, '5': 1.0}`.
Ranks/scores: `[{'feature': 'previous_age', 'score': 0.9950248756218905}]`.
Ties including unranked zero observations: `[{'score': 0, 'features': ['current_value', 'gap', 'noise_0', 'noise_1', 'noise_2', 'noise_3', 'noise_4', 'noise_5', 'noise_6']}]`.
Nuisance above mechanism: `[]`.

| Tail feature | Raw P outside / observed | Raw R outside / observed | Interval | Score |
|---|---|---|---|---|
| previous_age | 400/400 | 0/400 | [0.5, 1.5] | 0.995024876 |
| current_value | 0/400 | 0/400 | [0.5, 1.5] | 0 |
| gap | 0/400 | 0/400 | [0.496894167, 0.542462675] | 0 |
| noise_0 | 0/400 | 0/400 | [-1.37715413, 1.33129503] | 0 |
| noise_1 | 0/400 | 0/400 | [-1.37287793, 1.411062] | 0 |
| noise_2 | 0/400 | 0/400 | [-1.38617155, 1.4276214] | 0 |
| noise_3 | 0/400 | 0/400 | [-1.38691632, 1.41288477] | 0 |
| noise_4 | 0/400 | 0/400 | [-1.41813104, 1.40623806] | 0 |
| noise_5 | 0/400 | 0/400 | [-1.40661785, 1.40735368] | 0 |
| noise_6 | 0/400 | 0/400 | [-1.42332642, 1.43163587] | 0 |

## 103 / minority-stale-use

Capture: `{'captured_args': 4800, 'serialized': 800, 'envelope_bytes': 1295457, 'attempted': 1600, 'accepted': 1600, 'network_requests': 620, 'exported': 1600}`. Wrapper/context timings ns: `{'wrapper_p50': 264350.0, 'wrapper_p99': 531207.0, 'run_enter_p50': 29200.0, 'run_exit_p50': 9000.0}`.
Invariant counts by arm: `{'problem': {'not_applicable': 320, 'violated': 80}, 'reference': {'not_applicable': 320, 'satisfied': 80}}`.
Support-premise misses (scores unchanged): `[]`.

### legacy_median

Positive count 2; top 1 output.lane; recall@1/3/5 `{'1': 0.0, '3': 0.5, '5': 0.5}`.
Ranks/scores: `[{'feature': 'output.lane', 'score': 0.9950248756218905}, {'feature': 'current_value', 'score': 0.4853724757908721}]`.
Ties including unranked zero observations: `[{'score': 0, 'features': ['gap', 'noise_0', 'noise_1', 'noise_2', 'noise_3', 'noise_4', 'noise_5', 'noise_6', 'previous_age']}]`.
Nuisance above mechanism: `[]`.

### median

Positive count 1; top 1 current_value; recall@1/3/5 `{'1': 0.5, '3': 0.5, '5': 0.5}`.
Ranks/scores: `[{'feature': 'current_value', 'score': 0.4853724757908721}]`.
Ties including unranked zero observations: `[{'score': 0, 'features': ['gap', 'noise_0', 'noise_1', 'noise_2', 'noise_3', 'noise_4', 'noise_5', 'noise_6', 'previous_age']}]`.
Nuisance above mechanism: `[]`.

### tail

Positive count 2; top 1 current_value; recall@1/3/5 `{'1': 0.5, '3': 1.0, '5': 1.0}`.
Ranks/scores: `[{'feature': 'current_value', 'score': 0.7960199004975124}, {'feature': 'previous_age', 'score': 0.19900497512437812}]`.
Ties including unranked zero observations: `[{'score': 0, 'features': ['gap', 'noise_0', 'noise_1', 'noise_2', 'noise_3', 'noise_4', 'noise_5', 'noise_6']}]`.
Nuisance above mechanism: `[]`.

| Tail feature | Raw P outside / observed | Raw R outside / observed | Interval | Score |
|---|---|---|---|---|
| current_value | 320/400 | 0/400 | [0.5, 1.5] | 0.7960199 |
| previous_age | 80/400 | 0/400 | [0.5, 1.5] | 0.199004975 |
| gap | 22/400 | 22/400 | [0.503262919, 0.941954338] | 0 |
| noise_0 | 0/400 | 0/400 | [-1.4140212, 1.40209802] | 0 |
| noise_1 | 0/400 | 0/400 | [-1.43470833, 1.38969418] | 0 |
| noise_2 | 0/400 | 0/400 | [-1.40350206, 1.38374849] | 0 |
| noise_3 | 0/400 | 0/400 | [-1.39538655, 1.42223509] | 0 |
| noise_4 | 0/400 | 0/400 | [-1.34095758, 1.43135175] | 0 |
| noise_5 | 0/400 | 0/400 | [-1.41324559, 1.39617733] | 0 |
| noise_6 | 0/400 | 0/400 | [-1.38339592, 1.38250278] | 0 |

## 103 / benign-minority-tail

Capture: `{'captured_args': 4800, 'serialized': 800, 'envelope_bytes': 1295429, 'attempted': 1600, 'accepted': 1600, 'network_requests': 626, 'exported': 1600}`. Wrapper/context timings ns: `{'wrapper_p50': 247350.0, 'wrapper_p99': 500765.9999999998, 'run_enter_p50': 26800.0, 'run_exit_p50': 8700.0}`.
Invariant counts by arm: `{'problem': {'violated': 80, 'not_applicable': 320}, 'reference': {'not_applicable': 320, 'satisfied': 80}}`.
Support-premise misses (scores unchanged): `[]`.

### legacy_median

Positive count 2; top 1 output.lane; recall@1/3/5 `{'1': 0.0, '3': 0.5, '5': 0.5}`.
Ranks/scores: `[{'feature': 'output.lane', 'score': 0.9950248756218905}, {'feature': 'current_value', 'score': 0.4853724757908721}]`.
Ties including unranked zero observations: `[{'score': 0, 'features': ['gap', 'noise_0', 'noise_1', 'noise_2', 'noise_3', 'noise_4', 'noise_5', 'noise_6', 'previous_age']}]`.
Nuisance above mechanism: `[]`.

### median

Positive count 1; top 1 current_value; recall@1/3/5 `{'1': 0.5, '3': 0.5, '5': 0.5}`.
Ranks/scores: `[{'feature': 'current_value', 'score': 0.4853724757908721}]`.
Ties including unranked zero observations: `[{'score': 0, 'features': ['gap', 'noise_0', 'noise_1', 'noise_2', 'noise_3', 'noise_4', 'noise_5', 'noise_6', 'previous_age']}]`.
Nuisance above mechanism: `[]`.

### tail

Positive count 2; top 1 current_value; recall@1/3/5 `{'1': 0.5, '3': 1.0, '5': 1.0}`.
Ranks/scores: `[{'feature': 'current_value', 'score': 0.7960199004975124}, {'feature': 'previous_age', 'score': 0.19900497512437812}]`.
Ties including unranked zero observations: `[{'score': 0, 'features': ['gap', 'noise_0', 'noise_1', 'noise_2', 'noise_3', 'noise_4', 'noise_5', 'noise_6']}]`.
Nuisance above mechanism: `[]`.

| Tail feature | Raw P outside / observed | Raw R outside / observed | Interval | Score |
|---|---|---|---|---|
| current_value | 320/400 | 0/400 | [0.5, 1.5] | 0.7960199 |
| previous_age | 80/400 | 0/400 | [0.5, 1.5] | 0.199004975 |
| gap | 25/400 | 25/400 | [0.50635888, 0.934543521] | 0 |
| noise_0 | 0/400 | 0/400 | [-1.31525484, 4.05661797] | 0 |
| noise_1 | 0/400 | 0/400 | [-1.38226735, 1.42793754] | 0 |
| noise_2 | 0/400 | 0/400 | [-1.42028935, 1.36161174] | 0 |
| noise_3 | 0/400 | 0/400 | [-1.36745865, 1.41274178] | 0 |
| noise_4 | 0/400 | 0/400 | [-1.41535178, 1.41622976] | 0 |
| noise_5 | 0/400 | 0/400 | [-1.42025271, 1.41238259] | 0 |
| noise_6 | 0/400 | 0/400 | [-1.37159924, 1.39873694] | 0 |

## 103 / within-range-subgroup

Capture: `{'captured_args': 4800, 'serialized': 800, 'envelope_bytes': 1295667, 'attempted': 1600, 'accepted': 1600, 'network_requests': 626, 'exported': 1600}`. Wrapper/context timings ns: `{'wrapper_p50': 257600.0, 'wrapper_p99': 602274.9999999999, 'run_enter_p50': 28900.0, 'run_exit_p50': 8900.0}`.
Invariant counts by arm: `{'problem': {'violated': 80, 'not_applicable': 320}, 'reference': {'satisfied': 80, 'not_applicable': 320}}`.
Support-premise misses (scores unchanged): `[]`.

### legacy_median

Positive count 2; top 1 output.lane; recall@1/3/5 `{'1': 0.0, '3': 0.5, '5': 0.5}`.
Ranks/scores: `[{'feature': 'output.lane', 'score': 0.9950248756218905}, {'feature': 'current_value', 'score': 0.4853724757908721}]`.
Ties including unranked zero observations: `[{'score': 0, 'features': ['gap', 'noise_0', 'noise_1', 'noise_2', 'noise_3', 'noise_4', 'noise_5', 'noise_6', 'previous_age']}]`.
Nuisance above mechanism: `[]`.

### median

Positive count 1; top 1 current_value; recall@1/3/5 `{'1': 0.5, '3': 0.5, '5': 0.5}`.
Ranks/scores: `[{'feature': 'current_value', 'score': 0.4853724757908721}]`.
Ties including unranked zero observations: `[{'score': 0, 'features': ['gap', 'noise_0', 'noise_1', 'noise_2', 'noise_3', 'noise_4', 'noise_5', 'noise_6', 'previous_age']}]`.
Nuisance above mechanism: `[]`.

### tail

Positive count 1; top 1 current_value; recall@1/3/5 `{'1': 0.5, '3': 0.5, '5': 0.5}`.
Ranks/scores: `[{'feature': 'current_value', 'score': 0.7960199004975124}]`.
Ties including unranked zero observations: `[{'score': 0, 'features': ['gap', 'noise_0', 'noise_1', 'noise_2', 'noise_3', 'noise_4', 'noise_5', 'noise_6', 'previous_age']}]`.
Nuisance above mechanism: `[]`.

| Tail feature | Raw P outside / observed | Raw R outside / observed | Interval | Score |
|---|---|---|---|---|
| current_value | 320/400 | 0/400 | [0.5, 1.5] | 0.7960199 |
| gap | 27/400 | 27/400 | [0.506607917, 0.940722852] | 0 |
| noise_0 | 0/400 | 0/400 | [-1.39891194, 1.39033731] | 0 |
| noise_1 | 0/400 | 0/400 | [-1.39201361, 1.42656889] | 0 |
| noise_2 | 0/400 | 0/400 | [-1.38088931, 1.39884817] | 0 |
| noise_3 | 0/400 | 0/400 | [-1.38240547, 1.38542833] | 0 |
| noise_4 | 0/400 | 0/400 | [-1.41594721, 1.39923231] | 0 |
| noise_5 | 0/400 | 0/400 | [-1.37612119, 1.38997329] | 0 |
| noise_6 | 0/400 | 0/400 | [-1.4108288, 1.41193745] | 0 |
| previous_age | 0/400 | 0/400 | [0.5, 2.5] | 0 |

## 103 / moderate-decoy

Capture: `{'captured_args': 4800, 'serialized': 800, 'envelope_bytes': 1295466, 'attempted': 1600, 'accepted': 1600, 'network_requests': 581, 'exported': 1600}`. Wrapper/context timings ns: `{'wrapper_p50': 278950.0, 'wrapper_p99': 846346.0, 'run_enter_p50': 27300.0, 'run_exit_p50': 9700.0}`.
Invariant counts by arm: `{'problem': {'not_applicable': 320, 'violated': 80}, 'reference': {'not_applicable': 320, 'satisfied': 80}}`.
Support-premise misses (scores unchanged): `[]`.

### legacy_median

Positive count 3; top 1 output.lane; recall@1/3/5 `{'1': 0.0, '3': 0.5, '5': 0.5}`.
Ranks/scores: `[{'feature': 'output.lane', 'score': 0.9950248756218905}, {'feature': 'current_value', 'score': 0.4853724757908721}, {'feature': 'noise_0', 'score': 0.19203613608975048}]`.
Ties including unranked zero observations: `[{'score': 0, 'features': ['gap', 'noise_1', 'noise_2', 'noise_3', 'noise_4', 'noise_5', 'noise_6', 'previous_age']}]`.
Nuisance above mechanism: `[{'nuisance': 'noise_0', 'mechanism': 'previous_age'}]`.

### median

Positive count 2; top 1 current_value; recall@1/3/5 `{'1': 0.5, '3': 0.5, '5': 0.5}`.
Ranks/scores: `[{'feature': 'current_value', 'score': 0.4853724757908721}, {'feature': 'noise_0', 'score': 0.19203613608975048}]`.
Ties including unranked zero observations: `[{'score': 0, 'features': ['gap', 'noise_1', 'noise_2', 'noise_3', 'noise_4', 'noise_5', 'noise_6', 'previous_age']}]`.
Nuisance above mechanism: `[{'nuisance': 'noise_0', 'mechanism': 'previous_age'}]`.

### tail

Positive count 3; top 1 current_value; recall@1/3/5 `{'1': 0.5, '3': 1.0, '5': 1.0}`.
Ranks/scores: `[{'feature': 'current_value', 'score': 0.7960199004975124}, {'feature': 'previous_age', 'score': 0.19900497512437812}, {'feature': 'noise_0', 'score': 0.17412935323383086}]`.
Ties including unranked zero observations: `[{'score': 0, 'features': ['gap', 'noise_1', 'noise_2', 'noise_3', 'noise_4', 'noise_5', 'noise_6']}]`.
Nuisance above mechanism: `[]`.

| Tail feature | Raw P outside / observed | Raw R outside / observed | Interval | Score |
|---|---|---|---|---|
| current_value | 320/400 | 0/400 | [0.5, 1.5] | 0.7960199 |
| previous_age | 80/400 | 0/400 | [0.5, 1.5] | 0.199004975 |
| noise_0 | 70/400 | 0/400 | [-1.42224346, 1.37755025] | 0.174129353 |
| gap | 21/400 | 21/400 | [0.506389434, 0.931183432] | 0 |
| noise_1 | 0/400 | 0/400 | [-1.39995046, 1.37746052] | 0 |
| noise_2 | 0/400 | 0/400 | [-1.39022947, 1.40925928] | 0 |
| noise_3 | 0/400 | 0/400 | [-1.37228772, 1.40324269] | 0 |
| noise_4 | 0/400 | 0/400 | [-1.37096074, 1.37886343] | 0 |
| noise_5 | 0/400 | 0/400 | [-1.40184898, 1.37389073] | 0 |
| noise_6 | 0/400 | 0/400 | [-1.40406695, 1.38500859] | 0 |

## 103 / unrelated-nuisance

Capture: `{'captured_args': 4800, 'serialized': 800, 'envelope_bytes': 1295639, 'attempted': 1600, 'accepted': 1600, 'network_requests': 563, 'exported': 1600}`. Wrapper/context timings ns: `{'wrapper_p50': 346700.0, 'wrapper_p99': 758440.0, 'run_enter_p50': 29500.0, 'run_exit_p50': 12150.0}`.
Invariant counts by arm: `{'problem': {'violated': 80, 'not_applicable': 320}, 'reference': {'not_applicable': 320, 'satisfied': 80}}`.
Support-premise misses (scores unchanged): `[]`.

### legacy_median

Positive count 3; top 1 output.lane; recall@1/3/5 `{'1': 0.0, '3': 0.5, '5': 0.5}`.
Ranks/scores: `[{'feature': 'output.lane', 'score': 0.9950248756218905}, {'feature': 'current_value', 'score': 0.4853724757908721}, {'feature': 'noise_5', 'score': 0.019639565043119134}]`.
Ties including unranked zero observations: `[{'score': 0, 'features': ['gap', 'noise_0', 'noise_1', 'noise_2', 'noise_3', 'noise_4', 'noise_6', 'previous_age']}]`.
Nuisance above mechanism: `[{'nuisance': 'noise_5', 'mechanism': 'previous_age'}]`.

### median

Positive count 2; top 1 current_value; recall@1/3/5 `{'1': 0.5, '3': 0.5, '5': 0.5}`.
Ranks/scores: `[{'feature': 'current_value', 'score': 0.4853724757908721}, {'feature': 'noise_5', 'score': 0.019639565043119134}]`.
Ties including unranked zero observations: `[{'score': 0, 'features': ['gap', 'noise_0', 'noise_1', 'noise_2', 'noise_3', 'noise_4', 'noise_6', 'previous_age']}]`.
Nuisance above mechanism: `[{'nuisance': 'noise_5', 'mechanism': 'previous_age'}]`.

### tail

Positive count 2; top 1 current_value; recall@1/3/5 `{'1': 0.5, '3': 1.0, '5': 1.0}`.
Ranks/scores: `[{'feature': 'current_value', 'score': 0.7960199004975124}, {'feature': 'previous_age', 'score': 0.19900497512437812}]`.
Ties including unranked zero observations: `[{'score': 0, 'features': ['gap', 'noise_0', 'noise_1', 'noise_2', 'noise_3', 'noise_4', 'noise_5', 'noise_6']}]`.
Nuisance above mechanism: `[]`.

| Tail feature | Raw P outside / observed | Raw R outside / observed | Interval | Score |
|---|---|---|---|---|
| current_value | 320/400 | 0/400 | [0.5, 1.5] | 0.7960199 |
| previous_age | 80/400 | 0/400 | [0.5, 1.5] | 0.199004975 |
| gap | 21/400 | 21/400 | [0.502560948, 0.926855393] | 0 |
| noise_0 | 0/400 | 0/400 | [-1.40137437, 1.42015606] | 0 |
| noise_1 | 0/400 | 0/400 | [-1.39336901, 1.40640833] | 0 |
| noise_2 | 0/400 | 0/400 | [-1.4036679, 1.42773731] | 0 |
| noise_3 | 0/400 | 0/400 | [-1.41858436, 1.40848486] | 0 |
| noise_4 | 0/400 | 0/400 | [-1.41286166, 1.39739376] | 0 |
| noise_5 | 0/400 | 0/400 | [-1.36205496, 1.40940441] | 0 |
| noise_6 | 0/400 | 0/400 | [-1.38507494, 1.41526886] | 0 |
