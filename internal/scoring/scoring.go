// Package scoring mengimplementasikan .memory/formula-logic.md secara normatif.
package scoring

import (
	"math"
	"sort"
)

const eps = 1e-9

func f(v float64) *float64 { out := v; return &out }

// GrahamValue: sqrt(22.5*eps*bvps), nil bila eps/bvps <= 0.
func GrahamValue(epsV, bvps float64) *float64 {
	if epsV <= eps || bvps <= eps {
		return nil
	}
	return f(math.Sqrt(22.5 * epsV * bvps))
}

// MarginOfSafety: 1 - price/grahamValue.
func MarginOfSafety(price, grahamValue float64) *float64 {
	if price <= eps || grahamValue <= eps {
		return nil
	}
	return f(1 - price/grahamValue)
}

// EPSGrowth: eps/prev - 1, nil bila prev <= 0.
func EPSGrowth(epsV, prev float64) *float64 {
	if prev <= eps {
		return nil
	}
	return f(epsV/prev - 1)
}

// RevenueGrowth: analog EPSGrowth.
func RevenueGrowth(rev, prev float64) *float64 {
	if prev <= eps {
		return nil
	}
	return f(rev/prev - 1)
}

// PEG: pe / (epsGrowthFrac*100). epsGrowthFrac dalam fraksi (0.20 = 20%).
func PEG(pe, epsGrowthFrac float64) *float64 {
	if pe <= eps || epsGrowthFrac <= eps {
		return nil
	}
	gpct := epsGrowthFrac * 100
	if gpct <= eps {
		return nil
	}
	return f(pe / gpct)
}

// PercentileScore: percent-rank + average-ties. values = universe non-null.
func PercentileScore(values []float64, v float64, lowerIsBetter bool) float64 {
	n := len(values)
	if n == 0 {
		return 50
	}
	if n == 1 {
		return 50
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	// average rank (1-indexed) untuk ties dalam toleransi eps
	less, equal := 0, 0
	for _, x := range sorted {
		if x < v-eps {
			less++
		} else if math.Abs(x-v) <= eps {
			equal++
		}
	}
	r := float64(less) + (float64(equal)+1)/2.0
	p := (r - 1) / float64(n-1)
	if lowerIsBetter {
		return (1 - p) * 100
	}
	return p * 100
}

func GrahamScore(mos, pe, pb, epsG, roe float64) float64 {
	return 0.40*mos + 0.20*pe + 0.15*pb + 0.15*epsG + 0.10*roe
}

func LynchScore(peg, epsG, revG, roe float64) float64 {
	return 0.50*peg + 0.25*epsG + 0.15*revG + 0.10*roe
}

// GLScore: 0.55 Graham + 0.45 Lynch (varian 50/50 DITOLAK).
func GLScore(graham, lynch float64) float64 {
	return 0.55*graham + 0.45*lynch
}

func Label(gl float64) string {
	switch {
	case gl >= 90:
		return "Exceptional"
	case gl >= 80:
		return "Strong"
	case gl >= 70:
		return "Attractive"
	case gl >= 60:
		return "Neutral"
	default:
		return "Weak"
	}
}
