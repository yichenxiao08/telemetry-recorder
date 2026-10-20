# Frozen lane-01 diagnostic

Synthetic data only. Read-only pinned evidence; every complete report is checked against lane-01.json. No new seeds or proposed methods are evaluated. JSON contains full precision, ranges, per-session counts/frame ranges/noise means, and four chronological bin counts.

Numeric: scale=max(1.4826 MAD,epsilon); z=abs(P-R)/scale; score=(1-exp(-z/3))*(1-(e+1)/102), unless abs(P-R)<floor, then zero. e includes ties in the direction of P. For below-floor rows e/z are diagnostic terms: the scorer exits before computing them. Categorical medians/scales/z/e are inapplicable; show smoothed TV and raw category counts instead. Rank — means unranked, not tied for a discovery.

## decoy-heavy / 1009

| Field | Rank | P median | R median | MAD | epsilon | scale | z | e | Score | Floor exit |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---|
| output.lane | 1 | N/A | N/A | N/A | N/A | N/A | N/A | N/A | 0.980392157 | N/A |
| noise_2 | 2 | 10.149129 | 0.061138 | 0.561842 | 1.00 | 1.000000 | 10.087992 | 0 | 0.955892861 | false |
| noise_3 | 3 | 10.013526 | -0.021263 | 0.407799 | 1.00 | 1.000000 | 10.034789 | 0 | 0.955279095 | false |
| noise_0 | 4 | 9.938709 | -0.091614 | 0.424178 | 1.00 | 1.000000 | 10.030323 | 0 | 0.955227076 | false |
| noise_1 | 5 | 10.063085 | 0.053770 | 0.486507 | 1.00 | 1.000000 | 10.009315 | 0 | 0.954981341 | false |
| noise_4 | 6 | 10.020859 | 0.189518 | 0.408387 | 1.00 | 1.000000 | 9.831341 | 0 | 0.952829023 | false |
| noise_6 | 7 | 9.877766 | 0.087366 | 0.471254 | 1.00 | 1.000000 | 9.790399 | 0 | 0.952315576 | false |
| noise_5 | 8 | 9.864796 | 0.112185 | 0.469365 | 1.00 | 1.000000 | 9.752611 | 0 | 0.951835410 | false |
| gap | 9 | 0.520854 | 0.833883 | 0.063329 | 0.01 | 0.093891 | 3.333948 | 0 | 0.664297295 | false |
| previous_age | 10 | 2.000000 | 1.000000 | 0.000000 | 1.00 | 1.000000 | 1.000000 | 0 | 0.280689585 | false |
| chosen.option | — | N/A | N/A | N/A | N/A | N/A | N/A | N/A | 0.000000000 | N/A |
| reason_code | — | N/A | N/A | N/A | N/A | N/A | N/A | N/A | 0.000000000 | N/A |
| verdict | — | N/A | N/A | N/A | N/A | N/A | N/A | N/A | 0.000000000 | N/A |

Exact ties (including unranked zero observations): [[chosen.option reason_code verdict]]. Smallest distinct score gap: 5.20188715594e-05.

output.lane raw P/R categories: [map[left:100] map[right:100]].

chosen.option raw P/R categories: [map[returned:100] map[returned:100]].

reason_code raw P/R categories: [map[:100] map[:100]].

verdict raw P/R categories: [map[returned:100] map[returned:100]].

Chronological 50-frame bins P/R: [[20 30] [24 26] [30 20] [26 24]]. Session P/R counts and mean noise (averaged over seven fields):

| Session | P/R | P noise | R noise |
|---|---|---:|---:|
| lane-01-1009-decoy-heavy-session-0 | [10 10] | 9.902644 | 0.125297 |
| lane-01-1009-decoy-heavy-session-1 | [6 14] | 9.942718 | -0.037724 |
| lane-01-1009-decoy-heavy-session-2 | [10 10] | 9.969519 | 0.098517 |
| lane-01-1009-decoy-heavy-session-3 | [10 10] | 10.017059 | -0.005414 |
| lane-01-1009-decoy-heavy-session-4 | [8 12] | 10.101977 | -0.016014 |
| lane-01-1009-decoy-heavy-session-5 | [14 6] | 10.034586 | -0.004228 |
| lane-01-1009-decoy-heavy-session-6 | [7 13] | 9.932604 | 0.025962 |
| lane-01-1009-decoy-heavy-session-7 | [12 8] | 10.038023 | -0.008971 |
| lane-01-1009-decoy-heavy-session-8 | [11 9] | 10.006060 | 0.036216 |
| lane-01-1009-decoy-heavy-session-9 | [12 8] | 9.965694 | 0.127910 |

## harder / 1009

