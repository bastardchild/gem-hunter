package gemsentinel

import (
	"testing"

	"gemhunter/internal/domain"
)

func ptr(v float64) *float64 {
	return &v
}

func TestCalculateRatios(t *testing.T) {
	// X1: 50 / 200 = 0.25
	if x1 := CalculateX1(50, 200); x1 != 0.25 {
		t.Errorf("expected 0.25, got %f", x1)
	}
	if x1 := CalculateX1(50, 0); x1 != 0 {
		t.Errorf("expected 0 on 0 TA, got %f", x1)
	}

	// X2: 20 / 200 = 0.10
	if x2 := CalculateX2(20, 200); x2 != 0.10 {
		t.Errorf("expected 0.10, got %f", x2)
	}

	// X3: 15 / 50 = 0.30
	if x3 := CalculateX3(15, 50); x3 != 0.30 {
		t.Errorf("expected 0.30, got %f", x3)
	}

	// X4: 100 / 200 = 0.50
	if x4 := CalculateX4(100, 200); x4 != 0.50 {
		t.Errorf("expected 0.50, got %f", x4)
	}
}

func TestComputeSpringateScore(t *testing.T) {
	// Case 1: Healthy stock
	healthy := &domain.Snapshot{
		Ticker:             "BBCA",
		WorkingCapital:     ptr(100),
		TotalAssets:        ptr(500), // x1 = 0.2
		EBIT:               ptr(150), // x2 = 0.3
		ProfitBeforeTax:    ptr(120),
		CurrentLiabilities: ptr(100), // x3 = 1.2
		Revenue:            ptr(400), // x4 = 0.8
	}
	// Score = 1.03*0.2 + 3.07*0.3 + 0.66*1.2 + 0.40*0.8 = 0.206 + 0.921 + 0.792 + 0.320 = 2.239
	resHealthy := ComputeSpringateScore(healthy)
	if resHealthy.Score < 0.862 {
		t.Errorf("expected healthy score > 0.862, got %f", resHealthy.Score)
	}
	if resHealthy.DistressZone != "HEALTHY" {
		t.Errorf("expected HEALTHY zone, got %s", resHealthy.DistressZone)
	}

	// Case 2: Critical distress stock
	distress := &domain.Snapshot{
		Ticker:             "SRIL",
		WorkingCapital:     ptr(-50),
		TotalAssets:        ptr(200), // x1 = -0.25
		EBIT:               ptr(-20), // x2 = -0.10
		ProfitBeforeTax:    ptr(-30),
		CurrentLiabilities: ptr(150), // x3 = -0.20
		Revenue:            ptr(40),  // x4 = 0.20
	}
	// Score = 1.03*(-0.25) + 3.07*(-0.10) + 0.66*(-0.20) + 0.40*(0.20)
	// = -0.2575 - 0.307 - 0.132 + 0.08 = -0.6165
	resDistress := ComputeSpringateScore(distress)
	if resDistress.Score >= 0.50 {
		t.Errorf("expected critical score < 0.50, got %f", resDistress.Score)
	}
	if resDistress.DistressZone != "CRITICAL" {
		t.Errorf("expected CRITICAL zone, got %s", resDistress.DistressZone)
	}
}

func TestIdentifyPrimaryVulnerability(t *testing.T) {
	analysis := domain.SpringateAnalysis{
		X1: -0.2,
		X2: -0.1,
	}
	vuln := IdentifyPrimaryVulnerability(analysis)
	if vuln != "Defisit Modal Kerja & Rugi Operasional" {
		t.Errorf("unexpected vulnerability: %s", vuln)
	}
}
