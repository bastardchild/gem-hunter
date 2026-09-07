package sectors

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"time"

	"gemhunter/internal/domain"
)

const cacheSchema = `CREATE TABLE IF NOT EXISTS sectors_cache (
  ticker TEXT PRIMARY KEY,
  snapshot_json TEXT NOT NULL,
  fetched_at TEXT NOT NULL
);`

type Cache struct {
	db *sql.DB
}

func NewCache(db *sql.DB) (*Cache, error) {
	if _, err := db.Exec(cacheSchema); err != nil {
		return nil, fmt.Errorf("sectors cache schema: %w", err)
	}
	return &Cache{db: db}, nil
}

func (c *Cache) Get(ctx context.Context, ticker string) (*domain.Snapshot, error) {
	var snapJSON string
	var fetchedAt string
	err := c.db.QueryRowContext(ctx,
		`SELECT snapshot_json, fetched_at FROM sectors_cache WHERE ticker = ?`, ticker,
	).Scan(&snapJSON, &fetchedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var snap domain.Snapshot
	if err := json.Unmarshal([]byte(snapJSON), &snap); err != nil {
		return nil, err
	}
	return &snap, nil
}

func (c *Cache) IsFresh(ctx context.Context, ticker string, ttlHours int) bool {
	var fetchedAt string
	err := c.db.QueryRowContext(ctx,
		`SELECT fetched_at FROM sectors_cache WHERE ticker = ?`, ticker,
	).Scan(&fetchedAt)
	if err != nil {
		return false
	}
	t, err := time.Parse(time.RFC3339, fetchedAt)
	if err != nil {
		return false
	}
	return time.Since(t).Hours() < float64(ttlHours)
}

func (c *Cache) Put(ctx context.Context, ticker string, snap *domain.Snapshot) error {
	b, err := json.Marshal(snap)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = c.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO sectors_cache (ticker, snapshot_json, fetched_at) VALUES (?, ?, ?)`,
		ticker, string(b), now,
	)
	return err
}

func (c *Cache) PutBatch(ctx context.Context, snaps []*domain.Snapshot) error {
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := time.Now().UTC().Format(time.RFC3339)
	stmt, err := tx.PrepareContext(ctx,
		`INSERT OR REPLACE INTO sectors_cache (ticker, snapshot_json, fetched_at) VALUES (?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, snap := range snaps {
		b, err := json.Marshal(snap)
		if err != nil {
			return err
		}
		if _, err := stmt.ExecContext(ctx, snap.Ticker, string(b), now); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (c *Cache) GetAllFresh(ctx context.Context, ttlHours int) ([]domain.Snapshot, error) {
	cutoff := time.Now().UTC().Add(-time.Duration(ttlHours) * time.Hour).Format(time.RFC3339)
	rows, err := c.db.QueryContext(ctx,
		`SELECT snapshot_json FROM sectors_cache WHERE fetched_at > ?`, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Snapshot
	for rows.Next() {
		var snapJSON string
		if err := rows.Scan(&snapJSON); err != nil {
			return nil, err
		}
		var snap domain.Snapshot
		if err := json.Unmarshal([]byte(snapJSON), &snap); err != nil {
			continue
		}
		out = append(out, snap)
	}
	return out, rows.Err()
}

type screenerResult struct {
	Results []struct {
		Symbol      string `json:"symbol"`
		CompanyName string `json:"company_name"`
	} `json:"results"`
	Pagination struct {
		TotalCount int  `json:"total_count"`
		HasNext    bool `json:"has_next"`
		NextOffset int  `json:"next_offset"`
	} `json:"pagination"`
}

func (c *Client) FetchUniverse(ctx context.Context, minMarketCap float64, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 20
	}
	mcStr := strconv.FormatFloat(minMarketCap, 'f', 0, 64)
	q := url.Values{
		"where":    {fmt.Sprintf("market_cap > %s", mcStr)},
		"order_by": {"market_cap"},
		"limit":    {strconv.Itoa(limit)},
	}

	var allSymbols []string
	offset := 0
	for {
		pq := url.Values{}
		for k, vs := range q {
			for _, v := range vs {
				pq.Set(k, v)
			}
		}
		pq.Set("offset", strconv.Itoa(offset))

		st, body, _, err := c.get(ctx, "/v2/companies/", pq)
		if err != nil {
			log.Printf(`{"level":"error","event":"sectors_universe","error":%q}`, err.Error())
			break
		}
		if st >= 400 {
			log.Printf(`{"level":"error","event":"sectors_universe_status","status":%d}`, st)
			break
		}

		var result screenerResult
		if err := json.Unmarshal(body, &result); err != nil {
			log.Printf(`{"level":"error","event":"sectors_universe_decode","error":%q}`, err.Error())
			break
		}

		if len(result.Results) == 0 {
			break
		}

		for _, r := range result.Results {
			if r.Symbol != "" {
				allSymbols = append(allSymbols, r.Symbol)
			}
		}

		if len(allSymbols) >= limit || !result.Pagination.HasNext {
			break
		}
		offset += len(result.Results)
	}

	if len(allSymbols) > limit {
		allSymbols = allSymbols[:limit]
	}
	return allSymbols, nil
}

func (cl *Client) FetchSnapshotWithCache(ctx context.Context, cache *Cache, ticker string, ttlHours int) (domain.Snapshot, bool, error) {
	if cache != nil && cache.IsFresh(ctx, ticker, ttlHours) {
		snap, err := cache.Get(ctx, ticker)
		if err == nil && snap != nil {
			return *snap, true, nil
		}
	}

	snap, err := cl.FetchSnapshot(ctx, ticker)
	if err != nil {
		if cache != nil {
			snap, err2 := cache.Get(ctx, ticker)
			if err2 == nil && snap != nil {
				return *snap, true, nil
			}
		}
		return domain.Snapshot{}, false, err
	}

	if cache != nil {
		if err := cache.Put(ctx, ticker, &snap); err != nil {
			log.Printf(`{"level":"warn","event":"sectors_cache_put","ticker":%q,"error":%q}`, ticker, err.Error())
		}
	}

	return snap, false, nil
}

func (cl *Client) FetchLiveUniverse(ctx context.Context, cache *Cache, minMarketCap float64, maxStocks int, ttlHours int) ([]domain.Snapshot, error) {
	symbols, err := cl.FetchUniverse(ctx, minMarketCap, maxStocks)
	if err != nil {
		log.Printf(`{"level":"error","event":"sectors_universe_fetch","error":%q}`, err.Error())
		if cache != nil {
			fresh, cacheErr := cache.GetAllFresh(ctx, ttlHours)
			if cacheErr == nil && len(fresh) > 0 {
				log.Printf(`{"level":"warn","event":"sectors_universe_fallback_cache","count":%d}`, len(fresh))
				return fresh, nil
			}
		}
		return nil, err
	}

	log.Printf(`{"level":"info","event":"sectors_universe_fetched","count":%d}`, len(symbols))

	var snaps []domain.Snapshot
	for _, sym := range symbols {
		snap, stale, fetchErr := cl.FetchSnapshotWithCache(ctx, cache, sym, ttlHours)
		if fetchErr != nil {
			log.Printf(`{"level":"warn","event":"sectors_snapshot_fail","ticker":%q,"error":%q}`, sym, fetchErr.Error())
			continue
		}
		if !stale {
			time.Sleep(200 * time.Millisecond)
		}
		snaps = append(snaps, snap)
	}

	log.Printf(`{"level":"info","event":"sectors_universe_deep_done","count":%d}`, len(snaps))
	return snaps, nil
}