| Field | Rank | P median | R median | MAD | epsilon | scale | z | e | Score | Floor exit |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---|
| output.lane | 1 | N/A | N/A | N/A | N/A | N/A | N/A | N/A | 0.980392157 | N/A |
| gap | 2 | 0.520223 | 0.795499 | 0.071115 | 0.01 | 0.105435 | 2.610866 | 0 | 0.575471585 | false |
| noise_2 | 3 | 0.093053 | -0.261583 | 0.432687 | 1.00 | 1.000000 | 0.354636 | 40 | 0.066676784 | false |
| noise_0 | 4 | 0.136627 | -0.051452 | 0.525842 | 1.00 | 1.000000 | 0.188079 | 40 | 0.036341806 | false |
| noise_4 | 5 | -0.115521 | 0.020460 | 0.593320 | 1.00 | 1.000000 | 0.135981 | 44 | 0.024764343 | false |
| chosen.option | — | N/A | N/A | N/A | N/A | N/A | N/A | N/A | 0.000000000 | N/A |
| reason_code | — | N/A | N/A | N/A | N/A | N/A | N/A | N/A | 0.000000000 | N/A |
| verdict | — | N/A | N/A | N/A | N/A | N/A | N/A | N/A | 0.000000000 | N/A |
| noise_1 | — | 0.140307 | 0.062758 | 0.483471 | 1.00 | 1.000000 | 0.077549 | 45 | 0.000000000 | true |
| noise_3 | — | 0.000686 | -0.057768 | 0.441548 | 1.00 | 1.000000 | 0.058453 | 45 | 0.000000000 | true |
| noise_5 | — | 0.037591 | -0.055680 | 0.437714 | 1.00 | 1.000000 | 0.093270 | 42 | 0.000000000 | true |
| noise_6 | — | 0.015321 | 0.111879 | 0.499779 | 1.00 | 1.000000 | 0.096558 | 47 | 0.000000000 | true |
| previous_age | — | 1.000000 | 1.000000 | 0.000000 | 1.00 | 1.000000 | 0.000000 | 100 | 0.000000000 | true |

Exact ties (including unranked zero observations): [[chosen.option reason_code verdict noise_1 noise_3 noise_5 noise_6 previous_age]]. Smallest distinct score gap: 0.0115774631724.

output.lane raw P/R categories: [map[left:100] map[right:100]].

chosen.option raw P/R categories: [map[returned:100] map[returned:100]].

reason_code raw P/R categories: [map[:100] map[:100]].

verdict raw P/R categories: [map[returned:100] map[returned:100]].

Chronological 50-frame bins P/R: [[25 25] [21 29] [28 22] [26 24]]. Session P/R counts and mean noise (averaged over seven fields):

| Session | P/R | P noise | R noise |
|---|---|---:|---:|
| lane-01-1009-harder-session-0 | [13 7] | 0.014017 | -0.036557 |
| lane-01-1009-harder-session-1 | [6 14] | -0.087764 | -0.034907 |
| lane-01-1009-harder-session-2 | [12 8] | -0.028572 | -0.071595 |
| lane-01-1009-harder-session-3 | [9 11] | 0.033644 | 0.022981 |
| lane-01-1009-harder-session-4 | [6 14] | 0.038474 | -0.096308 |
| lane-01-1009-harder-session-5 | [10 10] | -0.088378 | -0.008827 |
| lane-01-1009-harder-session-6 | [13 7] | 0.115836 | 0.083775 |
| lane-01-1009-harder-session-7 | [12 8] | -0.025698 | -0.021381 |
| lane-01-1009-harder-session-8 | [9 11] | -0.028702 | -0.011350 |
| lane-01-1009-harder-session-9 | [10 10] | 0.092613 | -0.039908 |

## decoy-heavy / 2027

| Field | Rank | P median | R median | MAD | epsilon | scale | z | e | Score | Floor exit |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---|
| output.lane | 1 | N/A | N/A | N/A | N/A | N/A | N/A | N/A | 0.980392157 | N/A |
| noise_2 | 2 | 10.060667 | -0.119937 | 0.540261 | 1.00 | 1.000000 | 10.180604 | 0 | 0.956935655 | false |
| noise_6 | 3 | 10.018404 | -0.142645 | 0.506273 | 1.00 | 1.000000 | 10.161049 | 0 | 0.956718140 | false |
| noise_5 | 4 | 10.095559 | -0.049112 | 0.469635 | 1.00 | 1.000000 | 10.144672 | 0 | 0.956534883 | false |
| noise_4 | 5 | 10.049116 | 0.056589 | 0.544913 | 1.00 | 1.000000 | 9.992527 | 0 | 0.954783732 | false |
| noise_1 | 6 | 9.958006 | -0.018915 | 0.447903 | 1.00 | 1.000000 | 9.976921 | 0 | 0.954599037 | false |
| noise_0 | 7 | 9.843014 | -0.132764 | 0.380631 | 1.00 | 1.000000 | 9.975778 | 0 | 0.954585468 | false |
| noise_3 | 8 | 10.043745 | 0.078083 | 0.483917 | 1.00 | 1.000000 | 9.965662 | 0 | 0.954465192 | false |
| gap | 9 | 0.520910 | 0.797991 | 0.074785 | 0.01 | 0.110876 | 2.499025 | 0 | 0.559718809 | false |
| previous_age | 10 | 2.000000 | 1.000000 | 0.000000 | 1.00 | 1.000000 | 1.000000 | 0 | 0.280689585 | false |
| chosen.option | — | N/A | N/A | N/A | N/A | N/A | N/A | N/A | 0.000000000 | N/A |
| reason_code | — | N/A | N/A | N/A | N/A | N/A | N/A | N/A | 0.000000000 | N/A |
| verdict | — | N/A | N/A | N/A | N/A | N/A | N/A | N/A | 0.000000000 | N/A |

