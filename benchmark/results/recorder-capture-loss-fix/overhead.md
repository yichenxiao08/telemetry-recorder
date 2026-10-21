# Every overhead trial and miss

Same S0a payload/method. Alternating Checkpoint A/fixed order; fresh process/store per trial.

| Condition / mode | Trial / source | p50 / p99 us | 100/250 pass | Context enter p50 / p99 us | Context exit p50 / p99 us | Flush s | Attempted / ack / failed / dropped / unknown |
|---|---|---|---|---|---|---:|---|
| reachable / 60hz | 1 / checkpoint-a | 210.100 / 410.011 | False | 38.300 / 123.441 | 3.800 / 10.100 | 0.000018 | 3600 / 3600 / legacy unack=0 / drops=0; old unknown not separately classified |
| reachable / 60hz | 1 / fixed | 208.200 / 424.401 | False | 38.800 / 77.604 | 4.000 / 11.601 | 0.000151 | 3600 / 3600 / 0 / 0 / 0 |
| reachable / 60hz | 2 / fixed | 201.550 / 380.504 | False | 38.700 / 79.174 | 3.800 / 10.302 | 0.000050 | 3600 / 3600 / 0 / 0 / 0 |
| reachable / 60hz | 2 / checkpoint-a | 221.950 / 440.305 | False | 40.600 / 150.010 | 3.700 / 10.002 | 0.000010 | 3600 / 3600 / legacy unack=0 / drops=0; old unknown not separately classified |
| reachable / 60hz | 3 / checkpoint-a | 206.250 / 354.889 | False | 38.300 / 138.702 | 3.600 / 8.600 | 0.000011 | 3600 / 3600 / legacy unack=0 / drops=0; old unknown not separately classified |
| reachable / 60hz | 3 / fixed | 206.600 / 404.555 | False | 36.800 / 79.000 | 4.000 / 10.902 | 0.010719 | 3600 / 3600 / 0 / 0 / 0 |
| reachable / 60hz | 4 / fixed | 206.800 / 425.810 | False | 36.700 / 86.800 | 4.100 / 10.800 | 0.000066 | 3600 / 3600 / 0 / 0 / 0 |
| reachable / 60hz | 4 / checkpoint-a | 206.900 / 392.503 | False | 37.300 / 116.239 | 3.800 / 10.800 | 0.000019 | 3600 / 3600 / legacy unack=0 / drops=0; old unknown not separately classified |
| reachable / 60hz | 5 / checkpoint-a | 203.200 / 403.825 | False | 37.400 / 91.028 | 3.700 / 10.601 | 0.000006 | 3600 / 3600 / legacy unack=0 / drops=0; old unknown not separately classified |
| reachable / 60hz | 5 / fixed | 199.500 / 403.614 | False | 35.600 / 151.112 | 3.700 / 9.501 | 0.000049 | 3600 / 3600 / 0 / 0 / 0 |
| reachable / tight | 1 / checkpoint-a | 44.800 / 139.508 | True | 2.700 / 8.700 | 1.900 / 5.400 | 0.369830 | 20000 / 406 / legacy unack=0 / drops=19594; old unknown not separately classified |
| reachable / tight | 1 / fixed | 44.400 / 142.404 | True | 2.700 / 9.100 | 1.800 / 5.400 | 0.281401 | 20000 / 384 / 0 / 19616 / 0 |
| reachable / tight | 2 / fixed | 44.000 / 141.002 | True | 2.700 / 8.701 | 1.800 / 5.300 | 0.338888 | 20000 / 384 / 0 / 19616 / 0 |
| reachable / tight | 2 / checkpoint-a | 45.100 / 163.409 | True | 2.700 / 9.900 | 1.900 / 6.000 | 0.320109 | 20000 / 384 / legacy unack=0 / drops=19616; old unknown not separately classified |
| reachable / tight | 3 / checkpoint-a | 44.500 / 141.003 | True | 2.700 / 8.700 | 1.900 / 5.501 | 0.269459 | 20000 / 384 / legacy unack=0 / drops=19616; old unknown not separately classified |
| reachable / tight | 3 / fixed | 44.800 / 126.502 | True | 2.700 / 7.800 | 1.800 / 4.801 | 0.307275 | 20000 / 384 / 0 / 19616 / 0 |
| reachable / tight | 4 / fixed | 44.200 / 152.803 | True | 2.700 / 9.500 | 1.800 / 5.700 | 0.283514 | 20000 / 384 / 0 / 19616 / 0 |
| reachable / tight | 4 / checkpoint-a | 45.400 / 143.606 | True | 2.700 / 8.501 | 2.000 / 5.500 | 0.265894 | 20000 / 416 / legacy unack=0 / drops=19584; old unknown not separately classified |
| reachable / tight | 5 / checkpoint-a | 45.000 / 154.201 | True | 2.700 / 9.600 | 1.900 / 6.000 | 0.280996 | 20000 / 416 / legacy unack=0 / drops=19584; old unknown not separately classified |
| reachable / tight | 5 / fixed | 45.100 / 143.213 | True | 2.700 / 9.800 | 1.900 / 5.400 | 0.309532 | 20000 / 352 / 0 / 19648 / 0 |
| unreachable / 60hz | 1 / checkpoint-a | 218.950 / 978.697 | False | 47.300 / 210.033 | 7.400 / 22.204 | 0.000025 | 3600 / 0 / legacy unack=1242 / drops=2358; old unknown not separately classified |
| unreachable / 60hz | 1 / fixed | 345.050 / 700.975 | False | 52.550 / 146.008 | 8.600 / 15.800 | 10.002315 | 3600 / 0 / 578 / 2894 / 128 |
| unreachable / 60hz | 2 / fixed | 369.450 / 874.247 | False | 54.700 / 254.754 | 9.400 / 23.803 | 10.010393 | 3600 / 0 / 578 / 2894 / 128 |
| unreachable / 60hz | 2 / checkpoint-a | 235.200 / 992.539 | False | 54.500 / 249.984 | 8.300 / 44.114 | 0.000034 | 3600 / 0 / legacy unack=1240 / drops=2360; old unknown not separately classified |
| unreachable / 60hz | 3 / checkpoint-a | 217.600 / 1055.082 | False | 48.400 / 225.402 | 7.100 / 69.340 | 0.000033 | 3600 / 0 / legacy unack=1240 / drops=2360; old unknown not separately classified |
| unreachable / 60hz | 3 / fixed | 274.450 / 742.577 | False | 46.300 / 135.744 | 6.300 / 16.301 | 10.006514 | 3600 / 0 / 578 / 2894 / 128 |
| unreachable / 60hz | 4 / fixed | 264.250 / 965.936 | False | 44.100 / 153.722 | 6.200 / 21.503 | 10.011881 | 3600 / 0 / 578 / 2894 / 128 |
| unreachable / 60hz | 4 / checkpoint-a | 166.800 / 766.006 | False | 41.400 / 172.521 | 4.500 / 16.901 | 0.000021 | 3600 / 0 / legacy unack=1238 / drops=2362; old unknown not separately classified |
| unreachable / 60hz | 5 / checkpoint-a | 174.050 / 645.104 | False | 46.300 / 131.036 | 4.500 / 13.902 | 0.000030 | 3600 / 0 / legacy unack=1240 / drops=2360; old unknown not separately classified |
| unreachable / 60hz | 5 / fixed | 246.050 / 815.099 | False | 43.900 / 135.204 | 5.500 / 19.601 | 10.011776 | 3600 / 0 / 578 / 2894 / 128 |
| unreachable / tight | 1 / checkpoint-a | 43.400 / 143.803 | True | 2.600 / 8.800 | 1.800 / 5.700 | 0.057543 | 20000 / 0 / legacy unack=288 / drops=19712; old unknown not separately classified |
| unreachable / tight | 1 / fixed | 42.500 / 122.201 | True | 2.600 / 7.600 | 1.700 / 4.800 | 10.012215 | 20000 / 0 / 160 / 19712 / 128 |
| unreachable / tight | 2 / fixed | 43.400 / 100.201 | True | 2.600 / 6.300 | 1.700 / 3.900 | 10.015003 | 20000 / 0 / 160 / 19712 / 128 |
| unreachable / tight | 2 / checkpoint-a | 44.300 / 129.303 | True | 2.700 / 8.101 | 1.800 / 5.300 | 0.074986 | 20000 / 0 / legacy unack=288 / drops=19712; old unknown not separately classified |
| unreachable / tight | 3 / checkpoint-a | 42.700 / 127.102 | True | 2.600 / 7.900 | 1.800 / 5.100 | 0.068843 | 20000 / 0 / legacy unack=288 / drops=19712; old unknown not separately classified |
| unreachable / tight | 3 / fixed | 44.000 / 126.205 | True | 2.700 / 7.900 | 1.800 / 5.100 | 10.002625 | 20000 / 0 / 160 / 19712 / 128 |
| unreachable / tight | 4 / fixed | 44.300 / 138.902 | True | 2.700 / 8.600 | 1.800 / 5.400 | 10.014904 | 20000 / 0 / 160 / 19712 / 128 |
| unreachable / tight | 4 / checkpoint-a | 44.200 / 131.705 | True | 2.700 / 8.200 | 1.800 / 5.400 | 0.069902 | 20000 / 0 / legacy unack=288 / drops=19712; old unknown not separately classified |
| unreachable / tight | 5 / checkpoint-a | 44.300 / 143.001 | True | 2.700 / 8.901 | 1.800 / 5.400 | 0.026750 | 20000 / 0 / legacy unack=288 / drops=19712; old unknown not separately classified |
| unreachable / tight | 5 / fixed | 44.100 / 144.001 | True | 2.700 / 9.100 | 1.800 / 5.500 | 10.005569 | 20000 / 0 / 160 / 19712 / 128 |
| saturated / 60hz | 1 / checkpoint-a | 356.300 / 997.541 | False | 50.500 / 198.635 | 9.700 / 32.805 | 0.014286 | 3600 / 0 / legacy unack=0 / drops=3599; old unknown not separately classified |
| saturated / 60hz | 1 / fixed | 416.600 / 836.224 | False | 63.200 / 218.619 | 9.800 / 21.806 | 0.024488 | 3600 / 0 / 0 / 3599 / 1 |
| saturated / 60hz | 2 / fixed | 332.000 / 891.342 | False | 52.150 / 187.012 | 8.400 / 22.202 | 0.011012 | 3600 / 0 / 0 / 3599 / 1 |
| saturated / 60hz | 2 / checkpoint-a | 343.800 / 835.211 | False | 55.700 / 182.226 | 8.500 / 21.602 | 0.023077 | 3600 / 0 / legacy unack=0 / drops=3599; old unknown not separately classified |
| saturated / 60hz | 3 / checkpoint-a | 325.650 / 665.541 | False | 49.300 / 121.734 | 8.300 / 14.501 | 0.020719 | 3600 / 0 / legacy unack=0 / drops=3599; old unknown not separately classified |
| saturated / 60hz | 3 / fixed | 401.200 / 719.304 | False | 66.400 / 167.245 | 8.900 / 16.600 | 0.022167 | 3600 / 0 / 0 / 3599 / 1 |
| saturated / 60hz | 4 / fixed | 339.450 / 777.215 | False | 47.600 / 191.538 | 8.600 / 39.630 | 0.016216 | 3600 / 0 / 0 / 3599 / 1 |
| saturated / 60hz | 4 / checkpoint-a | 361.600 / 865.344 | False | 53.700 / 169.879 | 9.000 / 21.009 | 0.011866 | 3600 / 0 / legacy unack=0 / drops=3599; old unknown not separately classified |
| saturated / 60hz | 5 / checkpoint-a | 348.250 / 802.399 | False | 53.700 / 168.617 | 8.800 / 18.702 | 0.011719 | 3600 / 0 / legacy unack=0 / drops=3599; old unknown not separately classified |
| saturated / 60hz | 5 / fixed | 313.450 / 892.519 | False | 46.500 / 181.419 | 8.400 / 22.801 | 0.016022 | 3600 / 0 / 0 / 3599 / 1 |
| saturated / tight | 1 / checkpoint-a | 46.300 / 138.713 | True | 2.900 / 9.900 | 1.900 / 4.900 | 0.020152 | 20000 / 0 / legacy unack=0 / drops=19999; old unknown not separately classified |
| saturated / tight | 1 / fixed | 44.300 / 137.303 | True | 2.700 / 8.501 | 1.800 / 5.200 | 0.020929 | 20000 / 0 / 0 / 19999 / 1 |
| saturated / tight | 2 / fixed | 45.000 / 144.300 | True | 2.600 / 8.800 | 1.800 / 5.400 | 0.029774 | 20000 / 0 / 0 / 19999 / 1 |
| saturated / tight | 2 / checkpoint-a | 44.400 / 133.210 | True | 2.700 / 8.901 | 1.900 / 5.400 | 0.010614 | 20000 / 0 / legacy unack=0 / drops=19999; old unknown not separately classified |
| saturated / tight | 3 / checkpoint-a | 44.600 / 124.503 | True | 2.700 / 7.801 | 1.800 / 5.100 | 0.010110 | 20000 / 0 / legacy unack=0 / drops=19999; old unknown not separately classified |
| saturated / tight | 3 / fixed | 43.500 / 138.908 | True | 2.600 / 8.800 | 1.800 / 5.600 | 0.021661 | 20000 / 0 / 0 / 19999 / 1 |
| saturated / tight | 4 / fixed | 44.000 / 123.400 | True | 2.700 / 7.600 | 1.800 / 5.100 | 0.012204 | 20000 / 0 / 0 / 19999 / 1 |
| saturated / tight | 4 / checkpoint-a | 44.400 / 136.904 | True | 2.700 / 8.800 | 1.900 / 5.200 | 0.017952 | 20000 / 0 / legacy unack=0 / drops=19999; old unknown not separately classified |
| saturated / tight | 5 / checkpoint-a | 44.400 / 131.503 | True | 2.700 / 8.100 | 1.900 / 5.201 | 0.014383 | 20000 / 0 / legacy unack=0 / drops=19999; old unknown not separately classified |
| saturated / tight | 5 / fixed | 43.700 / 129.203 | True | 2.600 / 7.800 | 1.800 / 4.900 | 0.012783 | 20000 / 0 / 0 / 19999 / 1 |
