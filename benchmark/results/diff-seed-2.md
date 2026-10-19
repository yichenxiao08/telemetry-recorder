# M3 shop diff validation: seed 2

Generated from `benchmark/results/shop-demo-seed-2.json`; database `.cache/shop-demo/run-3490384829/signalscout.db`. Engine revision `8b6d48dea871db617a92e02f0991e94812c9f268` (dirty=false); demo revision `feaff2809898353dc3943dbbdeb46d3e8a47ea61`. Scoring v1 was fixed before ranking.

Representative: latest failing run `c354999411908c27ac1a86edae05c8d2`; flat DB_POOL_SIZE=5. Failure-phase cutoff: `2026-10-19T16:22:37.961361836Z`.

All 18 comparisons select 40 baselines. Maximum elapsed time including selection, evidence loading, extraction, scoring, persistence and connection closure: **41.281 ms**. Compilation/process startup excluded. Acceptance: **true**.

| Run | Mode | Pool rank | Retry rank | Acquire rank | Cache-miss rank | Baseline cache misses | Time ms | Window accepted |
|---|---|---:|---:|---:|---:|---:|---:|---|
| c354999411908c27ac1a86edae05c8d2 | before | 1 | 2 | 3 | 4 | 20/40 | 41.281 | true |
| c354999411908c27ac1a86edae05c8d2 | default | 0 | 1 | 2 | 3 | 15/40 | 18.037 | n/a |
| 0ab20f66e6a76a3f0cf14a8d67c53970 | before | 1 | 2 | 3 | 4 | 20/40 | 21.471 | true |
| 0ab20f66e6a76a3f0cf14a8d67c53970 | default | 0 | 1 | 2 | 3 | 15/40 | 20.900 | n/a |
| cba01f083bafe7e557b0c71aed950409 | before | 1 | 2 | 3 | 4 | 20/40 | 22.261 | true |
| cba01f083bafe7e557b0c71aed950409 | default | 0 | 1 | 2 | 3 | 15/40 | 21.759 | n/a |
| 4b11f1f120518a8bfadb57b27bf1ad64 | before | 1 | 2 | 3 | 4 | 20/40 | 17.418 | true |
| 4b11f1f120518a8bfadb57b27bf1ad64 | default | 0 | 1 | 2 | 3 | 21/40 | 23.045 | n/a |
| 47c2f2f18321f8d54544a5379aa794f0 | before | 1 | 2 | 3 | 4 | 20/40 | 22.087 | true |
| 47c2f2f18321f8d54544a5379aa794f0 | default | 0 | 1 | 2 | 3 | 21/40 | 26.201 | n/a |
| 7efe8b040333201354eb6379b0359827 | before | 1 | 2 | 3 | 4 | 20/40 | 26.450 | true |
| 7efe8b040333201354eb6379b0359827 | default | 0 | 1 | 2 | 3 | 21/40 | 24.178 | n/a |
| 13bd48578c81be3dcc15ca14eeed4154 | before | 1 | 2 | 3 | 4 | 20/40 | 20.791 | true |
| 13bd48578c81be3dcc15ca14eeed4154 | default | 3 | 1 | 2 | 4 | 20/40 | 13.947 | n/a |
| 17d96dab208e29e6d826a49682063940 | before | 1 | 2 | 3 | 4 | 20/40 | 12.946 | true |
| 17d96dab208e29e6d826a49682063940 | default | 3 | 1 | 2 | 4 | 20/40 | 16.689 | n/a |
| 3e41383e8cbf1c3c4d1f3717e73083a8 | before | 1 | 2 | 3 | 4 | 20/40 | 16.398 | true |
| 3e41383e8cbf1c3c4d1f3717e73083a8 | default | 3 | 1 | 2 | 4 | 20/40 | 15.284 | n/a |

Rank 0 means absent from the ranked output. Window acceptance requires the three requested differences in the top 3, a real cache-miss decoy below them, and time below one second. Default mode is reported without requiring a pool-size difference.

## Representative: before

Baseline pool sizes: `{"20":40}`; cache misses 20/40.

| Rank | Type | Feature | Score | Raw baseline rate |
|---:|---|---|---:|---:|
| 1 | config | DB_POOL_SIZE | 1.000000 | 1.0000 |
| 2 | presence | retry | 0.976190 | 0.0000 |
| 3 | latency | db.pool.acquire | 0.954545 | 0.0000 |
| 4 | presence | cache.miss | 0.500000 | 0.5000 |
| 5 | presence | db.pool.acquire | 0.500000 | 0.5000 |
| 6 | count | db.pool.acquire | 0.384101 | 0.0000 |

## Representative: default

Baseline pool sizes: `{"5":40}`; cache misses 15/40.

| Rank | Type | Feature | Score | Raw baseline rate |
|---:|---|---|---:|---:|
| 1 | presence | retry | 0.976190 | 0.0000 |
| 2 | latency | db.pool.acquire | 0.940841 | 0.0000 |
| 3 | presence | cache.miss | 0.619048 | 0.3750 |
| 4 | presence | db.pool.acquire | 0.619048 | 0.3750 |
| 5 | count | db.pool.acquire | 0.474998 | 0.0000 |
| 6 | count | cache.hit | 0.175481 | 0.3750 |

The latest failure's pool-size snapshot agrees with its recent passing baseline mode, so the pool-size config difference correctly disappears.

The explicit window uses the pre-change phase. Cache misses and pool acquisitions also occur in passing requests; the tables retain their raw baseline rates and ranks. Earlier failures can have a mixed default baseline and a pool-size difference. No context metric or context log differences are computed.
