package gemguard

import (
	"math"
)

// ARABoundary returns the upper auto-rejection limit percentage based on price level in IDX.
// > Rp 5.000: 20%
// Rp 200 - Rp 5.000: 25%
// Rp 50 - Rp 200: 35%
func ARABoundary(price float64) float64 {
	switch {
	case price > 5000:
		return 20.0
	case price >= 200:
		return 25.0
	case price >= 50:
		return 35.0
	default:
		return 35.0
	}
}

// CalculateVelocity calculates trading velocity:
// Velocity = Average Daily Volume / Free Float
func CalculateVelocity(avgVolume, freeFloat float64) float64 {
	if freeFloat <= 0 {
		return 0
	}
	return avgVolume / freeFloat
}

// CalculatePIR calculates Price Impact Ratio (PIR):
// PIR = |Price Change %| / Velocity
// If velocity <= 0, PIR is considered high (infinity / cap).
func CalculatePIR(priceChangePct, velocity float64) float64 {
	if velocity <= 0 {
		if math.Abs(priceChangePct) > 0 {
			return 999.99
		}
		return 0.0
	}
	return math.Abs(priceChangePct) / velocity
}

// VolumeSpike calculates ratio of latest volume to 20-day average:
// VolumeSpike = Vt / AvgVolume20
func VolumeSpike(latestVolume, avgVolume20 float64) float64 {
	if avgVolume20 <= 0 {
		return 1.0
	}
	return latestVolume / avgVolume20
}

// PriceSpikePct calculates percentage price spike between two periods:
// PriceSpike = ((Pt - Pt-1) / Pt-1) * 100
func PriceSpikePct(currentPrice, prevPrice float64) float64 {
	if prevPrice <= 0 {
		return 0.0
	}
	return ((currentPrice - prevPrice) / prevPrice) * 100.0
}

// Volatility calculates sample standard deviation of log returns:
// sigma = sqrt( 1/(n-1) * sum((Ri - mean)^2) )
func Volatility(prices []float64) float64 {
	if len(prices) < 2 {
		return 0.0
	}
	returns := make([]float64, len(prices)-1)
	for i := 0; i < len(prices)-1; i++ {
		if prices[i] <= 0 || prices[i+1] <= 0 {
			continue
		}
		returns[i] = math.Log(prices[i+1] / prices[i])
	}
	if len(returns) == 0 {
		return 0.0
	}
	mean := 0.0
	for _, r := range returns {
		mean += r
	}
	mean /= float64(len(returns))

	sumSq := 0.0
	for _, r := range returns {
		diff := r - mean
		sumSq += diff * diff
	}
	return math.Sqrt(sumSq / float64(len(returns)-1))
}

// RSI calculates the Relative Strength Index (period 14 typical).
func RSI(prices []float64, period int) float64 {
	if len(prices) <= period {
		return 50.0
	}
	var gains, losses float64
	for i := 1; i <= period; i++ {
		change := prices[i] - prices[i-1]
		if change > 0 {
			gains += change
		} else {
			losses += -change
		}
	}
	avgGain := gains / float64(period)
	avgLoss := losses / float64(period)

	for i := period + 1; i < len(prices); i++ {
		change := prices[i] - prices[i-1]
		if change > 0 {
			avgGain = (avgGain*float64(period-1) + change) / float64(period)
			avgLoss = (avgLoss * float64(period-1)) / float64(period)
		} else {
			avgGain = (avgGain * float64(period-1)) / float64(period)
			avgLoss = (avgLoss*float64(period-1) + (-change)) / float64(period)
		}
	}

	if avgLoss == 0 {
		return 100.0
	}
	rs := avgGain / avgLoss
	return 100.0 - (100.0 / (1.0 + rs))
}

// MovingAverageRatio calculates Pt / MA_n
func MovingAverageRatio(currentPrice float64, prices []float64) float64 {
	if len(prices) == 0 {
		return 1.0
	}
	sum := 0.0
	for _, p := range prices {
		sum += p
	}
	ma := sum / float64(len(prices))
	if ma <= 0 {
		return 1.0
	}
	return currentPrice / ma
}

