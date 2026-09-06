package sectors

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"gemhunter/internal/domain"
)

// DTOs — hanya di layer ini. Domain tidak boleh import struct ini.
type screenerRow struct {
	Symbol         string   `json:"symbol"`
	CompanyName    string   `json:"company_name"`
	Sector         string   `json:"sector"`
	SubSector      string   `json:"sub_sector"`
	Industry       string   `json:"industry"`
	MarketCap      float64  `json:"market_cap"`
	LastClosePrice float64  `json:"last_close_price"`
	PETTM          *float64 `json:"pe_ttm"`
	PBMRQ          *float64 `json:"pb_mrq"`
	Dermrq         *float64 `json:"der_mrq"`
	ROETTM         *float64 `json:"roe_ttm"`
	YieldTTM       *float64 `json:"yield_ttm"`
}

type reportEnvelope struct {
	Symbol      string `json:"symbol"`
	CompanyName string `json:"company_name"`
	Overview    struct {
		Sector         string  `json:"sector"`
		SubSector      string  `json:"sub_sector"`
		Industry       string  `json:"industry"`
		MarketCap      float64 `json:"market_cap"`
		LastClosePrice float64 `json:"last_close_price"`
		LatestCloseDt  string  `json:"latest_close_date"`
	} `json:"overview"`
	Valuation struct {
		Historical []struct {
			Year int      `json:"year"`
			PE   *float64 `json:"pe"`
			PB   *float64 `json:"pb"`
		} `json:"historical_valuation"`
	} `json:"valuation"`
	Financials struct {
		EPS     *float64 `json:"eps"`
		HistEPS map[string]struct {
			EPS       *float64 `json:"eps"`
			EPSGrowth *float64 `json:"eps_growth"`
		} `json:"historical_eps"`
		HistFin []struct {
			Year             int      `json:"year"`
			Revenue          *float64 `json:"revenue"`
			Earnings         *float64 `json:"earnings"`
			TotalEquity      *float64 `json:"total_equity"`
			OutstandingShare *float64 `json:"outstanding_shares"`
			ROE              *float64 `json:"roe"`
		} `json:"historical_financials"`
	} `json:"financials"`
	Dividend struct {
		YieldTTM *float64 `json:"yield_ttm"`
	} `json:"dividend"`
}

// Client dengan timeout, bounded retry + backoff, hormati Retry-After untuk 429.
type Client struct {
	base   string
	key    string
	http   *http.Client
	maxTry int
}

func New(base, key string) *Client {
	return &Client{base: base, key: key, http: &http.Client{Timeout: 15 * time.Second}, maxTry: 3}
}

func (c *Client) get(ctx context.Context, path string, q url.Values) (int, []byte, http.Header, error) {
	var last error
	for attempt := 0; attempt < c.maxTry; attempt++ {
		u := c.base + path
		if q != nil {
			u += "?" + q.Encode()
		}
		req, _ := http.NewRequestWithContext(ctx, "GET", u, nil)
		req.Header.Set("Authorization", c.key)
		resp, err := c.http.Do(req)
		if err != nil {
			last = err
			time.Sleep(time.Duration(1<<attempt) * 300 * time.Millisecond)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			return resp.StatusCode, nil, resp.Header, fmt.Errorf("sectors auth %d: no retry", resp.StatusCode)
		}
		if resp.StatusCode == 429 {
			wait := 2 * time.Second
			if ra := resp.Header.Get("Retry-After"); ra != "" {
				if d, err := time.ParseDuration(ra + "s"); err == nil {
					wait = d
				}
			}
			select {
			case <-ctx.Done():
				return 429, nil, resp.Header, ctx.Err()
			case <-time.After(wait):
			}
			last = fmt.Errorf("429 rate limited")
			continue
		}
		if resp.StatusCode >= 500 {
			last = fmt.Errorf("sectors 5xx %d", resp.StatusCode)
			time.Sleep(time.Duration(1<<attempt) * 500 * time.Millisecond)
			continue
		}
		return resp.StatusCode, body, resp.Header, nil
	}
	return 0, nil, nil, last
}

