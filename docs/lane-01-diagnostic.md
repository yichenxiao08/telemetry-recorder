# Lane-01 missed-mechanism diagnostic (2026-10-20)

## Boundary and reproduction

The refreshed remote main is 859487c. Slice 1 tip 05e9add is **not merged**
into it; codex/lane-diagnostic branches from codex/slice1-lane as authorized.
This is synthetic analysis and design, not M5–M8, P0 or a new engine release.
All original lane-01 files, profiles, constants and results remain unchanged.

From the repository root in native PowerShell:

```powershell
go run ./scripts/lanediagnostic --db .cache/lane-01/eval-002/fresh.db
```

The saved database is required; the script does not regenerate the experiment.
It opens only the existing read-only reader, resolves the immutable cohorts'
pinned decision revisions, and requires **complete report equality** with the
ten corresponding cases in benchmark/results/lane-01.json before printing
new diagnostic artifacts. It calls the unchanged scorer to verify each numeric
score. No new seed or proposed method is executed. The original first run and
confirmation remain distinct historical measurements.

Full per-field tables (all 13 fields, all ten cases), raw categorical counts,
ties, session counts and time blocks are in
[lane-01-diagnostic.md](../benchmark/results/lane-01-diagnostic.md).
[The JSON](../benchmark/results/lane-01-diagnostic.json) retains full precision,
numeric support ranges, squash and exceedance factors, and observed decision
timestamp ranges per cohort/session. Categorical medians, MAD, z and e are N/A;
inventing numeric equivalents would misrepresent the categorical TV formula.

## What failed, and why

There are 100 fully observed runs on each side in every case, ten shared session
IDs, zero missing feature values. Session IDs are constructed clusters, not
proof of independent trajectories. The seven decoys are noise_0 through noise_6.
The constant verdict, reason_code and chosen.option all score zero. The output
symptom is left in 100/100 problem examples versus 0/100 references, and always
scores 100/102 = 0.980392157, rank 1.

### Decoy-heavy: classifications (a), with a fixture-selection limitation (d)

Both mechanisms miss top 3/top 5 in **every** seed: gap is #9, previous_age #10.
The seven decoys occupy #2–8. Across all 35 decoy observations, z is
9.660699–10.183606 and score 0.950641950–0.956968918. Their reference MAD is
below epsilon/1.4826, so **epsilon=1 determines scale**, not a hidden change to
MAD. Their supports are disjoint: problem [9,11], reference [-1,1]. These are
real marginal separations in the generated data, despite irrelevance to the
target's computation.

Gap z by seed 1009/2027/3037/4051/5059 is
3.333948/2.499025/2.936005/2.574888/2.504740; scores are
0.664297295/0.559718809/0.618069241/0.570467969/0.560538002.
Previous age has problem median 2, reference median 1, MAD 0, epsilon/scale 1,
z=1, e=0 and score 0.280689585 in every seed. This verifies **(a)** for each of
the ten mechanism misses: the marginal decoy shifts are larger on the
profile's declared scales. Those scales encode measurement resolution and
are not a common causal-importance scale across arbitrary units.

**Formula saturation (b) exists but is not the cause of these rank misses.**
Every heavy-case numeric field has e=0 and the same exceedance factor 101/102.
The exponential squash is strictly increasing for these finite z values; it
compresses the decoys close to 0.95 but cannot reverse their ordering relative
to gap or age. There is no hard squash cap or exact positive-score tie in any
of these cases. The smallest positive score separations per seed are
0.000052018872/0.000013568720/0.000164806767/0.000025198006/0.000040638055.
Only the three constant categorical observations tie at zero. A rounded UI
could visually conceal differences; lexical tie-breaking did not decide these
mechanism ranks. The output TV and numeric scores are different association
summaries, not calibrated probabilities of a cause; rank 1 is additionally
a symptom-presentation limitation, **(d)**.

### Harder: classification (c), a verified statistic/formula miss

