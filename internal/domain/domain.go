package domain

import "time"

// CompanyFinancialSnapshot adalah model kanonis (lihat .memory/formula-logic.md §1).
type Snapshot struct {
	Ticker        string
	CompanyName   string
	Sector        string
	Industry      string
	IsFinancial   bool
	Price         float64
	MarketCap     float64
	EPS           *float64
	PrevEPS       *float64
	BVPS          *float64
	PE            *float64
	PB            *float64
	ROE           *float64 // fraksi desimal
	DER           *float64 // nil/N/A untuk bank
	Revenue       *float64
	PrevRevenue   *float64
	DividendYield *float64
	// Springate Financial Distress Metrics
	WorkingCapital     *float64
	TotalAssets        *float64
	EBIT               *float64
	ProfitBeforeTax    *float64
	CurrentLiabilities *float64
	DataDate      time.Time
	PeriodEnd     time.Time
	PublishedAt   time.Time
	FetchedAt     time.Time
}

// MarketDataProvider adalah boundary ke Sectors V2 (lihat .memory/skill.md).
type MarketDataProvider interface {
	GetCompanies(ctx interface{}) error
}

// SpringateAnalysis holds Springate Score ratios and distress classification.
type SpringateAnalysis struct {
	X1           float64 `json:"x1"` // Working Capital / Total Assets
	X2           float64 `json:"x2"` // EBIT / Total Assets
	X3           float64 `json:"x3"` // Profit Before Tax / Current Liabilities
	X4           float64 `json:"x4"` // Revenue / Total Assets
	Score        float64 `json:"score"`
	DistressZone string  `json:"distress_zone"` // "CRITICAL", "MODERATE", "HEALTHY"
}

// SentinelStock holds Gem Sentinel distress surveillance data for a single stock.
type SentinelStock struct {
	Rank               int               `json:"rank"`
	Ticker             string            `json:"ticker"`
	CompanyName        string            `json:"company_name"`
	Sector             string            `json:"sector"`
	Price              float64           `json:"price"`
	MarketCap          float64           `json:"market_cap"`
	Springate          SpringateAnalysis `json:"springate"`
	DistressLevel      string            `json:"distress_level"` // "🔴 Critical", "🟠 Moderate", "🟢 Healthy"
	PrimaryVulnerability string          `json:"primary_vulnerability"`
	AISentinelSummary  string            `json:"ai_sentinel_summary"`
}
