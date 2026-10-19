package runs

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

type OutcomeInput struct {
	Domain      string
	RootSeen    bool
	RootError   bool
	HTTPStatus  int
	DurationMS  float64
	ThresholdMS float64
	Failed      bool
	Loss        float64
}

func Outcome(in OutcomeInput) string {
	if in.Domain == "ml" {
		if in.Failed || math.IsNaN(in.Loss) || math.IsInf(in.Loss, 0) {
			return "fail"
		}
		return "pass"
	}
	if in.RootError || in.HTTPStatus >= 500 {
		return "fail"
	}
	if !in.RootSeen || in.ThresholdMS > 0 && in.DurationMS > in.ThresholdMS {
		return "degraded"
	}
	return "pass"
}

func text(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}
func numeric(v any) float64 { value, _ := strconv.ParseFloat(text(v), 64); return value }
func truth(v any) bool {
	s := strings.ToLower(text(v))
	return s == "true" || s == "1" || s == "fail" || s == "failed" || s == "error"
}
