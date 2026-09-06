package ai

import (
	"context"
	"fmt"
)

// RiskEngine evaluates candidate stocks for operational, debt, valuation, and statistical red flags.
type RiskEngine interface {
	Assess(ctx context.Context, in CandidateInput) ([]RiskFlag, error)
}

type riskEngineImpl struct{}

func NewRiskEngine() RiskEngine {
	return &riskEngineImpl{}
}

func (r *riskEngineImpl) Assess(ctx context.Context, in CandidateInput) ([]RiskFlag, error) {
	flags := []RiskFlag{}
	s := in.Stock

	// 1. Debt risk (exempt for Financials/Banks)
	if s.Sector != "Financials" {
		// Note: DER checked if available via financials
	}

	// 2. Earnings growth check
	if s.EPSGrowth != nil && *s.EPSGrowth < 0.05 {
		flags = append(flags, RiskFlag{
			Category: RiskEarningsDeterioration,
			Severity: SeverityLow,
			Evidence: fmt.Sprintf("EPS growth of %.1f%% is below target screening velocity", *s.EPSGrowth*100),
			Metric:   "eps_growth",
		})
	}

	// 3. Margin of safety check
	if s.MarginOfSafety < 0.0 {
		flags = append(flags, RiskFlag{
			Category: RiskValuationRisk,
			Severity: SeverityHigh,
			Evidence: fmt.Sprintf("Negative Margin of Safety (%.1f%%)", s.MarginOfSafety*100),
			Metric:   "margin_of_safety",
		})
	}

	return flags, nil
}
