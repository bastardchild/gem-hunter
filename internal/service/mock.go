package service

import (
	"time"

	"gemhunter/internal/domain"
)

// MockUniverse: 12 emiten sintetis agar MVP bisa jalan tanpa API key.
func MockUniverse() []domain.Snapshot {
	mk := func(t, name, sector string, price, eps, prev, bvps, pe, pb, roe, rev, prevRev float64) domain.Snapshot {
		return domain.Snapshot{Ticker: t, CompanyName: name, Sector: sector, Price: price,
			EPS: &eps, PrevEPS: &prev, BVPS: &bvps, PE: &pe, PB: &pb, ROE: &roe,
			Revenue: &rev, PrevRevenue: &prevRev,
			DataDate: time.Now().Add(-2 * time.Hour), PublishedAt: time.Now().Add(-2 * time.Hour), FetchedAt: time.Now()}
	}
	return []domain.Snapshot{
		mk("ABC", "ABC Corp", "Energy", 5000, 500, 416, 4000, 10, 1.25, 0.18, 1150, 1000),
		mk("BBCA", "Bank Central Asia", "Financials", 6175, 471, 449, 2100, 13.1, 2.9, 0.17, 112000, 100000),
		mk("TLKM", "Telkom Indonesia", "Technology", 3200, 200, 180, 1500, 16, 2.1, 0.14, 150000, 140000),
		mk("ASII", "Astra International", "Industrials", 5600, 600, 550, 3800, 9.3, 1.4, 0.12, 300000, 280000),
		mk("UNVR", "Unilever Indonesia", "Consumer", 4100, 150, 140, 900, 27, 4.5, 0.25, 42000, 40000),
		mk("INDF", "Indofood", "Consumer", 6800, 700, 600, 5200, 9.7, 1.3, 0.11, 110000, 100000),
		mk("ADRO", "Adaro Energy", "Energy", 2700, 400, 320, 2600, 6.75, 1.0, 0.15, 80000, 70000),
		mk("ICBP", "Indofood CBP", "Consumer", 10500, 800, 720, 6000, 13.1, 1.75, 0.13, 67000, 62000),
		mk("SMGR", "Semen Indonesia", "Materials", 3900, 250, 300, 2800, 15.6, 1.4, 0.06, 38000, 37000),
		mk("KLBF", "Kalbe Farma", "Healthcare", 1600, 60, 55, 500, 26.6, 3.2, 0.12, 30000, 28000),
		mk("PGAS", "PGN", "Energy", 1400, 120, 100, 1100, 11.6, 1.27, 0.09, 45000, 42000),
		mk("BMRI", "Bank Mandiri", "Financials", 5900, 550, 500, 2900, 10.7, 2.0, 0.16, 120000, 108000),
	}
}
