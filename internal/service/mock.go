package service

import (
	"time"

	"gemhunter/internal/domain"
)

// MockUniverse: 12 emiten sintetis agar MVP bisa jalan tanpa API key.
func MockUniverse() []domain.Snapshot {
	mk := func(t, name, sector string, price, eps, prev, bvps, pe, pb, roe, rev, prevRev float64, wc, ta, ebit, ebt, cl float64) domain.Snapshot {
		return domain.Snapshot{Ticker: t, CompanyName: name, Sector: sector, Price: price,
			EPS: &eps, PrevEPS: &prev, BVPS: &bvps, PE: &pe, PB: &pb, ROE: &roe,
			Revenue: &rev, PrevRevenue: &prevRev,
			WorkingCapital: &wc, TotalAssets: &ta, EBIT: &ebit, ProfitBeforeTax: &ebt, CurrentLiabilities: &cl,
			DataDate: time.Now().Add(-2 * time.Hour), PublishedAt: time.Now().Add(-2 * time.Hour), FetchedAt: time.Now()}
	}
	return []domain.Snapshot{
		// Beberapa emiten dalam zona financial distress untuk evaluasi Springate:
		// 1. SRIL: Modal kerja minus, EBIT rugi, utang lancar bengkak -> Springate < 0 (Critical)
		mk("SRIL", "Sri Rejeki Isman", "Consumer", 50, -40, -30, -100, 1.0, 0.5, -0.2, 500, 600, -800, 1500, -250, -300, 1200),
		// 2. WSKT: Beban utang tinggi, modal kerja tertekan -> Springate ~ 0.22 (Critical)
		mk("WSKT", "Waskita Karya", "Industrials", 180, -25, -20, 200, 2.0, 0.9, -0.1, 1500, 1800, -400, 3200, 50, -100, 2500),
		// 3. KRAS: Asset turnover lambat & likuiditas ketat -> Springate ~ 0.45 (Critical)
		mk("KRAS", "Krakatau Steel", "Materials", 140, 5, 2, 180, 28, 0.8, 0.02, 2200, 2400, -150, 4000, 80, 20, 2800),
		// 4. GIAA: Borderline distress -> Springate ~ 0.62 (Moderate)
		mk("GIAA", "Garuda Indonesia", "Industrials", 60, -10, -50, -50, 1.5, 0.8, -0.05, 3500, 2800, -300, 5000, 300, 80, 3800),
		// 5. DOID: Moderate leverage pressure -> Springate ~ 0.78 (Moderate)
		mk("DOID", "Delta Dunia Makmur", "Energy", 480, 45, 40, 520, 8.5, 0.9, 0.08, 6200, 5800, 100, 7500, 450, 250, 3100),

		// Emiten Sehat:
		mk("ABC", "ABC Corp", "Energy", 5000, 500, 416, 4000, 10, 1.25, 0.18, 1150, 1000, 500, 2500, 400, 350, 600),
		mk("BBCA", "Bank Central Asia", "Financials", 6175, 471, 449, 2100, 13.1, 2.9, 0.17, 112000, 100000, 45000, 180000, 35000, 32000, 65000),
		mk("TLKM", "Telkom Indonesia", "Technology", 3200, 200, 180, 1500, 16, 2.1, 0.14, 150000, 140000, 25000, 200000, 32000, 28000, 50000),
		mk("ASII", "Astra International", "Industrials", 5600, 600, 550, 3800, 9.3, 1.4, 0.12, 300000, 280000, 60000, 350000, 38000, 35000, 85000),
		mk("UNVR", "Unilever Indonesia", "Consumer", 4100, 150, 140, 900, 27, 4.5, 0.25, 42000, 40000, 2000, 18000, 6500, 6000, 7000),
		mk("INDF", "Indofood", "Consumer", 6800, 700, 600, 5200, 9.7, 1.3, 0.11, 110000, 100000, 18000, 120000, 15000, 13000, 35000),
		mk("ADRO", "Adaro Energy", "Energy", 2700, 400, 320, 2600, 6.75, 1.0, 0.15, 80000, 70000, 22000, 90000, 20000, 18000, 25000),
		mk("ICBP", "Indofood CBP", "Consumer", 10500, 800, 720, 6000, 13.1, 1.75, 0.13, 67000, 62000, 14000, 85000, 11000, 10000, 22000),
		mk("SMGR", "Semen Indonesia", "Materials", 3900, 250, 300, 2800, 15.6, 1.4, 0.06, 38000, 37000, 3000, 70000, 4000, 3000, 20000),
		mk("KLBF", "Kalbe Farma", "Healthcare", 1600, 60, 55, 500, 26.6, 3.2, 0.12, 30000, 28000, 8000, 25000, 4200, 3900, 5000),
		mk("PGAS", "PGN", "Energy", 1400, 120, 100, 1100, 11.6, 1.27, 0.09, 45000, 42000, 5000, 40000, 5000, 4200, 12000),
		mk("BMRI", "Bank Mandiri", "Financials", 5900, 550, 500, 2900, 10.7, 2.0, 0.16, 120000, 108000, 35000, 160000, 28000, 25000, 55000),
	}
}
