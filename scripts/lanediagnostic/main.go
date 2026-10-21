// Command lanediagnostic reads frozen synthetic lane-01 evidence; it never writes
// the store or evaluates a proposed scoring method.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"

	"github.com/signalscout/signalscout/internal/cohort"
	"github.com/signalscout/signalscout/internal/model"
	"github.com/signalscout/signalscout/internal/provenance"
	"github.com/signalscout/signalscout/internal/stats"
	"github.com/signalscout/signalscout/internal/store"
)

type terms struct {
	ProblemMedian    float64 `json:"problem_median"`
	ReferenceMedian  float64 `json:"reference_median"`
	MAD              float64 `json:"mad"`
	Epsilon          float64 `json:"epsilon"`
	Scale            float64 `json:"scale"`
	Z                float64 `json:"z"`
	E                int     `json:"e"`
	Squash           float64 `json:"squash"`
	ExceedanceFactor float64 `json:"exceedance_factor"`
	BelowFloor       bool    `json:"below_floor"`
	Score            float64 `json:"score"`
	ProblemMin       float64 `json:"problem_min"`
	ProblemMax       float64 `json:"problem_max"`
	ReferenceMin     float64 `json:"reference_min"`
	ReferenceMax     float64 `json:"reference_max"`
}

func numeric(p, r []float64, epsilon, floor float64) terms {
	x, m := stats.Median(p), stats.Median(r)
	t := terms{ProblemMedian: x, ReferenceMedian: m, MAD: stats.MAD(r), Epsilon: epsilon}
	t.Scale = math.Max(1.4826*t.MAD, epsilon)
	t.Z = math.Abs(x-m) / t.Scale
	for _, v := range r {
		if x >= m && v >= x || x < m && v <= x {
			t.E++
		}
	}
	t.Squash = 1 - math.Exp(-t.Z/3)
	t.ExceedanceFactor = 1 - float64(t.E+1)/float64(len(r)+2)
	t.BelowFloor = math.Abs(x-m) < floor
	t.Score = cohort.Numeric(p, r, epsilon, floor)
	t.ProblemMin, t.ProblemMax = extrema(p)
	t.ReferenceMin, t.ReferenceMax = extrema(r)
	return t
}
func extrema(v []float64) (float64, float64) {
	lo, hi := v[0], v[0]
	for _, x := range v {
		lo = math.Min(lo, x)
		hi = math.Max(hi, x)
	}
	return lo, hi
}
func values(xs []model.DecisionObservation, f model.NumericFeature) ([]float64, error) {
	out := []float64{}
	for _, x := range xs {
		a, ok := provenance.Get(x.Values, f.Selector)
		v, valid := a.(float64)
		if !ok || !valid {
			return nil, fmt.Errorf("missing numeric %s", f.Name)
		}
		if f.Subtract != "" {
			b, ok := provenance.Get(x.Values, f.Subtract)
			sub, valid := b.(float64)
			if !ok || !valid {
				return nil, fmt.Errorf("missing subtraction")
			}
			v -= sub
		}
		out = append(out, v)
	}
	return out, nil
}

type row struct {
	Feature    string           `json:"feature"`
	Rank       int              `json:"rank_zero_is_unranked"`
	Status     string           `json:"status"`
	Score      float64          `json:"score"`
	Terms      *terms           `json:"numeric,omitempty"`
	Categories []map[string]int `json:"categories_problem_reference,omitempty"`
}
type session struct {
	ID        string     `json:"id"`
	Counts    [2]int     `json:"problem_reference_counts"`
	FrameMin  [2]int     `json:"frame_min"`
	FrameMax  [2]int     `json:"frame_max"`
	NoiseMean [2]float64 `json:"noise_mean_all_seven"`
	TimeMin   [2]int64   `json:"decision_time_min_ns"`
	TimeMax   [2]int64   `json:"decision_time_max_ns"`
}
type diagnosis struct {
	Seed                    int        `json:"seed"`
	Variant                 string     `json:"variant"`
	Rows                    []row      `json:"fields"`
	Sessions                []session  `json:"sessions"`
	TimeBins                [4][2]int  `json:"consecutive_50_frame_bins_problem_reference"`
	ExactTies               [][]string `json:"exact_score_ties_including_unranked"`
	MinimumPositiveScoreGap float64    `json:"minimum_positive_score_gap"`
}