Exact ties (including unranked zero observations): [[chosen.option reason_code verdict]]. Smallest distinct score gap: 1.3568720436e-05.

output.lane raw P/R categories: [map[left:100] map[right:100]].

chosen.option raw P/R categories: [map[returned:100] map[returned:100]].

reason_code raw P/R categories: [map[:100] map[:100]].

verdict raw P/R categories: [map[returned:100] map[returned:100]].

Chronological 50-frame bins P/R: [[25 25] [21 29] [25 25] [29 21]]. Session P/R counts and mean noise (averaged over seven fields):

| Session | P/R | P noise | R noise |
|---|---|---:|---:|
| lane-01-2027-decoy-heavy-session-0 | [9 11] | 9.925449 | -0.031904 |
| lane-01-2027-decoy-heavy-session-1 | [10 10] | 10.158382 | -0.016167 |
| lane-01-2027-decoy-heavy-session-2 | [10 10] | 10.010428 | -0.035794 |
| lane-01-2027-decoy-heavy-session-3 | [7 13] | 10.001228 | -0.134610 |
| lane-01-2027-decoy-heavy-session-4 | [10 10] | 10.007144 | -0.070962 |
| lane-01-2027-decoy-heavy-session-5 | [9 11] | 10.067594 | 0.006180 |
| lane-01-2027-decoy-heavy-session-6 | [12 8] | 9.964014 | -0.001314 |
| lane-01-2027-decoy-heavy-session-7 | [9 11] | 10.019479 | 0.039639 |
| lane-01-2027-decoy-heavy-session-8 | [14 6] | 10.006033 | -0.019590 |
| lane-01-2027-decoy-heavy-session-9 | [10 10] | 9.988490 | -0.013271 |

## harder / 2027

| Field | Rank | P median | R median | MAD | epsilon | scale | z | e | Score | Floor exit |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---|
| output.lane | 1 | N/A | N/A | N/A | N/A | N/A | N/A | N/A | 0.980392157 | N/A |
| gap | 2 | 0.519732 | 0.806824 | 0.068849 | 0.01 | 0.102076 | 2.812535 | 0 | 0.602434264 | false |
| noise_6 | 3 | -0.179078 | -0.015934 | 0.484931 | 1.00 | 1.000000 | 0.163144 | 40 | 0.031653677 | false |
| noise_5 | 4 | -0.069020 | 0.081042 | 0.508966 | 1.00 | 1.000000 | 0.150062 | 44 | 0.027265168 | false |
| chosen.option | — | N/A | N/A | N/A | N/A | N/A | N/A | N/A | 0.000000000 | N/A |
| reason_code | — | N/A | N/A | N/A | N/A | N/A | N/A | N/A | 0.000000000 | N/A |
| verdict | — | N/A | N/A | N/A | N/A | N/A | N/A | N/A | 0.000000000 | N/A |
| noise_0 | — | -0.122826 | -0.156986 | 0.513672 | 1.00 | 1.000000 | 0.034160 | 48 | 0.000000000 | true |
| noise_1 | — | -0.005165 | -0.050629 | 0.446654 | 1.00 | 1.000000 | 0.045464 | 46 | 0.000000000 | true |
| noise_2 | — | -0.056458 | 0.032982 | 0.593145 | 1.00 | 1.000000 | 0.089440 | 46 | 0.000000000 | true |
| noise_3 | — | 0.067830 | 0.030479 | 0.421676 | 1.00 | 1.000000 | 0.037351 | 46 | 0.000000000 | true |
| noise_4 | — | 0.056589 | 0.049116 | 0.489083 | 1.00 | 1.000000 | 0.007473 | 50 | 0.000000000 | true |
| previous_age | — | 1.000000 | 1.000000 | 0.000000 | 1.00 | 1.000000 | 0.000000 | 100 | 0.000000000 | true |

Exact ties (including unranked zero observations): [[chosen.option reason_code verdict noise_0 noise_1 noise_2 noise_3 noise_4 previous_age]]. Smallest distinct score gap: 0.00438850958187.

