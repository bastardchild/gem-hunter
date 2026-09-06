package ai

import (
	"context"
	"fmt"
	"sync"

	"gemhunter/internal/repository"
	"gemhunter/internal/service"
)

// Agent orchestrates screener, tools, analyst, and risk components.
type Agent struct {
	screener Screener
	analyst  Analyst
	risk     RiskEngine
	tools    Tools
	store    *repository.Store

	mu       sync.RWMutex
	cache    map[string]*RunAnalyses // runID -> analyses
	latestByTicker map[string]*Analysis
}

// NewAgent constructs a coordinated AI Agent layer.
func NewAgent(store *repository.Store) *Agent {
	tools := NewTools(store)
	return &Agent{
		screener:       NewScreener(),
		analyst:        NewAnalyst(tools),
		risk:           NewRiskEngine(),
		tools:          tools,
		store:          store,
		cache:          make(map[string]*RunAnalyses),
		latestByTicker: make(map[string]*Analysis),
	}
}

// ProcessRun runs the complete AI analysis pipeline for a quant run result asynchronously.
func (a *Agent) ProcessRun(ctx context.Context, r service.RunResult) (*RunAnalyses, error) {
	runAnalyses := &RunAnalyses{
		RunID:        r.RunID,
		Status:       "pending",
		CalculatedAt: r.CalculatedAt,
		Analyses:     make(map[string]Analysis),
	}

	inputs := make([]CandidateInput, 0, len(r.Stocks))
	for _, s := range r.Stocks {
		inputs = append(inputs, CandidateInput{
			Stock:        s,
			RunID:        r.RunID,
			CalculatedAt: r.CalculatedAt,
			DataAsOf:     r.DataAsOf,
			Stale:        r.Stale,
			DataQuality:  "ok",
		})
	}

	// 1. Screener triage
	_, err := a.screener.Screen(ctx, inputs)
	if err != nil {
		runAnalyses.Status = "failed"
		return runAnalyses, fmt.Errorf("screener failed: %w", err)
	}

	// 2. Analyst & Risk evaluation per candidate
	for _, in := range inputs {
		analysis, err := a.analyst.Analyze(ctx, in)
		if err != nil {
			continue
		}
		runAnalyses.Analyses[in.Stock.Ticker] = *analysis

		a.mu.Lock()
		a.latestByTicker[in.Stock.Ticker] = analysis
		a.mu.Unlock()
	}

	runAnalyses.Status = "done"

	a.mu.Lock()
	a.cache[r.RunID] = runAnalyses
	a.mu.Unlock()

	return runAnalyses, nil
}

// GetAnalysis retrieves the latest analysis for a specific ticker.
func (a *Agent) GetAnalysis(ticker string) (*Analysis, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	an, ok := a.latestByTicker[ticker]
	return an, ok
}

// GetRunAnalyses retrieves all analyses for a specific quant run.
func (a *Agent) GetRunAnalyses(runID string) (*RunAnalyses, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	r, ok := a.cache[runID]
	return r, ok
}