For **all five seeds**, gap is #2 with scores respectively
0.575471585/0.602434264/0.581381748/0.519385877/0.565191123; it does not miss
top 3 or top 5. Previous_age is unranked, score zero: problem has 80 ones and
20 twos, reference 100 ones. Both medians are 1, MAD=0, scale=epsilon=1, z=0.
This is **(c)** for each of the five age misses. The existing invariant already
exposes 20/100 declared-version violations; it does not prove actual reads.
There is observable subgroup information that the median throws away.
This miss is attributable to the chosen statistic/formula, not absent capture.

For diagnostic completeness directional e would be 100 (ties included), but
the real scorer exits at effect=0 < floor=1 **before** computing e. The floor,
exceedance smoothing and saturation are therefore not separate explanations
for hiding this subgroup. All unranked features share score zero; stable sort
order does not convert them into discoveries. Some independent-noise medians
cross the fixed 0.1 floor by chance: 3/2/1/4/4 positive decoys by seed. Their
scores are 0.019010305–0.075074730, above age's zero but below gap. This is
finite-sample marginal association **(a)**, not demonstrated computational use.
There is no multiple-testing significance claim in these scores. The full
table lists each positive decoy and every below-floor zero, including their e.

## Is decoy-heavy fair?

The fixture generator first constructs problem/reference inputs separately,
adding +10 to **all seven** problem noise fields, then shuffles and derives
labels from the returned lane. The target never reads noise. Thus the oracle
labels are not passed to the function, but selection is deliberately entangled
with nuisance inputs. This is a **cohort-construction artifact**, not a genuine
uncontrolled workload discovery. Even gap and previous_lane are constructed
to separate outputs; the supplied previous version has no verified historical
origin. In harder, stale age is an associated minority mechanism label, not
proof that removing staleness alone fixes a left output while the prior lane
and gate remain unchanged.

More precisely, the plain target uses gap and previous_lane to choose lane;
previous_frame is only copied to previous_version in its return metadata.
Changing that version alone cannot change lane in this target. The stale-age
label is therefore a declared freshness violation/associated marker, not an
experimentally established cause of the lane choice. This additional **(d)**
fixture-validity limitation applies in all seeds of both variants. It does not
erase the separate mathematical finding that the median hides a recorded
minority, nor justify counting its future detection as demonstrated causal
diagnosis. No counterfactual target run is needed for this source-level fact;
the diagnostic executes only saved-data comparisons.

The saved rows rule out the suggested *different sessions/time ranges* explanation:
both labels occur in all ten sessions in every case, 6–14 examples per label
per session. Both labels occur in every consecutive 50-frame time block
(20–30 per label). All cases use one shared bounded selection window and
shuffled frame order; per-session frame/time ranges and counts are retained.
Heavy-case noise stays near 10 versus 0 **within** every session; pooled session
imbalance cannot explain away the shift. Frame-order bins describe execution
order, not an independent time-effect experiment. No unmatched earlier
reference phase is needed to produce this artifact.

**Verdict:** fair as an adversarial test showing the limits of marginal
associations; unfair as an acceptance demand that marginal data alone rank the
unused decoys below actual mechanisms. Stronger marginal contrast is not causal
relevance. Matching on session alone will not solve it. Conditioning on the
fully separating gate has no common support, so an honest conditional method
must sometimes answer unsupported rather than promise to recover the causes.

## Recommendation

Review the two narrowly proposed additions and symptom roles in
[problem-model.md](problem-model.md#proposed-v1-analysis-only-not-authorized-for-implementation).
Prioritize minority coverage and honest overlap diagnostics; do not tune constants
or claim a conditional comparison solves nonidentifiability. Keep the frozen
lane-01 evaluation as a retained miss. Obtain approval for a fresh preregistration
and implementation before running the reserved seeds. Real-world accuracy and
independent histories remain unestablished. No new difference type, migration,
production engine logic or scoring constant is included in this task.