// FetchSnapshot: overview+valuation+financials+dividend (4 section) lalu mapping ke domain.
func (c *Client) FetchSnapshot(ctx context.Context, symbol string) (domain.Snapshot, error) {
	q := url.Values{"sections": []string{"overview,valuation,financials,dividend"}}
	st, body, _, err := c.get(ctx, "/v2/company/report/"+url.PathEscape(symbol)+"/", q)
	if err != nil {
		return domain.Snapshot{}, err
	}
	if st == 404 {
		return domain.Snapshot{}, fmt.Errorf("symbol %s not found", symbol)
	}
	var rep reportEnvelope
	if err := json.Unmarshal(body, &rep); err != nil {
		return domain.Snapshot{}, fmt.Errorf("decode report: %w", err)
	}
	return MapReport(rep), nil
}

func fptr(v float64) *float64 { o := v; return &o }

// MapReport: Sectors DTO -> domain.Snapshot. BVPS dihitung (total_equity/outstanding_shares).
func MapReport(r reportEnvelope) domain.Snapshot {
	s := domain.Snapshot{Ticker: r.Symbol, CompanyName: r.CompanyName,
		Sector: r.Overview.Sector, Industry: r.Overview.Industry,
		Price: r.Overview.LastClosePrice, MarketCap: r.Overview.MarketCap, FetchedAt: time.Now().UTC()}
	if r.Overview.Sector == "Financials" {
		s.IsFinancial = true
	}
	if dt, err := time.Parse("2006-01-02", r.Overview.LatestCloseDt); err == nil {
		s.DataDate = dt
	}
	s.EPS = r.Financials.EPS
	s.DividendYield = r.Dividend.YieldTTM
	// historis: pilih 2 tahun terbaru
	type yr struct {
		y      int
		eps    *float64
		epsG   *float64
		rev    *float64
		eq     *float64
		sh     *float64
		pe, pb *float64
	}
	m := map[int]*yr{}
	for _, h := range r.Financials.HistFin {
		e := m[h.Year]
		if e == nil {
			e = &yr{y: h.Year}
			m[h.Year] = e
		}
		e.rev, e.eq, e.sh = h.Revenue, h.TotalEquity, h.OutstandingShare
	}
	for ys, he := range r.Financials.HistEPS {
		var y int
		fmt.Sscanf(ys, "%d", &y)
		e := m[y]
		if e == nil {
			e = &yr{y: y}
			m[y] = e
		}
		e.eps, e.epsG = he.EPS, he.EPSGrowth
	}
	for _, hv := range r.Valuation.Historical {
		e := m[hv.Year]
		if e == nil {
			e = &yr{y: hv.Year}
			m[hv.Year] = e
		}
		e.pe, e.pb = hv.PE, hv.PB
	}
	years := []int{}
	for y := range m {
		years = append(years, y)
	}
	// sort desc sederhana
	for i := 0; i < len(years); i++ {
		for j := i + 1; j < len(years); j++ {
			if years[j] > years[i] {
				years[i], years[j] = years[j], years[i]
			}
		}
	}
	if len(years) > 0 {
		cur := m[years[0]]
		if cur.eps != nil {
			s.EPS = cur.eps
		}
		if cur.pe != nil {
			s.PE = cur.pe
		}
		if cur.pb != nil {
			s.PB = cur.pb
		}
		if cur.eq != nil && cur.sh != nil && *cur.sh > 0 {
			s.BVPS = fptr(*cur.eq / *cur.sh)
		}
		if cur.rev != nil {
			s.Revenue = cur.rev
		}
		if len(years) > 1 {
			prv := m[years[1]]
			if prv.eps != nil {
				s.PrevEPS = prv.eps
			}
			if prv.rev != nil {
				s.PrevRevenue = prv.rev
			}
			s.PeriodEnd = time.Date(years[0], 12, 31, 0, 0, 0, 0, time.UTC)
			s.PublishedAt = s.PeriodEnd
		}
	}
	return s
}
