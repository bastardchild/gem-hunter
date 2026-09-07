package repository

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gemhunter/internal/service"

	_ "modernc.org/sqlite"
)

// Store: SQLite (WAL) sebagai persistent store + cache in-memory untuk latest.
// Litestream hanya replikasi file DB (sidecar opsional, nonaktif di dev).
type Store struct {
	mu     sync.RWMutex
	db     *sql.DB
	latest service.RunResult
	hist   []service.RunResult
}

func New(dbPath, schema string) (*Store, error) {
	if dir := filepath.Dir(dbPath); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	db, err := sql.Open("sqlite", dbPath+"?cache=shared")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000; PRAGMA foreign_keys=ON;`); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func f64(p *float64) sql.NullFloat64 {
	if p == nil {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: *p, Valid: true}
}

func (s *Store) Save(ctx context.Context, r service.RunResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := s.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO ranking_runs(run_id,started_at,finished_at,status,universe_size,eligible_count) VALUES(?,?,?,?,?,?)`,
		r.RunID, now, now, "SUCCESS", r.Count, r.Count); err != nil {
		return err
	}
	for _, st := range r.Stocks {
		if _, err := s.db.ExecContext(ctx,
			`INSERT INTO ranking_results(run_id,ticker,rank,gl_score,graham_score,lynch_score,graham_value,margin_of_safety,peg,eps_growth,revenue_growth,roe,price,data_date,calculated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			r.RunID, st.Ticker, st.Rank, st.GLScore, st.GrahamScore, st.LynchScore,
			st.GrahamValue, st.MarginOfSafety, f64(st.PEG), f64(st.EPSGrowth),
			f64(st.RevenueGrowth), f64(st.ROE), st.Price, st.DataDate, st.CalculatedAt); err != nil {
			return err
		}
	}
	s.latest = r
	s.hist = append(s.hist, r)
	if len(s.hist) > 100 {
		s.hist = s.hist[len(s.hist)-100:]
	}
	return nil
}

func (s *Store) Latest() (service.RunResult, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.latest.RunID == "" && len(s.latest.Stocks) == 0 {
		return s.latest, false
	}
	return s.latest, true
}

func (s *Store) ListRuns(ctx context.Context, limit int) ([]service.RunResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.hist) == 0 {
		if s.latest.RunID != "" {
			return []service.RunResult{s.latest}, nil
		}
		return nil, nil
	}
	out := make([]service.RunResult, 0, len(s.hist))
	for i := len(s.hist) - 1; i >= 0 && len(out) < limit; i-- {
		out = append(out, s.hist[i])
	}
	return out, nil
}

type Subscription struct {
	ID             int64  `json:"id"`
	Email          string `json:"email"`
	NotifyGems     bool   `json:"notify_gems"`
	NotifyGuard    bool   `json:"notify_guard"`
	NotifySentinel bool   `json:"notify_sentinel"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

func (s *Store) SaveSubscription(ctx context.Context, sub Subscription) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO email_subscriptions (email, notify_gems, notify_guard, notify_sentinel, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(email) DO UPDATE SET
			notify_gems = excluded.notify_gems,
			notify_guard = excluded.notify_guard,
			notify_sentinel = excluded.notify_sentinel,
			updated_at = excluded.updated_at
	`, sub.Email, sub.NotifyGems, sub.NotifyGuard, sub.NotifySentinel, now, now)
	return err
}

func (s *Store) GetSubscription(ctx context.Context, email string) (Subscription, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var sub Subscription
	row := s.db.QueryRowContext(ctx, `
		SELECT id, email, notify_gems, notify_guard, notify_sentinel, created_at, updated_at
		FROM email_subscriptions WHERE email = ?
	`, email)
	err := row.Scan(&sub.ID, &sub.Email, &sub.NotifyGems, &sub.NotifyGuard, &sub.NotifySentinel, &sub.CreatedAt, &sub.UpdatedAt)
	if err == sql.ErrNoRows {
		return sub, false, nil
	}
	if err != nil {
		return sub, false, err
	}
	return sub, true, nil
}

func (s *Store) ListSubscriptions(ctx context.Context) ([]Subscription, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, email, notify_gems, notify_guard, notify_sentinel, created_at, updated_at
		FROM email_subscriptions
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Subscription
	for rows.Next() {
		var sub Subscription
		if err := rows.Scan(&sub.ID, &sub.Email, &sub.NotifyGems, &sub.NotifyGuard, &sub.NotifySentinel, &sub.CreatedAt, &sub.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, sub)
	}
	return list, nil
}

func (s *Store) LogEmail(ctx context.Context, email, emailType, subject, status, errMsg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO email_logs (email, type, subject, status, error_msg, sent_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, email, emailType, subject, status, errMsg, now)
	return err
}
