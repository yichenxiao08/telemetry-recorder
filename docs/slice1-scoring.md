# Slice 1 scoring contract (frozen before evaluation)

Scope: synthetic lane-01; independent module, no changes to M3/M4b scoring.
Use problem-model.md's categorical Laplace total variation and numeric median
shift with MAD factor 1.4826, squash 3, exceedance smoothing +1/+2. Ten observed
runs in EACH cohort are required per feature; missing capture is unknown.
No feature name receives a boost. Identical distributions score zero. Numeric
scores below a profile-declared absolute floor are zero. Empty categorical
strings are explicit values; missing selectors are excluded, never empty strings.
Sort score descending, then type lexicographically, then stable feature key.

Profiles declare step, unit, finite units/epsilon/floor for each numeric selector,
and categorical selectors (verdict, reason_code, chosen.option, optional returned
categories). They may declare a subtraction of two captured scalars. This is
needed for a version age rather than comparing absolute frame indices from
different times; it is a declared domain transform, not discovered causality.
The generic engine contains no lane/threshold/staleness/name-specific logic.
The one declared-used version invariant is parameterized by capture field names
and expected delta. It needs no reference, and reports missing capture as unknown.
Only observation and association report tiers exist in this slice.

Cohort selection pins IDs and label revisions under an explicit bounded time
window/limit. It requires declared unit/profile/workload consistency and an
explicit reference selection. Keep disagreeing reviewers; unknown labels cannot
become references. Reject overlap and ambiguous active labels for label selection;
explicit run sets can include disputed marks but must record that policy.
Predicate support is a bounded captured-input equality predicate, not SQL or a
query language. Labels use a dimension/value predicate. Pin exact exclusions,
selected label revisions, capture counts and session counts. No population claims.

lane-01 seeds, distributions, roles, transforms, epsilons/floors and hard variants
are fixed in problem.yaml before execution. The fixture is transparently authored
with knowledge of the requested mechanisms; authored_blind means evaluation
outcomes unseen, not an independent blind benchmark. Keep every miss, including
absent median effects and decoys ranked above mechanisms. Existing source audit
results and shop regressions remain unchanged; no production privacy is claimed.