output.lane raw P/R categories: [map[left:100] map[right:100]].

chosen.option raw P/R categories: [map[returned:100] map[returned:100]].

reason_code raw P/R categories: [map[:100] map[:100]].

verdict raw P/R categories: [map[returned:100] map[returned:100]].

Chronological 50-frame bins P/R: [[23 27] [21 29] [30 20] [26 24]]. Session P/R counts and mean noise (averaged over seven fields):

| Session | P/R | P noise | R noise |
|---|---|---:|---:|
| lane-01-2027-harder-session-0 | [8 12] | -0.054095 | 0.026193 |
| lane-01-2027-harder-session-1 | [9 11] | -0.083766 | 0.068864 |
| lane-01-2027-harder-session-2 | [11 9] | -0.038530 | 0.071430 |
| lane-01-2027-harder-session-3 | [8 12] | -0.085742 | -0.092559 |
| lane-01-2027-harder-session-4 | [8 12] | -0.059693 | 0.003102 |
| lane-01-2027-harder-session-5 | [12 8] | 0.084358 | 0.015095 |
| lane-01-2027-harder-session-6 | [13 7] | -0.111845 | -0.140482 |
| lane-01-2027-harder-session-7 | [11 9] | 0.068998 | 0.046643 |
| lane-01-2027-harder-session-8 | [11 9] | -0.077193 | -0.042685 |
| lane-01-2027-harder-session-9 | [9 11] | 0.043737 | 0.054542 |

## decoy-heavy / 3037

| Field | Rank | P median | R median | MAD | epsilon | scale | z | e | Score | Floor exit |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---|
| output.lane | 1 | N/A | N/A | N/A | N/A | N/A | N/A | N/A | 0.980392157 | N/A |
| noise_1 | 2 | 10.075478 | -0.108128 | 0.439432 | 1.00 | 1.000000 | 10.183606 | 0 | 0.956968918 | false |
| noise_6 | 3 | 10.106583 | 0.042803 | 0.519046 | 1.00 | 1.000000 | 10.063780 | 0 | 0.955614896 | false |
| noise_5 | 4 | 10.025155 | -0.021759 | 0.579930 | 1.00 | 1.000000 | 10.046915 | 0 | 0.955419941 | false |
| noise_4 | 5 | 9.979659 | -0.053072 | 0.480509 | 1.00 | 1.000000 | 10.032731 | 0 | 0.955255134 | false |
| noise_2 | 6 | 10.075734 | 0.071105 | 0.424366 | 1.00 | 1.000000 | 10.004629 | 0 | 0.954926291 | false |
| noise_0 | 7 | 9.865611 | -0.064226 | 0.421336 | 1.00 | 1.000000 | 9.929837 | 0 | 0.954035944 | false |
| noise_3 | 8 | 9.859059 | 0.198360 | 0.545037 | 1.00 | 1.000000 | 9.660699 | 0 | 0.950641950 | false |
| gap | 9 | 0.517263 | 0.786132 | 0.061767 | 0.01 | 0.091576 | 2.936005 | 0 | 0.618069241 | false |
| previous_age | 10 | 2.000000 | 1.000000 | 0.000000 | 1.00 | 1.000000 | 1.000000 | 0 | 0.280689585 | false |
| chosen.option | — | N/A | N/A | N/A | N/A | N/A | N/A | N/A | 0.000000000 | N/A |
| reason_code | — | N/A | N/A | N/A | N/A | N/A | N/A | N/A | 0.000000000 | N/A |
| verdict | — | N/A | N/A | N/A | N/A | N/A | N/A | N/A | 0.000000000 | N/A |

Exact ties (including unranked zero observations): [[chosen.option reason_code verdict]]. Smallest distinct score gap: 0.00016480677029.

output.lane raw P/R categories: [map[left:100] map[right:100]].

chosen.option raw P/R categories: [map[returned:100] map[returned:100]].

reason_code raw P/R categories: [map[:100] map[:100]].

verdict raw P/R categories: [map[returned:100] map[returned:100]].

Chronological 50-frame bins P/R: [[25 25] [30 20] [25 25] [20 30]]. Session P/R counts and mean noise (averaged over seven fields):

| Session | P/R | P noise | R noise |
|---|---|---:|---:|
| lane-01-3037-decoy-heavy-session-0 | [13 7] | 10.005265 | 0.145140 |
| lane-01-3037-decoy-heavy-session-1 | [8 12] | 9.910836 | 0.052075 |
| lane-01-3037-decoy-heavy-session-2 | [12 8] | 10.078502 | 0.018988 |
| lane-01-3037-decoy-heavy-session-3 | [11 9] | 9.968575 | 0.003855 |
| lane-01-3037-decoy-heavy-session-4 | [11 9] | 10.061010 | -0.121543 |
| lane-01-3037-decoy-heavy-session-5 | [10 10] | 9.869435 | 0.063913 |
| lane-01-3037-decoy-heavy-session-6 | [10 10] | 10.067308 | 0.028784 |
| lane-01-3037-decoy-heavy-session-7 | [8 12] | 10.010870 | 0.112825 |
| lane-01-3037-decoy-heavy-session-8 | [8 12] | 9.980278 | -0.018902 |
| lane-01-3037-decoy-heavy-session-9 | [9 11] | 10.104287 | -0.109886 |

