package service

import (
	"sort"
	"time"

	"gemhunter/internal/domain"
	"gemhunter/internal/gemsentinel"
	"gemhunter/internal/scoring"
)

// RankedStock sesuai task.md §14.
type RankedStock struct {
	Rank           int      `json:"rank"`
	Ticker         string   `json:"ticker"`
	CompanyName    string   `json:"company_name"`
	Sector         string   `json:"sector"`
	Price          float64  `json:"price"`
	GLScore        float64  `json:"gl_score"`
	GrahamScore    float64  `json:"graham_score"`
	LynchScore     float64  `json:"lynch_score"`
	GrahamValue    float64  `json:"graham_value"`
	MarginOfSafety float64  `json:"margin_of_safety"`
	PE             *float64 `json:"pe"`
	PB             *float64 `json:"pb"`
	PEG            *float64 `json:"peg"`
	EPSGrowth      *float64 `json:"eps_growth"`
	RevenueGrowth  *float64 `json:"revenue_growth"`
	ROE            *float64 `json:"roe"`
	DataDate       string   `json:"data_date"`
	CalculatedAt   string   `json:"calculated_at"`
}

type RunResult struct {
	RunID        string        `json:"run_id"`
	CalculatedAt string        `json:"calculated_at"`
	DataAsOf     string        `json:"data_as_of"`
	Count        int           `json:"count"`
	Stocks       []RankedStock `json:"stocks"`
	Stale        bool          `json:"stale"`
}

// Rank: pure function atas snapshot (tanpa time.Now/HTTP di dalam formula).
func Rank(snaps []domain.Snapshot, now time.Time, maxAgeH int, minEPSg float64) RunResult {
	// dedup ticker: publishedAt terbaru menang
	best := map[string]domain.Snapshot{}
	for _, s := range snaps {
		if p, ok := best[s.Ticker]; !ok || s.PublishedAt.After(p.PublishedAt) {
			best[s.Ticker] = s
		}
	}
	type mid struct {
		snap domain.Snapshot
		gv   float64
		mos  float64
		epsG *float64
		revG *float64
		peg  *float64
		pe   *float64
		pb   *float64
		roe  *float64
	}
	mids := []mid{}
	for _, s := range best {
		if s.Ticker == "" || s.Price <= 0 || s.EPS == nil {
			continue
		}
		if maxAgeH > 0 && !s.DataDate.IsZero() && now.Sub(s.DataDate).Hours() > float64(maxAgeH) {
			continue
		}
		if s.BVPS == nil {
			continue
		}
		gv := scoring.GrahamValue(*s.EPS, *s.BVPS)
		if gv == nil {
			continue
		}
		mos := scoring.MarginOfSafety(s.Price, *gv)
		if mos == nil || s.PE == nil || *s.PE <= 0 {
			continue
		}
		var epsG, revG *float64
		if s.PrevEPS != nil {
			epsG = scoring.EPSGrowth(*s.EPS, *s.PrevEPS)
		}
		if s.Revenue != nil && s.PrevRevenue != nil {
			revG = scoring.RevenueGrowth(*s.Revenue, *s.PrevRevenue)
		}
		if epsG == nil || *epsG <= minEPSg {
			continue
		}
		peg := scoring.PEG(*s.PE, *epsG)
		if peg == nil {
			continue
		}
		mids = append(mids, mid{s, *gv, *mos, epsG, revG, peg, s.PE, s.PB, s.ROE})
	}
	collect := func(f func(mid) (float64, bool)) []float64 {
		out := []float64{}
		for _, m := range mids {
			if v, ok := f(m); ok {
				out = append(out, v)
			}
		}
		return out
	}
	mosV := collect(func(m mid) (float64, bool) { return m.mos, true })
	peV := collect(func(m mid) (float64, bool) { return *m.pe, true })
	pbV := collect(func(m mid) (float64, bool) { return *m.pb, m.pb != nil })
	epsV := collect(func(m mid) (float64, bool) { return *m.epsG, true })
	revV := collect(func(m mid) (float64, bool) { return *m.revG, m.revG != nil })
	pegV := collect(func(m mid) (float64, bool) { return *m.peg, true })
	roeV := collect(func(m mid) (float64, bool) { return *m.roe, m.roe != nil })

	out := []RankedStock{}
	for _, m := range mids {
		sMOS := scoring.PercentileScore(mosV, m.mos, false)
		sPE := scoring.PercentileScore(peV, *m.pe, true)
		sPB := 50.0
		if m.pb != nil {
			sPB = scoring.PercentileScore(pbV, *m.pb, true)
		}
		sEPS := scoring.PercentileScore(epsV, *m.epsG, false)
		sROE := 50.0
		if m.roe != nil {
			sROE = scoring.PercentileScore(roeV, *m.roe, false)
		}
		// renormalisasi proporsional bila PB/ROE hilang (profil bank)
		g := 0.40*sMOS + 0.20*sPE + 0.15*sPB + 0.15*sEPS + 0.10*sROE
		sPEG := scoring.PercentileScore(pegV, *m.peg, true)
		sREV := 50.0
		if m.revG != nil {
			sREV = scoring.PercentileScore(revV, *m.revG, false)
		}
		l := 0.50*sPEG + 0.25*sEPS + 0.15*sREV + 0.10*sROE
		gl := scoring.GLScore(g, l)
		out = append(out, RankedStock{Ticker: m.snap.Ticker, CompanyName: m.snap.CompanyName,
			Sector: m.snap.Sector, Price: m.snap.Price, GLScore: gl, GrahamScore: g, LynchScore: l,
			GrahamValue: m.gv, MarginOfSafety: m.mos, PE: m.pe, PB: m.pb, PEG: m.peg,
			EPSGrowth: m.epsG, RevenueGrowth: m.revG, ROE: m.roe,
			DataDate: m.snap.DataDate.Format("2006-01-02"), CalculatedAt: now.Format("2006-01-02 15:04")})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].GLScore != out[j].GLScore {
			return out[i].GLScore > out[j].GLScore
		}
		if out[i].GrahamScore != out[j].GrahamScore {
			return out[i].GrahamScore > out[j].GrahamScore
		}
		if out[i].LynchScore != out[j].LynchScore {
			return out[i].LynchScore > out[j].LynchScore
		}
		if out[i].MarginOfSafety != out[j].MarginOfSafety {
			return out[i].MarginOfSafety > out[j].MarginOfSafety
		}
		ei, ej := 0.0, 0.0
		if out[i].EPSGrowth != nil {
			ei = *out[i].EPSGrowth
		}
		if out[j].EPSGrowth != nil {
			ej = *out[j].EPSGrowth
		}
		if ei != ej {
			return ei > ej
		}
		return out[i].Ticker < out[j].Ticker
	})
	if len(out) > 10 {
		out = out[:10]
	}
	for i := range out {
		out[i].Rank = i + 1
	}
	return RunResult{CalculatedAt: now.Format("2006-01-02 15:04"), Count: len(out), Stocks: out}
}

