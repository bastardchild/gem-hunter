package ai

import (
	"time"

	"gemhunter/internal/service"
)

// RiskCategory categorizes detected financial/market red flags.
type RiskCategory string

const (
	RiskValuationRisk                 RiskCategory = "valuation_risk"
	RiskEarningsDeterioration         RiskCategory = "earnings_deterioration"
	RiskDebtRisk                      RiskCategory = "debt_risk"
	RiskPriceMomentumDeterioration    RiskCategory = "price_momentum_deterioration"
	RiskSectorConcentration           RiskCategory = "sector_concentration"
	RiskDataAnomaly                   RiskCategory = "data_anomaly"
)

// RiskSeverity levels.
type RiskSeverity string

const (
	SeverityLow    RiskSeverity = "low"
	SeverityMedium RiskSeverity = "medium"
	SeverityHigh   RiskSeverity = "high"
)

// RiskFlag is a specific identified risk item.
type RiskFlag struct {
	Category RiskCategory `json:"category"`
	Severity RiskSeverity `json:"severity"`
	Evidence string       `json:"evidence"`
	Metric   string       `json:"metric"`
}

// ScoreBreakdown represents the percentile scores of each factor component.
type ScoreBreakdown struct {
	SMOS            float64 `json:"s_mos"`
	SPE             float64 `json:"s_pe"`
	SPB             float64 `json:"s_pb"`
	SEPSGrowth      float64 `json:"s_eps_growth"`
	SRevenueGrowth  float64 `json:"s_revenue_growth"`
	SROE            float64 `json:"s_roe"`
	SPEG            float64 `json:"s_peg"`
	BankRenormalize bool    `json:"bank_renormalize"`
}

// ScreenerResult represents the triaging assessment by Screener Agent.
type ScreenerResult struct {
	Ticker   string `json:"ticker"`
	DeepDive bool   `json:"deep_dive"`
	Reason   string `json:"reason"`
}

// CandidateInput is the rich contextual input supplied to the AI Analyst Layer.
type CandidateInput struct {
	Stock         service.RankedStock `json:"stock"`
	RunID         string              `json:"run_id"`
	CalculatedAt  string              `json:"calculated_at"`
	DataAsOf      string              `json:"data_as_of"`
	Stale         bool                `json:"stale"`
	Breakdown     ScoreBreakdown      `json:"breakdown"`
	PreviousRank  *int                `json:"previous_rank,omitempty"`
	ScoreDelta6h  *float64            `json:"score_delta_6h,omitempty"`
	SectorPeers   []string            `json:"sector_peers,omitempty"`
	SectorAvgPE   *float64            `json:"sector_avg_pe,omitempty"`
	SectorAvgROE  *float64            `json:"sector_avg_roe,omitempty"`
	DataQuality   string              `json:"data_quality"`
}

// Analysis is the structured output produced for a single ranked stock.
type Analysis struct {
	Ticker       string     `json:"ticker"`
	RunID        string     `json:"run_id"`
	Model        string     `json:"model"`
	Provider     string     `json:"provider"`
	Summary      string     `json:"summary"`
	Strengths    []string   `json:"strengths"`
	Risks        []string   `json:"risks"`
	WhyRanked    string     `json:"why_ranked"`
	BullCase     string     `json:"bull_case"`
	BearCase     string     `json:"bear_case"`
	Confidence   int        `json:"confidence"` // 0 - 100
	MissingData  []string   `json:"missing_data"`
	RiskFlags    []RiskFlag `json:"risk_flags"`
	CreatedAt    time.Time  `json:"created_at"`
	PromptHash   string     `json:"prompt_hash,omitempty"`
	IsStale      bool       `json:"is_stale"`
}

// RunAnalyses stores the complete batch of AI analyses for a specific quant run.
type RunAnalyses struct {
	RunID        string              `json:"run_id"`
	Status       string              `json:"status"` // "pending", "partial", "done", "failed"
	CalculatedAt string              `json:"calculated_at"`
	Analyses     map[string]Analysis `json:"analyses"`
	UpdatedAt    time.Time           `json:"updated_at"`
}
