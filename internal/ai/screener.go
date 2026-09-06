package ai

import (
	"context"
)

// Screener performs triage on top candidates to determine if a deep dive is required.
type Screener interface {
	Screen(ctx context.Context, inputs []CandidateInput) ([]ScreenerResult, error)
}

type screenerImpl struct{}

func NewScreener() Screener {
	return &screenerImpl{}
}

func (s *screenerImpl) Screen(ctx context.Context, inputs []CandidateInput) ([]ScreenerResult, error) {
	results := make([]ScreenerResult, 0, len(inputs))
	for _, in := range inputs {
		res := ScreenerResult{
			Ticker:   in.Stock.Ticker,
			DeepDive: true,
			Reason:   "Standard candidate deep dive",
		}
		if in.Stock.GrahamScore > 85.0 && in.Stock.LynchScore < 60.0 {
			res.Reason = "Value dominant profile; verify growth sustainability"
		} else if in.Stock.LynchScore > 85.0 && in.Stock.GrahamScore < 60.0 {
			res.Reason = "Growth dominant profile; check valuation margin of safety"
		}
		results = append(results, res)
	}
	return results, nil
}