## harder / 3037

| Field | Rank | P median | R median | MAD | epsilon | scale | z | e | Score | Floor exit |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---|
| output.lane | 1 | N/A | N/A | N/A | N/A | N/A | N/A | N/A | 0.980392157 | N/A |
| gap | 2 | 0.519249 | 0.788304 | 0.068380 | 0.01 | 0.101380 | 2.653926 | 0 | 0.581381748 | false |
| noise_0 | 3 | -0.178496 | -0.025312 | 0.539600 | 1.00 | 1.000000 | 0.153184 | 39 | 0.030258151 | false |
| chosen.option | — | N/A | N/A | N/A | N/A | N/A | N/A | N/A | 0.000000000 | N/A |
| reason_code | — | N/A | N/A | N/A | N/A | N/A | N/A | N/A | 0.000000000 | N/A |
| verdict | — | N/A | N/A | N/A | N/A | N/A | N/A | N/A | 0.000000000 | N/A |
| noise_1 | — | 0.066728 | 0.148296 | 0.492874 | 1.00 | 1.000000 | 0.081567 | 47 | 0.000000000 | true |
| noise_2 | — | 0.005033 | 0.079857 | 0.545223 | 1.00 | 1.000000 | 0.074825 | 44 | 0.000000000 | true |
| noise_3 | — | -0.086832 | -0.076140 | 0.402716 | 1.00 | 1.000000 | 0.010691 | 48 | 0.000000000 | true |
| noise_4 | — | -0.070134 | -0.147305 | 0.495714 | 1.00 | 1.000000 | 0.077170 | 44 | 0.000000000 | true |
| noise_5 | — | 0.003235 | 0.093373 | 0.425770 | 1.00 | 1.000000 | 0.090138 | 43 | 0.000000000 | true |
| noise_6 | — | 0.078238 | 0.142921 | 0.464488 | 1.00 | 1.000000 | 0.064683 | 47 | 0.000000000 | true |
| previous_age | — | 1.000000 | 1.000000 | 0.000000 | 1.00 | 1.000000 | 0.000000 | 100 | 0.000000000 | true |

Exact ties (including unranked zero observations): [[chosen.option reason_code verdict noise_1 noise_2 noise_3 noise_4 noise_5 noise_6 previous_age]]. Smallest distinct score gap: 0.030258151443.

output.lane raw P/R categories: [map[left:100] map[right:100]].

chosen.option raw P/R categories: [map[returned:100] map[returned:100]].

reason_code raw P/R categories: [map[:100] map[:100]].

verdict raw P/R categories: [map[returned:100] map[returned:100]].

Chronological 50-frame bins P/R: [[26 24] [28 22] [21 29] [25 25]]. Session P/R counts and mean noise (averaged over seven fields):

| Session | P/R | P noise | R noise |
|---|---|---:|---:|
| lane-01-3037-harder-session-0 | [13 7] | -0.138801 | 0.161772 |
| lane-01-3037-harder-session-1 | [9 11] | 0.049055 | 0.020250 |
| lane-01-3037-harder-session-2 | [11 9] | 0.072049 | -0.011663 |
| lane-01-3037-harder-session-3 | [11 9] | 0.041414 | -0.171458 |
| lane-01-3037-harder-session-4 | [10 10] | 0.115203 | 0.105997 |
| lane-01-3037-harder-session-5 | [8 12] | -0.079803 | 0.026968 |
| lane-01-3037-harder-session-6 | [8 12] | 0.006743 | -0.026168 |
| lane-01-3037-harder-session-7 | [9 11] | -0.016300 | 0.042756 |
| lane-01-3037-harder-session-8 | [11 9] | -0.013852 | 0.061472 |
| lane-01-3037-harder-session-9 | [10 10] | -0.140660 | 0.089777 |

## decoy-heavy / 4051

