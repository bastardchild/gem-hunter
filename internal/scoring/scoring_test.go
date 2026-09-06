package scoring

import (
	"math"
	"testing"
)

func TestGrahamValue(t *testing.T) {
	g := GrahamValue(500, 4000)
	if g == nil || math.Abs(*g-6708.2039) > 0.05 {
		t.Fatalf("GrahamValue ABC = %v, want ~6708.20", g)
	}
	if GrahamValue(-1, 4000) != nil || GrahamValue(500, 0) != nil {
		t.Fatal("EPS/BVPS <= 0 harus nil")
	}
}

func TestMOS(t *testing.T) {
	g := GrahamValue(500, 4000)
	m := MarginOfSafety(5000, *g)
	if m == nil || math.Abs(*m-0.2547) > 0.001 {
		t.Fatalf("MOS ABC = %v, want ~0.2547", m)
	}
}

func TestEPSGrowthPEG(t *testing.T) {
	e := EPSGrowth(120, 100)
	if e == nil || math.Abs(*e-0.20) > 1e-9 {
		t.Fatalf("EPSGrowth = %v", e)
	}
	if EPSGrowth(100, 0) != nil || EPSGrowth(100, -5) != nil {
		t.Fatal("denom <= 0 harus nil")
	}
	p := PEG(10, 0.20)
	if p == nil || math.Abs(*p-0.50) > 1e-9 {
		t.Fatalf("PEG ABC = %v, want 0.50", p)
	}
	if PEG(10, 0) != nil || PEG(-5, 0.2) != nil || PEG(10, -0.1) != nil {
		t.Fatal("PEG invalid harus nil")
	}
}

func TestPercentile(t *testing.T) {
	vals := []float64{10, 20, 30, 40, 50}
	if s := PercentileScore(vals, 50, false); math.Abs(s-100) > 1e-9 {
		t.Fatalf("max higher = %v, want 100", s)
	}
	if s := PercentileScore(vals, 10, true); math.Abs(s-100) > 1e-9 {
		t.Fatalf("min lower = %v, want 100", s)
	}
	if s := PercentileScore([]float64{5}, 5, false); s != 50 {
		t.Fatalf("n==1 harus 50, dapat %v", s)
	}
}

func TestScoresABC(t *testing.T) {
	g := GrahamScore(82, 90, 85, 80, 78)
	if math.Abs(g-83.35) > 1e-9 {
		t.Fatalf("GrahamScore = %v, want 83.35", g)
	}
	l := LynchScore(95, 80, 75, 78)
	if math.Abs(l-86.55) > 1e-9 {
		t.Fatalf("LynchScore = %v, want 86.55", l)
	}
	gl := GLScore(g, l)
	if math.Abs(gl-84.79) > 0.01 {
		t.Fatalf("GLScore = %v, want 84.79", gl)
	}
}
