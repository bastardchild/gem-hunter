package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

// Analyst produces the structured investment thesis and explanation for a candidate stock.
type Analyst interface {
	Analyze(ctx context.Context, in CandidateInput) (*Analysis, error)
}

type analystImpl struct {
	tools Tools
	llm   *LLMClient
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

	if a.llm != nil && a.llm.Available() {
		enriched := a.enrichWithLLM(ctx, in, analysis)
		if enriched != nil {
			analysis = enriched
		}
	}

	if err := ValidateAnalysis(analysis, in); err != nil {
		return nil, fmt.Errorf("guardrail check failed: %w", err)
	}

	return analysis, nil
}

func (a *analystImpl) enrichWithLLM(ctx context.Context, in CandidateInput, base *Analysis) *Analysis {
	var b strings.Builder
	b.WriteString("Candidate: " + in.Stock.Ticker + " (" + in.Stock.CompanyName + ")\n")
	b.WriteString("Sector: " + in.Stock.Sector + "\n")
	b.WriteString("Rank: #" + fmt.Sprintf("%d", in.Stock.Rank) + "\n")
	b.WriteString("GLScore: " + fmt.Sprintf("%.1f", in.Stock.GLScore) + "\n")
	b.WriteString("GrahamScore: " + fmt.Sprintf("%.1f", in.Stock.GrahamScore) + "\n")
	b.WriteString("LynchScore: " + fmt.Sprintf("%.1f", in.Stock.LynchScore) + "\n")
	b.WriteString("GrahamValue: " + fmt.Sprintf("Rp %.0f", in.Stock.GrahamValue) + "\n")
	b.WriteString("MarginOfSafety: " + fmt.Sprintf("%.1f%%", in.Stock.MarginOfSafety*100) + "\n")
	if in.Stock.PE != nil {
		b.WriteString("PE: " + fmt.Sprintf("%.2f", *in.Stock.PE) + "\n")
	}
	if in.Stock.PB != nil {
		b.WriteString("PB: " + fmt.Sprintf("%.2f", *in.Stock.PB) + "\n")
	}
	if in.Stock.PEG != nil {
		b.WriteString("PEG: " + fmt.Sprintf("%.2f", *in.Stock.PEG) + "\n")
	}
	if in.Stock.EPSGrowth != nil {
		b.WriteString("EPSGrowth: " + fmt.Sprintf("%.1f%%", *in.Stock.EPSGrowth*100) + "\n")
	}
	if in.Stock.RevenueGrowth != nil {
		b.WriteString("RevenueGrowth: " + fmt.Sprintf("%.1f%%", *in.Stock.RevenueGrowth*100) + "\n")
	}
	if in.Stock.ROE != nil {
		b.WriteString("ROE: " + fmt.Sprintf("%.1f%%", *in.Stock.ROE*100) + "\n")
	}
	b.WriteString("Price: " + fmt.Sprintf("Rp %.0f", in.Stock.Price) + "\n")
	b.WriteString("DataDate: " + in.Stock.DataDate + "\n")

	prompt := `Write a concise 2-3 sentence qualitative Summary for this stock, then 2-3 bullet Strengths and 2-3 bullet Risks.
Then provide 1-2 sentence Bull Case and 1-2 sentence Bear Case. Then a single N/A-for-banks note if applicable.
Do not invent or modify any scores. Do not give direct buy/sell advice.
Use objective terms only: "ranking", "signal", "screening". Respond in JSON with keys: summary, strengths (array), risks (array), bull_case, bear_case.`

	content, err := a.llm.Complete(ctx, SystemPromptAnalyst, prompt+"\n\n"+b.String())
	if err != nil {
		log.Printf(`{"level":"warn","event":"llm_enrich_fail","ticker":%q,"error":%q}`, in.Stock.Ticker, err.Error())
		return nil
	}

	var parsed struct {
		Summary   string   `json:"summary"`
		Strengths []string `json:"strengths"`
		Risks     []string `json:"risks"`
		BullCase  string   `json:"bull_case"`
		BearCase  string   `json:"bear_case"`
	}
	cleaned := strings.TrimSpace(content)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	if err := json.Unmarshal([]byte(cleaned), &parsed); err != nil {
		log.Printf(`{"level":"warn","event":"llm_json_decode_fail","ticker":%q,"error":%q}`, in.Stock.Ticker, err.Error())
		return nil
	}

	out := *base
	out.Provider = "llm"
	out.Model = a.llm.model
	if parsed.Summary != "" {
		out.Summary = parsed.Summary
	}
	if len(parsed.Strengths) > 0 {
		out.Strengths = parsed.Strengths
	}
	if len(parsed.Risks) > 0 {
		out.Risks = parsed.Risks
	}
	if parsed.BullCase != "" {
		out.BullCase = parsed.BullCase
	}
	if parsed.BearCase != "" {
		out.BearCase = parsed.BearCase
	}
	return &out
}
