package gemsentinel

import (
	"fmt"
	"math"

	"gemhunter/internal/domain"
)

// Springate Formula Weights:
// Score = 1.03(X1) + 3.07(X2) + 0.66(X3) + 0.40(X4)
// Cutoff:
// Score < 0.862 -> Financial Distress
// Score >= 0.862 -> Healthy
const (
	WeightX1 = 1.03
	WeightX2 = 3.07
	WeightX3 = 0.66
	WeightX4 = 0.40

	CutoffDistress = 0.862
	CutoffCritical = 0.500
)

// CalculateX1 calculates Working Capital / Total Assets.
func CalculateX1(workingCapital, totalAssets float64) float64 {
	if totalAssets <= 0 {
		return 0.0
	}
	return workingCapital / totalAssets
}

// CalculateX2 calculates EBIT / Total Assets.
func CalculateX2(ebit, totalAssets float64) float64 {
	if totalAssets <= 0 {
		return 0.0
	}
	return ebit / totalAssets
}

// CalculateX3 calculates Profit Before Tax (EBT) / Current Liabilities.
func CalculateX3(profitBeforeTax, currentLiabilities float64) float64 {
	if currentLiabilities <= 0 {
		return 0.0
	}
	return profitBeforeTax / currentLiabilities
}

// CalculateX4 calculates Revenue / Total Assets.
func CalculateX4(revenue, totalAssets float64) float64 {
	if totalAssets <= 0 {
		return 0.0
	}
	return revenue / totalAssets
}

// ComputeSpringateScore calculates the Springate Score and analysis for a given financial snapshot.
func ComputeSpringateScore(snapshot *domain.Snapshot) domain.SpringateAnalysis {
	if snapshot == nil {
		return domain.SpringateAnalysis{
			Score:        0,
			DistressZone: "CRITICAL",
		}
	}

	var wc, ta, ebit, ebt, cl, rev float64

	if snapshot.WorkingCapital != nil {
		wc = *snapshot.WorkingCapital
	}
	if snapshot.TotalAssets != nil {
		ta = *snapshot.TotalAssets
	}
	if snapshot.EBIT != nil {
		ebit = *snapshot.EBIT
	}
	if snapshot.ProfitBeforeTax != nil {
		ebt = *snapshot.ProfitBeforeTax
	}
	if snapshot.CurrentLiabilities != nil {
		cl = *snapshot.CurrentLiabilities
	}
	if snapshot.Revenue != nil {
		rev = *snapshot.Revenue
	}

	x1 := CalculateX1(wc, ta)
	x2 := CalculateX2(ebit, ta)
	x3 := CalculateX3(ebt, cl)
	x4 := CalculateX4(rev, ta)

	score := (WeightX1 * x1) + (WeightX2 * x2) + (WeightX3 * x3) + (WeightX4 * x4)
	score = math.Round(score*10000) / 10000

	zone := "HEALTHY"
	if score < CutoffCritical {
		zone = "CRITICAL"
	} else if score < CutoffDistress {
		zone = "MODERATE"
	}

	return domain.SpringateAnalysis{
		X1:           math.Round(x1*10000) / 10000,
		X2:           math.Round(x2*10000) / 10000,
		X3:           math.Round(x3*10000) / 10000,
		X4:           math.Round(x4*10000) / 10000,
		Score:        score,
		DistressZone: zone,
	}
}

// IdentifyPrimaryVulnerability determines the most vulnerable Springate ratio.
func IdentifyPrimaryVulnerability(analysis domain.SpringateAnalysis) string {
	if analysis.X1 < 0 && analysis.X2 < 0 {
		return "Working Capital Deficit & Operating Loss"
	}
	if analysis.X1 < 0 {
		return "Working Capital Deficit"
	}
	if analysis.X2 < 0 {
		return "Operating Loss (EBIT Deficit)"
	}
	if analysis.X3 < 0 {
		return "Low Ability to Cover Current Liabilities"
	}
	if analysis.X4 < 0.20 {
		return "Low Asset Turnover Efficiency"
	}
	return "Declining Margin & Liquidity Ratios"
}

// GenerateSentinelSummary produces deterministic AI analysis for Gem Sentinel distress stocks.
func GenerateSentinelSummary(stock domain.SentinelStock) string {
	sp := stock.Springate
	if sp.Score < 0.50 {
		return fmt.Sprintf("Issuer flagged as Critical Financial Distress (Springate Score: %.2f < 0.50). Pressure is driven mainly by %s, with heavily stressed working capital (X1=%.2f) and operating profit (X2=%.2f).",
			sp.Score, stock.PrimaryVulnerability, sp.X1, sp.X2)
	}
	if sp.Score < 0.862 {
		return fmt.Sprintf("Issuer sits in the Moderate Financial Distress zone (Springate Score: %.2f < 0.862). Short-term liabilities (X3=%.2f) pressure solvency and warrant close monitoring.",
			sp.Score, sp.X3)
	}
	return fmt.Sprintf("Issuer shows relatively sound financial condition (Springate Score: %.2f >= 0.862).", sp.Score)
}
