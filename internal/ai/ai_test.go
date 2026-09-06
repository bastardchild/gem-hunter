package ai_test

import (
	"context"
	"errors"
	"testing"

	"gemhunter/internal/ai"
	"gemhunter/internal/service"
)

func TestGuardrails(t *testing.T) {
	// Forbidden word test
	err := ai.ValidateText("Saham ini pasti untung 100%", 90.0, 90.0, 90.0)
	if err == nil || !errors.Is(err, ai.ErrForbiddenWord) {
		t.Errorf("Expected forbidden word error, got: %v", err)
	}

	// Score tampering test
	err = ai.ValidateText("Engine produced gl_score = 99.9 for this stock", 90.0, 90.0, 90.0)
	if err == nil || !errors.Is(err, ai.ErrScoreTampered) {
		t.Errorf("Expected score tampered error, got: %v", err)
	}

	// Clean text test
	err = ai.ValidateText("Stock exhibits solid Graham margin of safety and good Lynch PEG score.", 90.0, 90.0, 90.0)
	if err != nil {
		t.Errorf("Expected clean validation, got: %v", err)
	}
}

func TestAnalystDeterministic(t *testing.T) {
	tools := ai.NewTools(nil)
	analyst := ai.NewAnalyst(tools)

	epsG := 0.20
	peg := 0.8
	roe := 0.18

	input := ai.CandidateInput{
		Stock: service.RankedStock{
			Rank:           1,
			Ticker:         "BBCA",
			CompanyName:    "Bank Central Asia Tbk",
			Sector:         "Financials",
			Price:          9800,
			GLScore:        91.4,
			GrahamScore:    94.1,
			LynchScore:     88.2,
			GrahamValue:    12500,
			MarginOfSafety: 0.27,
			EPSGrowth:      &epsG,
			PEG:            &peg,
			ROE:            &roe,
		},
		RunID:        "run-test-1",
		CalculatedAt: "2026-09-06 12:00",
		DataAsOf:     "2026-09-06 12:00",
		Stale:        false,
	}

	analysis, err := analyst.Analyze(context.Background(), input)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if analysis.Ticker != "BBCA" {
		t.Errorf("Expected ticker BBCA, got %s", analysis.Ticker)
	}
	if analysis.Confidence < 70 {
		t.Errorf("Expected high confidence score, got %d", analysis.Confidence)
	}
	if len(analysis.Strengths) == 0 {
		t.Errorf("Expected strength points")
	}
	if len(analysis.Risks) == 0 {
		t.Errorf("Expected risk points")
	}
}