| Field | Rank | P median | R median | MAD | epsilon | scale | z | e | Score | Floor exit |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---|
| output.lane | 1 | N/A | N/A | N/A | N/A | N/A | N/A | N/A | 0.980392157 | N/A |
| noise_6 | 2 | 9.885486 | -0.135817 | 0.534892 | 1.00 | 1.000000 | 10.021303 | 0 | 0.955121773 | false |
| noise_3 | 3 | 10.032968 | 0.013820 | 0.471565 | 1.00 | 1.000000 | 10.019148 | 0 | 0.955096575 | false |
| noise_2 | 4 | 10.105783 | 0.099057 | 0.547410 | 1.00 | 1.000000 | 10.006727 | 0 | 0.954950944 | false |
| noise_1 | 5 | 10.032269 | 0.081796 | 0.511666 | 1.00 | 1.000000 | 9.950473 | 0 | 0.954283818 | false |
| noise_0 | 6 | 9.986730 | 0.075775 | 0.527613 | 1.00 | 1.000000 | 9.910955 | 0 | 0.953807636 | false |
| noise_5 | 7 | 10.045978 | 0.146156 | 0.516431 | 1.00 | 1.000000 | 9.899822 | 0 | 0.953672342 | false |
| noise_4 | 8 | 9.866966 | 0.002777 | 0.499575 | 1.00 | 1.000000 | 9.864189 | 0 | 0.953235940 | false |
| gap | 9 | 0.519951 | 0.796835 | 0.072530 | 0.01 | 0.107532 | 2.574888 | 0 | 0.570467969 | false |
| previous_age | 10 | 2.000000 | 1.000000 | 0.000000 | 1.00 | 1.000000 | 1.000000 | 0 | 0.280689585 | false |
| chosen.option | — | N/A | N/A | N/A | N/A | N/A | N/A | N/A | 0.000000000 | N/A |
| reason_code | — | N/A | N/A | N/A | N/A | N/A | N/A | N/A | 0.000000000 | N/A |
| verdict | — | N/A | N/A | N/A | N/A | N/A | N/A | N/A | 0.000000000 | N/A |

Exact ties (including unranked zero observations): [[chosen.option reason_code verdict]]. Smallest distinct score gap: 2.51980063702e-05.

output.lane raw P/R categories: [map[left:100] map[right:100]].

chosen.option raw P/R categories: [map[returned:100] map[returned:100]].

reason_code raw P/R categories: [map[:100] map[:100]].

verdict raw P/R categories: [map[returned:100] map[returned:100]].

Chronological 50-frame bins P/R: [[28 22] [21 29] [24 26] [27 23]]. Session P/R counts and mean noise (averaged over seven fields):

| Session | P/R | P noise | R noise |
|---|---|---:|---:|
| lane-01-4051-decoy-heavy-session-0 | [14 6] | 9.966010 | -0.011515 |
| lane-01-4051-decoy-heavy-session-1 | [7 13] | 9.979009 | 0.124913 |
| lane-01-4051-decoy-heavy-session-2 | [12 8] | 9.910998 | 0.012729 |
| lane-01-4051-decoy-heavy-session-3 | [6 14] | 10.021662 | -0.059113 |
| lane-01-4051-decoy-heavy-session-4 | [10 10] | 10.040281 | -0.007828 |
| lane-01-4051-decoy-heavy-session-5 | [7 13] | 9.857870 | -0.058123 |
| lane-01-4051-decoy-heavy-session-6 | [11 9] | 10.000587 | 0.104016 |
| lane-01-4051-decoy-heavy-session-7 | [11 9] | 10.002651 | -0.040076 |
| lane-01-4051-decoy-heavy-session-8 | [11 9] | 10.008767 | 0.107659 |
| lane-01-4051-decoy-heavy-session-9 | [11 9] | 10.057792 | -0.004355 |

## harder / 4051

| Field | Rank | P median | R median | MAD | epsilon | scale | z | e | Score | Floor exit |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---|
| output.lane | 1 | N/A | N/A | N/A | N/A | N/A | N/A | N/A | 0.980392157 | N/A |
| gap | 2 | 0.517727 | 0.785718 | 0.081045 | 0.01 | 0.120157 | 2.230344 | 0 | 0.519385877 | false |
| noise_1 | 3 | 0.205678 | -0.158429 | 0.492687 | 1.00 | 1.000000 | 0.364107 | 34 | 0.075074730 | false |
| noise_5 | 4 | -0.156271 | 0.054999 | 0.520532 | 1.00 | 1.000000 | 0.211270 | 40 | 0.040667190 | false |
| noise_6 | 5 | 0.134413 | -0.032859 | 0.509631 | 1.00 | 1.000000 | 0.167272 | 39 | 0.032964127 | false |
| noise_3 | 6 | 0.017464 | 0.121295 | 0.512857 | 1.00 | 1.000000 | 0.103832 | 44 | 0.019010305 | false |
| chosen.option | — | N/A | N/A | N/A | N/A | N/A | N/A | N/A | 0.000000000 | N/A |
| reason_code | — | N/A | N/A | N/A | N/A | N/A | N/A | N/A | 0.000000000 | N/A |
| verdict | — | N/A | N/A | N/A | N/A | N/A | N/A | N/A | 0.000000000 | N/A |
| noise_0 | — | -0.054495 | -0.052765 | 0.515810 | 1.00 | 1.000000 | 0.001730 | 50 | 0.000000000 | true |
| noise_2 | — | 0.126017 | 0.123762 | 0.536834 | 1.00 | 1.000000 | 0.002255 | 50 | 0.000000000 | true |
| noise_4 | — | -0.072342 | 0.021179 | 0.472238 | 1.00 | 1.000000 | 0.093521 | 46 | 0.000000000 | true |
| previous_age | — | 1.000000 | 1.000000 | 0.000000 | 1.00 | 1.000000 | 0.000000 | 100 | 0.000000000 | true |

