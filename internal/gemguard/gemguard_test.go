package gemguard_test

import (
	"math"
	"testing"

	"gemhunter/internal/gemguard"
)

func TestARABoundary(t *testing.T) {
	tests := []struct {
		price    float64
		expected float64
	}{
		{6000, 20.0},
		{5000, 25.0},
		{1500, 25.0},
		{200, 25.0},
		{150, 35.0},
		{50, 35.0},
		{40, 35.0},
	}

	for _, tt := range tests {
		got := gemguard.ARABoundary(tt.price)
		if got != tt.expected {
			t.Errorf("ARABoundary(%.0f) = %.1f, want %.1f", tt.price, got, tt.expected)
		}
	}
}

func TestCalculateVelocityAndPIR(t *testing.T) {
	// 1,000,000 shares avg daily volume, 50,000,000 shares free float -> velocity = 0.02
	velocity := gemguard.CalculateVelocity(1000000, 50000000)
	if math.Abs(velocity-0.02) > 1e-6 {
		t.Errorf("CalculateVelocity = %f, want 0.02", velocity)
	}

	// 10% price change, velocity 0.02 -> PIR = 10 / 0.02 = 500
	pir := gemguard.CalculatePIR(10.0, velocity)
	if math.Abs(pir-500.0) > 1e-4 {
		t.Errorf("CalculatePIR = %f, want 500.0", pir)
	}

	// Zero free float check
	zeroVel := gemguard.CalculateVelocity(1000, 0)
	if zeroVel != 0.0 {
		t.Errorf("CalculateVelocity with zero float = %f, want 0.0", zeroVel)
	}
	cappedPIR := gemguard.CalculatePIR(5.0, 0)
	if cappedPIR < 999.0 {
		t.Errorf("CalculatePIR with 0 velocity = %f, want capped > 999", cappedPIR)
	}
}

func TestVolumeSpikeAndPriceSpikePct(t *testing.T) {
	vs := gemguard.VolumeSpike(500000, 100000)
	if math.Abs(vs-5.0) > 1e-6 {
		t.Errorf("VolumeSpike = %f, want 5.0", vs)
	}

	ps := gemguard.PriceSpikePct(1250, 1000)
	if math.Abs(ps-25.0) > 1e-6 {
		t.Errorf("PriceSpikePct = %f, want 25.0", ps)
	}
}

func TestVolatilityAndRSI(t *testing.T) {
	prices := []float64{100, 105, 103, 108, 110, 109, 115, 120}
	vol := gemguard.Volatility(prices)
	if vol <= 0 {
		t.Errorf("Volatility should be positive, got %f", vol)
	}

	rsi := gemguard.RSI(prices, 5)
	if rsi < 0 || rsi > 100 {
		t.Errorf("RSI out of bounds [0, 100], got %f", rsi)
	}
}

func TestComputeRiskScore(t *testing.T) {
	// Clean low risk stock
	clean := gemguard.StockSecurityAnalysis{
		Ticker:            "BBCA",
		Price:             9800,
		PriceImpactRatio:  0.4,
		VolumeSpike:       1.1,
		PriceSpikePct:     1.5,
		ARAThreshold:      20.0,
		ARACountConsec:    0,
		DER:               0.8,
		InsiderSellingPct: 0.0,
		RSI:               55.0,
	}
	gemguard.ComputeRiskScore(&clean)
	if clean.RiskLevel != "LOW" {
		t.Errorf("Clean stock expected LOW risk, got %s (score %.1f)", clean.RiskLevel, clean.RiskScore)
	}
	if clean.UMASuspected {
		t.Errorf("Clean stock should not be UMA suspected")
	}

	// Manipulated / High Risk stock
	manipulated := gemguard.StockSecurityAnalysis{
		Ticker:            "PUMP",
		Price:             450,
		PriceImpactRatio:  4.5, // > 2.0 (+30)
		VolumeSpike:       5.5, // >= 5.0 (+25)
		PriceSpikePct:     34.5, // near ARA 35% (+25)
		ARAThreshold:      35.0,
		ARACountConsec:    2, // consec ARA (+20)
		DER:               3.2, // DER > 2 (+10)
		InsiderSellingPct: 7.5, // insider dump (+15)
		RSI:               88.0, // overbought (+5)
	}
	gemguard.ComputeRiskScore(&manipulated)
	if manipulated.RiskLevel != "HIGH" {
		t.Errorf("Manipulated stock expected HIGH risk, got %s (score %.1f)", manipulated.RiskLevel, manipulated.RiskScore)
	}
	if !manipulated.UMASuspected {
		t.Errorf("Manipulated stock expected UMASuspected = true")
	}
	if len(manipulated.PrimaryRiskFactors) < 3 {
		t.Errorf("Expected multiple risk factors detected, got %d", len(manipulated.PrimaryRiskFactors))
	}
}