func analyze(ctx context.Context, r *store.Reader, seed int, variant string, frozen cohort.Result) (diagnosis, error) {
	d := diagnosis{Seed: seed, Variant: variant, Rows: []row{}, Sessions: []session{}, ExactTies: [][]string{}}
	cs, sets, err := r.CohortPair(ctx, frozen.Problem.ID, frozen.Reference.ID)
	if err != nil {
		return d, err
	}
	got, err := cohort.Compare(cs[0], cs[1], sets[0], sets[1])
	if err != nil {
		return d, err
	}
	a, _ := json.Marshal(got)
	b, _ := json.Marshal(frozen)
	if string(a) != string(b) {
		return d, fmt.Errorf("frozen report changed for %d %s", seed, variant)
	}
	rank := 0
	for _, f := range got.Differences {
		x := row{Feature: f.Feature, Score: f.Score, Status: f.Status}
		if f.Status == "ok" && f.Score > 0 {
			rank++
			x.Rank = rank
		}
		for _, n := range cs[0].Profile.Numeric {
			if n.Name == f.Feature {
				p, e := values(sets[0], n)
				if e != nil {
					return d, e
				}
				q, e := values(sets[1], n)
				if e != nil {
					return d, e
				}
				t := numeric(p, q, n.Epsilon, n.Floor)
				if t.Score != f.Score {
					return d, fmt.Errorf("score mismatch")
				}
				x.Terms = &t
			}
		}
		if x.Terms == nil {
			for _, c := range cs[0].Profile.Categories {
				if c.Name == f.Feature {
					for _, xs := range sets {
						counts := map[string]int{}
						for _, obs := range xs {
							v, ok := provenance.Get(obs.Values, c.Selector)
							s, valid := v.(string)
							if !ok || !valid {
								return d, fmt.Errorf("missing category")
							}
							counts[s]++
						}
						x.Categories = append(x.Categories, counts)
					}
				}
			}
		}
		d.Rows = append(d.Rows, x)
	}
	ties := map[float64][]string{}
	scores := []float64{}
	for _, x := range d.Rows {
		if _, ok := ties[x.Score]; !ok {
			scores = append(scores, x.Score)
		}
		ties[x.Score] = append(ties[x.Score], x.Feature)
	}
	sort.Float64s(scores)
	d.MinimumPositiveScoreGap = 1
	for i, s := range scores {
		if len(ties[s]) > 1 {
			d.ExactTies = append(d.ExactTies, ties[s])
		}
		if i > 0 {
			d.MinimumPositiveScoreGap = math.Min(d.MinimumPositiveScoreGap, s-scores[i-1])
		}
	}
	sessions := map[string]*session{}
	for group, xs := range sets {
		for _, obs := range xs {
			s := sessions[obs.Session]
			if s == nil {
				s = &session{ID: obs.Session, FrameMin: [2]int{999999, 999999}}
				sessions[obs.Session] = s
			}
			v, _ := provenance.Get(obs.Values, "inputs.frame")
			frame := int(v.(float64))
			s.Counts[group]++
			if s.TimeMin[group] == 0 || obs.TS < s.TimeMin[group] {
				s.TimeMin[group] = obs.TS
			}
			s.TimeMax[group] = max(s.TimeMax[group], obs.TS)
			s.FrameMin[group] = min(s.FrameMin[group], frame)
			s.FrameMax[group] = max(s.FrameMax[group], frame)
			d.TimeBins[(frame-1000)/50][group]++
			for i := range 7 {
				v, _ := provenance.Get(obs.Values, fmt.Sprintf("inputs.noise.%d", i))
				s.NoiseMean[group] += v.(float64)
			}
		}
	}
	for _, s := range sessions {
		for g := range 2 {
			if s.Counts[g] > 0 {
				s.NoiseMean[g] /= float64(7 * s.Counts[g])
			}
		}
		d.Sessions = append(d.Sessions, *s)
	}
	sort.Slice(d.Sessions, func(i, j int) bool { return d.Sessions[i].ID < d.Sessions[j].ID })
	return d, nil
}
func markdown(ds []diagnosis) string {
	var b strings.Builder
	b.WriteString("# Frozen lane-01 diagnostic\n\nSynthetic data only. Read-only pinned evidence; every complete report is checked against lane-01.json. No new seeds or proposed methods are evaluated. JSON contains full precision, ranges, per-session counts/frame ranges/noise means, and four chronological bin counts.\n\nNumeric: scale=max(1.4826 MAD,epsilon); z=abs(P-R)/scale; score=(1-exp(-z/3))*(1-(e+1)/102), unless abs(P-R)<floor, then zero. e includes ties in the direction of P. For below-floor rows e/z are diagnostic terms: the scorer exits before computing them. Categorical medians/scales/z/e are inapplicable; show smoothed TV and raw category counts instead. Rank — means unranked, not tied for a discovery.\n")
	for _, d := range ds {
		fmt.Fprintf(&b, "\n## %s / %d\n\n| Field | Rank | P median | R median | MAD | epsilon | scale | z | e | Score | Floor exit |\n|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---|\n", d.Variant, d.Seed)
		for _, x := range d.Rows {
			rank := "—"
			if x.Rank > 0 {
				rank = fmt.Sprint(x.Rank)
			}
			if t := x.Terms; t != nil {
				fmt.Fprintf(&b, "| %s | %s | %.6f | %.6f | %.6f | %.2f | %.6f | %.6f | %d | %.9f | %t |\n", x.Feature, rank, t.ProblemMedian, t.ReferenceMedian, t.MAD, t.Epsilon, t.Scale, t.Z, t.E, t.Score, t.BelowFloor)
			} else {
				fmt.Fprintf(&b, "| %s | %s | N/A | N/A | N/A | N/A | N/A | N/A | N/A | %.9f | N/A |\n", x.Feature, rank, x.Score)
			}
		}
		fmt.Fprintf(&b, "\nExact ties (including unranked zero observations): %v. Smallest distinct score gap: %.12g.\n", d.ExactTies, d.MinimumPositiveScoreGap)
		for _, x := range d.Rows {
			if x.Terms == nil {
				fmt.Fprintf(&b, "\n%s raw P/R categories: %v.\n", x.Feature, x.Categories)
			}
		}
		fmt.Fprintf(&b, "\nChronological 50-frame bins P/R: %v. Session P/R counts and mean noise (averaged over seven fields):\n\n| Session | P/R | P noise | R noise |\n|---|---|---:|---:|\n", d.TimeBins)
		for _, s := range d.Sessions {
			fmt.Fprintf(&b, "| %s | %v | %.6f | %.6f |\n", s.ID, s.Counts, s.NoiseMean[0], s.NoiseMean[1])
		}
	}
	return b.String()
}
func run() error {
	db := flag.String("db", ".cache/lane-01/eval-002/fresh.db", "existing frozen store")
	input := flag.String("input", "benchmark/results/lane-01.json", "frozen reports")
	out := flag.String("out", "benchmark/results/lane-01-diagnostic", "new output prefix")
	flag.Parse()
	raw, err := os.ReadFile(*input)
	if err != nil {
		return err
	}
	var report struct {
		Cases []struct {
			Seed    int           `json:"seed"`
			Variant string        `json:"variant"`
			Result  cohort.Result `json:"result"`
		} `json:"cases"`
	}
	if err = json.Unmarshal(raw, &report); err != nil {
		return err
	}
	ctx := context.Background()
	r, err := store.OpenReader(ctx, *db)
	if err != nil {
		return err
	}
	defer func() { _ = r.Close() }()
	ds := []diagnosis{}
	for _, c := range report.Cases {
		if c.Variant != "decoy-heavy" && c.Variant != "harder" {
			continue
		}
		d, e := analyze(ctx, r, c.Seed, c.Variant, c.Result)
		if e != nil {
			return e
		}
		ds = append(ds, d)
	}
	if len(ds) != 10 {
		return fmt.Errorf("expected ten frozen cases, got %d", len(ds))
	}
	data, err := json.MarshalIndent(ds, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(*out+".json", append(data, '\n'), 0600); err != nil {
		return err
	}
	return os.WriteFile(*out+".md", []byte(markdown(ds)), 0600)
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