// GetTop5SpringateDistress calculates Springate scores across all snapshots,
// sorts ascending by score (highest financial distress risk first),
// and returns the top 5 stocks in distress zone (score < 0.862).
func GetTop5SpringateDistress(snaps []domain.Snapshot) []domain.SentinelStock {
	// Dedup ticker
	best := map[string]domain.Snapshot{}
	for _, s := range snaps {
		if p, ok := best[s.Ticker]; !ok || s.PublishedAt.After(p.PublishedAt) {
			best[s.Ticker] = s
		}
	}

	distressStocks := []domain.SentinelStock{}
	for _, s := range best {
		if s.Ticker == "" {
			continue
		}
		analysis := gemsentinel.ComputeSpringateScore(&s)
		
		distressLevel := "🟢 Healthy"
		if analysis.Score < gemsentinel.CutoffCritical {
			distressLevel = "🔴 Critical"
		} else if analysis.Score < gemsentinel.CutoffDistress {
			distressLevel = "🟠 Moderate"
		}

		vuln := gemsentinel.IdentifyPrimaryVulnerability(analysis)

		distressStocks = append(distressStocks, domain.SentinelStock{
			Ticker:               s.Ticker,
			CompanyName:          s.CompanyName,
			Sector:               s.Sector,
			Price:                s.Price,
			MarketCap:            s.MarketCap,
			Springate:            analysis,
			DistressLevel:        distressLevel,
			PrimaryVulnerability: vuln,
		})
	}

	// Sort ascending by Springate Score (lowest score = highest bankruptcy risk)
	sort.Slice(distressStocks, func(i, j int) bool {
		if distressStocks[i].Springate.Score != distressStocks[j].Springate.Score {
			return distressStocks[i].Springate.Score < distressStocks[j].Springate.Score
		}
		return distressStocks[i].Ticker < distressStocks[j].Ticker
	})

	// Filter only stocks in distress zone (< 0.862) or top 5 lowest if available
	filtered := []domain.SentinelStock{}
	for _, st := range distressStocks {
		if st.Springate.Score < gemsentinel.CutoffDistress {
			filtered = append(filtered, st)
		}
	}

	// Fallback to top 5 lowest if none strictly below cutoff, or take up to 5
	out := filtered
	if len(out) == 0 && len(distressStocks) > 0 {
		out = distressStocks
	}
	if len(out) > 5 {
		out = out[:5]
	}

	for i := range out {
		out[i].Rank = i + 1
		out[i].AISentinelSummary = gemsentinel.GenerateSentinelSummary(out[i])
	}

	return out
}
