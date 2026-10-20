# M3 shop diff validation: seed 1

Generated from `.cache/m4b/seed-1-regression-demo.json`; database `.cache/m4b/seed-1.db`. Engine revision `fd19eece879dadc1fc9e3ace91397a3fba75559c` (dirty=false); demo revision `e0097edea656a73432c9e784d30ad2d22516efa0`. Scoring v1 was fixed before ranking.

Representative: latest failing run `c779cc44e52e5af434d9ce3e2d2b465f`; flat DB_POOL_SIZE=5. Failure-phase cutoff: `2026-10-19T08:57:48.408403076Z`.

All 18 comparisons select 40 baselines. Maximum elapsed time including selection, evidence loading, extraction, scoring, persistence and connection closure: **19.296 ms**. Compilation/process startup excluded. Acceptance: **true**.

| Run | Mode | Pool rank | Retry rank | Acquire rank | Cache-miss rank | Baseline cache misses | Time ms | Window accepted |
|---|---|---:|---:|---:|---:|---:|---:|---|
| c779cc44e52e5af434d9ce3e2d2b465f | before | 1 | 2 | 3 | 4 | 16/40 | 14.582 | true |
| c779cc44e52e5af434d9ce3e2d2b465f | default | 0 | 1 | 2 | 3 | 18/40 | 13.860 | n/a |
| bacb3e98faf20115e0bbdeff5ecc9ff1 | before | 1 | 2 | 3 | 4 | 16/40 | 12.396 | true |
| bacb3e98faf20115e0bbdeff5ecc9ff1 | default | 0 | 1 | 2 | 3 | 18/40 | 14.381 | n/a |
| da0cd0313bb5c6da62732e5302c5995a | before | 1 | 2 | 3 | 4 | 16/40 | 13.816 | true |
| da0cd0313bb5c6da62732e5302c5995a | default | 0 | 1 | 2 | 3 | 18/40 | 13.239 | n/a |
| b89b39104d5d27232e3c404ef8a8b0db | before | 1 | 2 | 3 | 4 | 16/40 | 13.271 | true |
| b89b39104d5d27232e3c404ef8a8b0db | default | 0 | 1 | 2 | 3 | 13/40 | 19.296 | n/a |
| 521a541bb362b93d55272760cf86dc3f | before | 1 | 2 | 3 | 4 | 16/40 | 10.987 | true |
| 521a541bb362b93d55272760cf86dc3f | default | 0 | 1 | 2 | 3 | 13/40 | 14.711 | n/a |
| b98ef50b5ee825243baa0c2e2b56bb32 | before | 1 | 2 | 3 | 4 | 16/40 | 15.840 | true |
| b98ef50b5ee825243baa0c2e2b56bb32 | default | 0 | 1 | 2 | 3 | 13/40 | 10.275 | n/a |
| 39815e2a82e0fa4227612d242db60679 | before | 1 | 2 | 3 | 4 | 16/40 | 8.805 | true |
| 39815e2a82e0fa4227612d242db60679 | default | 3 | 1 | 2 | 4 | 16/40 | 12.511 | n/a |
| 100daf6b520c3fe4bac166cb49bf89dd | before | 1 | 2 | 3 | 4 | 16/40 | 14.392 | true |
| 100daf6b520c3fe4bac166cb49bf89dd | default | 3 | 1 | 2 | 4 | 16/40 | 6.571 | n/a |
| 78689eba5892f10825d2315d2e4c174e | before | 1 | 2 | 3 | 4 | 16/40 | 11.518 | true |
| 78689eba5892f10825d2315d2e4c174e | default | 3 | 1 | 2 | 4 | 16/40 | 12.712 | n/a |

Rank 0 means absent from the ranked output. Window acceptance requires the three requested differences in the top 3, a real cache-miss decoy below them, and time below one second. Default mode is reported without requiring a pool-size difference.

## Representative: before

Baseline pool sizes: `{"20":40}`; cache misses 16/40.

| Rank | Type | Feature | Score | Raw baseline rate |
|---:|---|---|---:|---:|
| 1 | config | DB_POOL_SIZE | 1.000000 | 1.0000 |
| 2 | presence | retry | 0.976190 | 0.0000 |
| 3 | latency | db.pool.acquire | 0.944444 | 0.0000 |
| 4 | presence | cache.miss | 0.595238 | 0.4000 |
| 5 | presence | db.pool.acquire | 0.595238 | 0.4000 |
| 6 | count | db.pool.acquire | 0.474998 | 0.0000 |
| 7 | count | cache.hit | 0.168731 | 0.4000 |

## Representative: default

Baseline pool sizes: `{"5":40}`; cache misses 18/40.

| Rank | Type | Feature | Score | Raw baseline rate |
|---:|---|---|---:|---:|
| 1 | presence | retry | 0.976190 | 0.0000 |
| 2 | latency | db.pool.acquire | 0.945724 | 0.0000 |
| 3 | presence | cache.miss | 0.547619 | 0.4500 |
| 4 | presence | db.pool.acquire | 0.547619 | 0.4500 |
| 5 | count | db.pool.acquire | 0.474998 | 0.0000 |
| 6 | count | cache.hit | 0.155233 | 0.4500 |

The latest failure's pool-size snapshot agrees with its recent passing baseline mode, so the pool-size config difference correctly disappears.

The explicit window uses the pre-change phase. Cache misses and pool acquisitions also occur in passing requests; the tables retain their raw baseline rates and ranks. Earlier failures can have a mixed default baseline and a pool-size difference. No context metric or context log differences are computed.
