package ai

import (
	"context"
	"fmt"
	"time"
)

// Analyst produces the structured investment thesis and explanation for a candidate stock.
type Analyst interface {
	Analyze(ctx context.Context, in CandidateInput) (*Analysis, error)
}

type analystImpl struct {
	tools Tools
}

// NewAnalyst constructs the core analyst engine.
func NewAnalyst(tools Tools) Analyst {
	return &analystImpl{tools: tools}
}

func (a *analystImpl) Analyze(ctx context.Context, in CandidateInput) (*Analysis, error) {
	s := in.Stock

	// Strengths collection
	strengths := []string{}
	if s.EPSGrowth != nil && *s.EPSGrowth > 0.10 {
		strengths = append(strengths, fmt.Sprintf("Strong EPS growth of %.1f%% YoY", *s.EPSGrowth*100))
	}
	if s.MarginOfSafety > 0 {
		strengths = append(strengths, fmt.Sprintf("Positive Graham Margin of Safety (%.1f%% discount to Graham value)", s.MarginOfSafety*100))
	}
	if s.ROE != nil && *s.ROE > 0.15 {
		strengths = append(strengths, fmt.Sprintf("High Return on Equity of %.1f%%", *s.ROE*100))
	}
	if s.PEG != nil && *s.PEG < 1.0 {
		strengths = append(strengths, fmt.Sprintf("Attractive Lynch PEG ratio of %.2f (< 1.0)", *s.PEG))
	}
	if len(strengths) == 0 {
		strengths = append(strengths, "Sufficient quantitative profile across Graham and Lynch factors")
	}

	// Risks collection
	risks := []string{}
	if s.PE != nil && *s.PE > 20.0 {
		risks = append(risks, fmt.Sprintf("Relatively elevated P/E multiple of %.1fx", *s.PE))
	}
	if s.MarginOfSafety <= 0 {
		risks = append(risks, "Narrow or negative Graham margin of safety relative to market price")
	}
	if s.RevenueGrowth != nil && *s.RevenueGrowth < 0.05 {
		risks = append(risks, fmt.Sprintf("Modest top-line revenue growth of %.1f%%", *s.RevenueGrowth*100))
	}
	if len(risks) == 0 {
		risks = append(risks, "Market liquidity and macro sector cyclicality risks")
	}

	// Why it ranked
	whyRanked := fmt.Sprintf(WhyRankedBalanced, fmt.Sprintf("%.1f", s.GrahamScore), fmt.Sprintf("%.1f", s.LynchScore))
	if s.GrahamScore > s.LynchScore+15 {
		whyRanked = fmt.Sprintf(WhyRankedValueDominant, fmt.Sprintf("%.1f%%", s.MarginOfSafety*100))
	} else if s.LynchScore > s.GrahamScore+15 {
		pegStr := "n/a"
		if s.PEG != nil {
			pegStr = fmt.Sprintf("%.2f", *s.PEG)
		}
		epsStr := "n/a"
		if s.EPSGrowth != nil {
			epsStr = fmt.Sprintf("%.1f%%", *s.EPSGrowth*100)
		}
		whyRanked = fmt.Sprintf(WhyRankedGrowthDominant, epsStr, pegStr)
	}

	// Summary
	summary := fmt.Sprintf("Strong quantitative ranking (#%d) combining Graham value (score %.1f) and Lynch growth (score %.1f).", s.Rank, s.GrahamScore, s.LynchScore)
	if in.Stale {
		summary = "⚠ Data may be stale: " + summary
	}

	// Missing data notes
	missing := []string{}
	if s.Sector == "Financials" {
		missing = append(missing, "DER not applicable (standard financial institution profile)")
	}
	if s.RevenueGrowth == nil {
		missing = append(missing, "Historical revenue growth comparison data unavailable")
	}

	// Statistical anomaly inspection
	var flags []RiskFlag
	if a.tools != nil {
		flags = a.tools.DetectAnomalies(ctx, s, nil)
	}

	// Confidence calibration: 70 - 95 based on completeness
	conf := 85
	if in.Stale {
		conf -= 15
	}
	if len(flags) > 0 {
		conf -= 5 * len(flags)
	}
	if conf < 50 {
		conf = 50
	}

	analysis := &Analysis{
		Ticker:      s.Ticker,
		RunID:       in.RunID,
		Model:       "gemhunter-deterministic-v1",
		Provider:    "quant-heuristic",
		Summary:     summary,
		Strengths:   strengths,
		Risks:       risks,
		WhyRanked:   whyRanked,
		BullCase:    fmt.Sprintf("Continued earnings expansion sustains PEG ratio and supports fundamental Graham value appreciation towards Rp %.0f.", s.GrahamValue),
		BearCase:    "Compression of valuation multiples or decelerating earnings growth could impact relative ranking.",
		Confidence:  conf,
		MissingData: missing,
		RiskFlags:   flags,
		CreatedAt:   time.Now().UTC(),
		IsStale:     in.Stale,
	}

	if err := ValidateAnalysis(analysis, in); err != nil {
		return nil, fmt.Errorf("guardrail check failed: %w", err)
	}

	return analysis, nil
}