Exact ties (including unranked zero observations): [[chosen.option reason_code verdict noise_0 noise_2 noise_4 previous_age]]. Smallest distinct score gap: 0.00770306266535.

output.lane raw P/R categories: [map[left:100] map[right:100]].

chosen.option raw P/R categories: [map[returned:100] map[returned:100]].

reason_code raw P/R categories: [map[:100] map[:100]].

verdict raw P/R categories: [map[returned:100] map[returned:100]].

Chronological 50-frame bins P/R: [[23 27] [21 29] [30 20] [26 24]]. Session P/R counts and mean noise (averaged over seven fields):

| Session | P/R | P noise | R noise |
|---|---|---:|---:|
| lane-01-4051-harder-session-0 | [10 10] | 0.090206 | 0.024048 |
| lane-01-4051-harder-session-1 | [9 11] | 0.074116 | -0.061178 |
| lane-01-4051-harder-session-2 | [8 12] | 0.070974 | 0.033549 |
| lane-01-4051-harder-session-3 | [7 13] | 0.103245 | 0.108287 |
| lane-01-4051-harder-session-4 | [10 10] | -0.029939 | -0.038665 |
| lane-01-4051-harder-session-5 | [14 6] | 0.000005 | 0.048554 |
| lane-01-4051-harder-session-6 | [10 10] | 0.003532 | -0.116753 |
| lane-01-4051-harder-session-7 | [10 10] | -0.143406 | -0.116248 |
| lane-01-4051-harder-session-8 | [11 9] | -0.099418 | 0.100115 |
| lane-01-4051-harder-session-9 | [11 9] | 0.106580 | -0.008233 |

## decoy-heavy / 5059

| Field | Rank | P median | R median | MAD | epsilon | scale | z | e | Score | Floor exit |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---|
| output.lane | 1 | N/A | N/A | N/A | N/A | N/A | N/A | N/A | 0.980392157 | N/A |
| noise_2 | 2 | 10.104002 | 0.020049 | 0.462640 | 1.00 | 1.000000 | 10.083953 | 0 | 0.955846651 | false |
| noise_3 | 3 | 10.043541 | -0.026021 | 0.455495 | 1.00 | 1.000000 | 10.069562 | 0 | 0.955681482 | false |
| noise_5 | 4 | 10.082702 | 0.016671 | 0.498364 | 1.00 | 1.000000 | 10.066032 | 0 | 0.955640844 | false |
| noise_4 | 5 | 9.985996 | 0.010244 | 0.512731 | 1.00 | 1.000000 | 9.975752 | 0 | 0.954585160 | false |
| noise_6 | 6 | 9.989833 | 0.046208 | 0.492974 | 1.00 | 1.000000 | 9.943625 | 0 | 0.954201753 | false |
| noise_0 | 7 | 9.996860 | 0.057152 | 0.491488 | 1.00 | 1.000000 | 9.939709 | 0 | 0.954154733 | false |
| noise_1 | 8 | 10.014452 | 0.112635 | 0.445808 | 1.00 | 1.000000 | 9.901816 | 0 | 0.953696617 | false |
| gap | 9 | 0.520808 | 0.796847 | 0.074334 | 0.01 | 0.110207 | 2.504740 | 0 | 0.560538002 | false |
| previous_age | 10 | 2.000000 | 1.000000 | 0.000000 | 1.00 | 1.000000 | 1.000000 | 0 | 0.280689585 | false |
| chosen.option | — | N/A | N/A | N/A | N/A | N/A | N/A | N/A | 0.000000000 | N/A |
| reason_code | — | N/A | N/A | N/A | N/A | N/A | N/A | N/A | 0.000000000 | N/A |
| verdict | — | N/A | N/A | N/A | N/A | N/A | N/A | N/A | 0.000000000 | N/A |

Exact ties (including unranked zero observations): [[chosen.option reason_code verdict]]. Smallest distinct score gap: 4.06380553843e-05.

output.lane raw P/R categories: [map[left:100] map[right:100]].

chosen.option raw P/R categories: [map[returned:100] map[returned:100]].

reason_code raw P/R categories: [map[:100] map[:100]].

verdict raw P/R categories: [map[returned:100] map[returned:100]].

Chronological 50-frame bins P/R: [[25 25] [30 20] [20 30] [25 25]]. Session P/R counts and mean noise (averaged over seven fields):

