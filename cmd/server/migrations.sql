CREATE TABLE IF NOT EXISTS companies (
  ticker TEXT PRIMARY KEY, company_name TEXT, sector TEXT, industry TEXT
);
CREATE TABLE IF NOT EXISTS financial_snapshots (
  id INTEGER PRIMARY KEY AUTOINCREMENT, ticker TEXT REFERENCES companies(ticker),
  price REAL, eps REAL, bvps REAL,
  pe REAL, pb REAL, roe REAL,
  data_date TEXT, fetched_at TEXT
);
CREATE TABLE IF NOT EXISTS daily_prices (
  ticker TEXT, date TEXT, close REAL, volume INTEGER, market_cap REAL,
  PRIMARY KEY (ticker, date)
);
CREATE TABLE IF NOT EXISTS ranking_runs (
  run_id TEXT PRIMARY KEY, started_at TEXT, finished_at TEXT,
  status TEXT, universe_size INTEGER, eligible_count INTEGER
);
CREATE TABLE IF NOT EXISTS ranking_results (
  id INTEGER PRIMARY KEY AUTOINCREMENT, run_id TEXT REFERENCES ranking_runs(run_id),
  ticker TEXT, rank INTEGER, gl_score REAL, graham_score REAL,
  lynch_score REAL, graham_value REAL, margin_of_safety REAL,
  peg REAL, eps_growth REAL, revenue_growth REAL,
  roe REAL, price REAL, data_date TEXT, calculated_at TEXT
);
CREATE INDEX IF NOT EXISTS idx_results_ticker ON ranking_results(ticker);
CREATE INDEX IF NOT EXISTS idx_results_run ON ranking_results(run_id);
CREATE INDEX IF NOT EXISTS idx_results_rank ON ranking_results(rank);
CREATE INDEX IF NOT EXISTS idx_results_calc ON ranking_results(calculated_at);

CREATE TABLE IF NOT EXISTS email_subscriptions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  email TEXT UNIQUE NOT NULL,
  notify_gems BOOLEAN DEFAULT 1,
  notify_guard BOOLEAN DEFAULT 1,
  notify_sentinel BOOLEAN DEFAULT 1,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS email_logs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  email TEXT NOT NULL,
  type TEXT NOT NULL,
  subject TEXT NOT NULL,
  status TEXT NOT NULL,
  error_msg TEXT,
  sent_at TEXT NOT NULL
);
