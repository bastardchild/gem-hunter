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
	DataDate      time.Time
	PeriodEnd     time.Time
	PublishedAt   time.Time
	FetchedAt     time.Time
}

// MarketDataProvider adalah boundary ke Sectors V2 (lihat .memory/skill.md).
type MarketDataProvider interface {
	GetCompanies(ctx interface{}) error
}