| Session | P/R | P noise | R noise |
|---|---|---:|---:|
| lane-01-5059-decoy-heavy-session-0 | [11 9] | 10.043843 | -0.063117 |
| lane-01-5059-decoy-heavy-session-1 | [9 11] | 9.991631 | 0.083469 |
| lane-01-5059-decoy-heavy-session-2 | [12 8] | 10.042782 | -0.025675 |
| lane-01-5059-decoy-heavy-session-3 | [14 6] | 9.980913 | 0.035008 |
| lane-01-5059-decoy-heavy-session-4 | [9 11] | 10.086130 | 0.038406 |
| lane-01-5059-decoy-heavy-session-5 | [8 12] | 10.011122 | -0.066070 |
| lane-01-5059-decoy-heavy-session-6 | [7 13] | 10.114049 | 0.016060 |
| lane-01-5059-decoy-heavy-session-7 | [10 10] | 10.093730 | 0.041438 |
| lane-01-5059-decoy-heavy-session-8 | [12 8] | 9.901568 | -0.090247 |
| lane-01-5059-decoy-heavy-session-9 | [8 12] | 9.987814 | 0.163271 |

## harder / 5059

| Field | Rank | P median | R median | MAD | epsilon | scale | z | e | Score | Floor exit |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---|
| output.lane | 1 | N/A | N/A | N/A | N/A | N/A | N/A | N/A | 0.980392157 | N/A |
| gap | 2 | 0.518912 | 0.782825 | 0.070153 | 0.01 | 0.104009 | 2.537406 | 0 | 0.565191123 | false |
| noise_5 | 3 | 0.257080 | -0.058392 | 0.491094 | 1.00 | 1.000000 | 0.315473 | 35 | 0.064587646 | false |
| noise_6 | 4 | 0.076291 | -0.138510 | 0.454140 | 1.00 | 1.000000 | 0.214800 | 39 | 0.042000081 | false |
| noise_4 | 5 | 0.019654 | -0.123521 | 0.554762 | 1.00 | 1.000000 | 0.143176 | 43 | 0.026500441 | false |
| noise_2 | 6 | -0.163061 | -0.042256 | 0.547004 | 1.00 | 1.000000 | 0.120805 | 45 | 0.021668857 | false |
| chosen.option | — | N/A | N/A | N/A | N/A | N/A | N/A | N/A | 0.000000000 | N/A |
| reason_code | — | N/A | N/A | N/A | N/A | N/A | N/A | N/A | 0.000000000 | N/A |
| verdict | — | N/A | N/A | N/A | N/A | N/A | N/A | N/A | 0.000000000 | N/A |
| noise_0 | — | -0.094707 | 0.002657 | 0.446569 | 1.00 | 1.000000 | 0.097364 | 45 | 0.000000000 | true |
| noise_1 | — | 0.080139 | 0.178915 | 0.402128 | 1.00 | 1.000000 | 0.098776 | 47 | 0.000000000 | true |
| noise_3 | — | 0.112285 | 0.041863 | 0.587449 | 1.00 | 1.000000 | 0.070423 | 46 | 0.000000000 | true |
| previous_age | — | 1.000000 | 1.000000 | 0.000000 | 1.00 | 1.000000 | 0.000000 | 100 | 0.000000000 | true |

Exact ties (including unranked zero observations): [[chosen.option reason_code verdict noise_0 noise_1 noise_3 previous_age]]. Smallest distinct score gap: 0.00483158464893.

output.lane raw P/R categories: [map[left:100] map[right:100]].

chosen.option raw P/R categories: [map[returned:100] map[returned:100]].

reason_code raw P/R categories: [map[:100] map[:100]].

verdict raw P/R categories: [map[returned:100] map[returned:100]].

Chronological 50-frame bins P/R: [[23 27] [28 22] [26 24] [23 27]]. Session P/R counts and mean noise (averaged over seven fields):

| Session | P/R | P noise | R noise |
|---|---|---:|---:|
| lane-01-5059-harder-session-0 | [9 11] | 0.062591 | -0.089696 |
| lane-01-5059-harder-session-1 | [10 10] | 0.032825 | -0.032439 |
| lane-01-5059-harder-session-2 | [10 10] | 0.038676 | -0.025373 |
| lane-01-5059-harder-session-3 | [13 7] | -0.080634 | 0.069849 |
| lane-01-5059-harder-session-4 | [9 11] | -0.070631 | 0.010230 |
| lane-01-5059-harder-session-5 | [9 11] | 0.010719 | -0.021442 |
| lane-01-5059-harder-session-6 | [12 8] | 0.084766 | 0.096712 |
| lane-01-5059-harder-session-7 | [8 12] | 0.018648 | -0.024833 |
| lane-01-5059-harder-session-8 | [7 13] | 0.044001 | -0.081584 |
| lane-01-5059-harder-session-9 | [13 7] | -0.028649 | 0.019122 |
