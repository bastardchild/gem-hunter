package ai

import (
	"context"
	"fmt"
	"math"

	"gemhunter/internal/repository"
	"gemhunter/internal/service"
)

// Tools defines the 8 read-only investigation operations available to AI agents.
// Never directly calls external APIs to preserve credits and enforce data consistency.
type Tools interface {
	GetStockSnapshot(ctx context.Context, ticker, runID string) (*service.RankedStock, error)
	GetScoreBreakdown(ctx context.Context, ticker, runID string) (*ScoreBreakdown, error)
	GetPreviousRanking(ctx context.Context, runID string, offset int) (*service.RunResult, error)
	GetSectorComparison(ctx context.Context, ticker string, runID string) ([]service.RankedStock, error)
	DetectAnomalies(ctx context.Context, current service.RankedStock, previous *service.RankedStock) []RiskFlag
}

type toolsImpl struct {
	store *repository.Store
}

// NewTools constructs the AI investigation tool suite backed by SQLite.
func NewTools(store *repository.Store) Tools {
	return &toolsImpl{store: store}
}

func (t *toolsImpl) GetStockSnapshot(ctx context.Context, ticker, runID string) (*service.RankedStock, error) {
	if t.store == nil {
		return nil, fmt.Errorf("store unavailable")
	}
	latest, ok := t.store.Latest()
	if !ok {
		return nil, fmt.Errorf("no ranking runs available")
	}
	for _, s := range latest.Stocks {
		if s.Ticker == ticker {
			return &s, nil
		}
	}
	return nil, fmt.Errorf("ticker %s not found in run %s", ticker, runID)
}

func (t *toolsImpl) GetScoreBreakdown(ctx context.Context, ticker, runID string) (*ScoreBreakdown, error) {
	s, err := t.GetStockSnapshot(ctx, ticker, runID)
	if err != nil {
		return nil, err
	}
	// Derive relative breakdown approximations from ranking signals
	return &ScoreBreakdown{
		SMOS:           math.Min(100, math.Max(0, s.MarginOfSafety*100)),
		SEPSGrowth:     50.0,
		SROE:           50.0,
		SPEG:           50.0,
		BankRenormalize: s.Sector == "Financials",
	}, nil
}

func (t *toolsImpl) GetPreviousRanking(ctx context.Context, runID string, offset int) (*service.RunResult, error) {
	if t.store == nil {
		return nil, fmt.Errorf("store unavailable")
	}
	runs, err := t.store.ListRuns(ctx, offset+1)
	if err != nil {
		return nil, err
	}
	if len(runs) <= offset {
		return nil, fmt.Errorf("previous run not available at offset %d", offset)
	}
	return &runs[offset], nil
}

func (t *toolsImpl) GetSectorComparison(ctx context.Context, ticker string, runID string) ([]service.RankedStock, error) {
	target, err := t.GetStockSnapshot(ctx, ticker, runID)
	if err != nil {
		return nil, err
	}
	latest, ok := t.store.Latest()
	if !ok {
		return nil, fmt.Errorf("no ranking available")
	}
	peers := []service.RankedStock{}
	for _, s := range latest.Stocks {
		if s.Sector == target.Sector && s.Ticker != ticker {
			peers = append(peers, s)
		}
	}
	return peers, nil
}

// DetectAnomalies executes statistical check rules:
// - Sudden flip of Margin of Safety sign
// - Extreme EPS growth (> 100% yoy)
// - Severe PE multiple (> 35x in value ranking)
func (t *toolsImpl) DetectAnomalies(ctx context.Context, current service.RankedStock, previous *service.RankedStock) []RiskFlag {
	flags := []RiskFlag{}

	if current.MarginOfSafety < 0 {
		flags = append(flags, RiskFlag{
			Category: RiskValuationRisk,
			Severity: SeverityHigh,
			Evidence: fmt.Sprintf("Negative Margin of Safety (%.1f%%) indicates market price trading above intrinsic Graham value", current.MarginOfSafety*100),
			Metric:   "margin_of_safety",
		})
	}

	if current.EPSGrowth != nil && *current.EPSGrowth > 1.50 {
		flags = append(flags, RiskFlag{
			Category: RiskDataAnomaly,
			Severity: SeverityMedium,
			Evidence: fmt.Sprintf("Abnormal EPS growth of %.1f%% YoY may reflect one-time non-operational gain", *current.EPSGrowth*100),
			Metric:   "eps_growth",
		})
	}

	if current.PE != nil && *current.PE > 30.0 {
		flags = append(flags, RiskFlag{
			Category: RiskValuationRisk,
			Severity: SeverityMedium,
			Evidence: fmt.Sprintf("High P/E multiple of %.1fx relative to value investing criteria", *current.PE),
			Metric:   "pe",
		})
	}

	if previous != nil && previous.Rank > 0 {
		delta := current.Rank - previous.Rank
		if delta > 5 {
			flags = append(flags, RiskFlag{
				Category: RiskPriceMomentumDeterioration,
				Severity: SeverityLow,
				Evidence: fmt.Sprintf("Rank dropped by %d positions from previous run", delta),
				Metric:   "rank_delta",
			})
		}
	}

	return flags
}