// StockSecurityAnalysis holds complete Gem Guard security metrics for a single stock.
type StockSecurityAnalysis struct {
	Rank               int     `json:"rank"`
	Ticker             string  `json:"ticker"`
	CompanyName        string  `json:"company_name"`
	Sector             string  `json:"sector"`
	Price              float64 `json:"price"`
	MarketCap          float64 `json:"market_cap"`
	FreeFloat          float64 `json:"free_float"`
	Velocity           float64 `json:"velocity"`
	PriceImpactRatio   float64 `json:"price_impact_ratio"`
	PIRThreshold       float64 `json:"pir_threshold"`
	PriceSpikePct      float64 `json:"price_spike_pct"`
	VolumeSpike        float64 `json:"volume_spike"`
	ARAThreshold       float64 `json:"ara_threshold"`
	ARACountConsec     int     `json:"ara_count_consec"`
	UMASuspected       bool    `json:"uma_suspected"`
	DER                float64 `json:"der"`
	CurrentRatio       float64 `json:"current_ratio"`
	ROE                float64 `json:"roe"`
	EPS                float64 `json:"eps"`
	InsiderHoldingPct  float64 `json:"insider_holding_pct"`
	InsiderSellingPct  float64 `json:"insider_selling_pct"`
	Volatility         float64 `json:"volatility"`
	RSI                float64 `json:"rsi"`
	MovingAvgRatio     float64 `json:"moving_avg_ratio"`
	RiskScore          float64 `json:"risk_score"` // 0 - 100
	RiskLevel          string  `json:"risk_level"` // "LOW", "MEDIUM", "HIGH"
	PrimaryRiskFactors []string `json:"primary_risk_factors"`
}

// ComputeRiskScore derives composite risk score and level from market & fundamental features.
func ComputeRiskScore(analysis *StockSecurityAnalysis) {
	factors := []string{}
	score := 0.0

	// 1. Price Impact Ratio (PIR): Cap/threshold > 2.0
	if analysis.PriceImpactRatio > 2.0 {
		score += 30.0
		factors = append(factors, "High Price Impact Ratio (> 2.0) indicating concentrated float sensitivity")
	} else if analysis.PriceImpactRatio > 1.0 {
		score += 15.0
	}

	// 2. Volume Spike: > 3x - 5x
	if analysis.VolumeSpike >= 5.0 {
		score += 25.0
		factors = append(factors, "Extreme Volume Spike (>= 5x of 20-day average)")
	} else if analysis.VolumeSpike >= 3.0 {
		score += 15.0
		factors = append(factors, "Elevated Volume Spike (>= 3x of 20-day average)")
	}

	// 3. Price Spike near ARA
	if analysis.PriceSpikePct >= analysis.ARAThreshold {
		score += 25.0
		factors = append(factors, "Price change reached Auto Rejection Atas (ARA)")
	} else if analysis.PriceSpikePct >= analysis.ARAThreshold*0.8 {
		score += 10.0
	}

	// 4. Consecutive ARA
	if analysis.ARACountConsec >= 2 {
		score += 20.0
		factors = append(factors, "Consecutive ARA days triggers UMA criteria")
	}

	// 5. High Leverage (DER > 2.0)
	if analysis.DER > 2.0 {
		score += 10.0
		factors = append(factors, "High Financial Leverage (DER > 2.0)")
	}

	// 6. Extreme insider selling (> 5% in a month)
	if analysis.InsiderSellingPct > 5.0 {
		score += 15.0
		factors = append(factors, "Significant insider liquidation (> 5% within 1 month)")
	}

	// 7. Volatility & RSI extreme
	if analysis.RSI > 80.0 {
		score += 5.0
		factors = append(factors, "Severely overbought momentum (RSI > 80)")
	}

	if score > 100.0 {
		score = 100.0
	}

	analysis.RiskScore = score
	analysis.PrimaryRiskFactors = factors

	if score >= 70.0 {
		analysis.RiskLevel = "HIGH"
	} else if score >= 40.0 {
		analysis.RiskLevel = "MEDIUM"
	} else {
		analysis.RiskLevel = "LOW"
	}

	if analysis.ARACountConsec >= 2 || (analysis.PriceSpikePct >= analysis.ARAThreshold*0.95 && analysis.VolumeSpike >= 3.0) {
		analysis.UMASuspected = true
	}
}
