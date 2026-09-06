# SKILL — Gem Hunter Quantitative Screening (Go/Fiber Stack)

Skill untuk agen yang mengimplementasikan Gem Hunter: screening + ranking saham BEI berbasis Graham + Lynch via Sectors.app V2.

## Kapan dipakai

Pakai skill ini ketika: menambah/mengubah provider Sectors, normalisasi snapshot, formula scoring, eligibility, ranking, scheduler/cache, endpoint Fiber, template HTMX/Alpine, atau test/docs terkait.

## Stack & Batasan (tidak boleh diganti tanpa alasan teknis kuat)

- Backend: Go + Fiber + Go modules. Frontend: HTML SSR + HTMX + Alpine.js (+ Tailwind bila perlu). Tanpa React/Vue/Next.
- Infra: Docker multi-stage + Compose (app, redis + litestream opsional). DB: SQLite (WAL) persisten; Redis hanya cache/lock/rate-limit/koordinasi.
- Test: `go testing` + `httptest` + integration (SQLite+Redis) + mock Sectors. Obs: structured log + request ID + `/health` + `/ready` + metrics.

## Arsitektur

Clean/Hexagonal pragmatis. Handler = validasi → application service → response/template. Nol rumus di handler/JS/template.

```
Sectors DTO → CompanyFinancialSnapshot → Graham/Lynch engines → Percentile → Scores → Eligibility → Ranking → Store/Cache/Publish
```

Snapshot wajib: ticker, name, sector, industry, price, marketCap, EPS, prevEPS, BVPS, PE, PB, ROE, DER, revenue, prevRevenue, dividendYield, dataDate, fetchedAt (+ `period_end/published_at` anti look-ahead bias).

## Data Source — Sectors V2

- Key hanya dari `SECTORS_API_KEY`. Jangan tebak endpoint/field — baca dok resmi V2 dulu, buat tabel mapping `Sectors field → domain field → formula`. Yang tak tersedia: cari endpoint lain → hitung dari field ada → tandai `unavailable`, jangan karang.
- Interface:
```go
type MarketDataProvider interface {
  GetCompanies(ctx context.Context) ([]Company, error)
  GetCompanyFinancials(ctx context.Context, ticker string) (Financials, error)
  GetCompanyValuation(ctx context.Context, ticker string) (Valuation, error)
  GetDailyData(ctx context.Context, ticker string, from, to time.Time) ([]DailyBar, error)
}
```

## Formula (sumber kebenaran)

- Graham: `GrahamValue = sqrt(22.5 × EPS × BVPS)`; null bila EPS≤0/BVPS≤0. `MOS = (GrahamValue − Price)/GrahamValue`.
- EPSGrowth: `EPS_t/EPS_prev − 1` (CAGR bila multi-periode); null bila denom≤0. RevenueGrowth analog.
- Lynch: `PEG = PE / EPSGrowthPercent` (growth dalam persen, cth 20 bukan 0.20). Null bila PE≤0 atau growth≤0 → Lynch turun/tidak eligible.
- Percentile cross-sectional 0–100: higher-is-better `p×100`; lower-is-better `(1−p)×100`. Satu metode (nearest-rank/interpolated), konsisten + dokumentasikan.
- Skor:
  - Graham = `0.40 MOS + 0.20 PE + 0.15 PB + 0.15 EPSGrowth + 0.10 ROE`
  - Lynch = `0.50 PEG + 0.25 EPSGrowth + 0.15 RevenueGrowth + 0.10 ROE`
  - GL = `0.55 Graham + 0.45 Lynch` (0–100). Label: 90+ Exceptional, 80+ Strong, 70+ Attractive, 60+ Neutral, <60 Weak.
- GLScore adalah composite milik Gem Hunter, BUKAN formula resmi Graham/Lynch. Bahasa: "quantitative screening score / ranking signal", wajib disclaimer bukan rekomendasi investasi.

## Eligibility, Bank, Ranking

- Filter via config env (`MIN_EPS_GROWTH, MAX_DATA_AGE, MIN_MARKET_CAP, MIN_AVG_VALUE_20D`): ticker/price/EPS valid, fundamental lengkap, tidak stale, PE>0, GrahamValue valid, EPSGrowth>0 untuk Lynch. Jangan terlalu ketat sampai universe habis.
- Bank/finansial: profile terpisah (`GrahamLynchFinancialProfile`), abaikan DER, tandai N/A, renormalisasi bobot — jangan hukum bank karena balance sheet berbeda.
- Sort: GLScore, GrahamScore, LynchScore, MOS, EPSGrowth (semua ↓) → Top 10 `RankedStock{rank,ticker,name,sector,price,GL,Graham,Lynch,GrahamValue,MOS,PE,PB,PEG,EPSGrowth,RevenueGrowth,ROE,breakdown,dataDate,calculatedAt}`.
- Kualitas data: `DataQualityScore{completeness,freshness,consistency}`; di bawah threshold → exclude / `unavailable`, jangan skor palsu. Simpan tiap run (`ranking_runs`, `ranking_results`) untuk history/backtest. Backtest wajib `available_at <= ranking_time` + biaya (fee, pajak, slippage, spread).

## Scheduler, Cache, API, UI

- Scheduler 6 jam internal: `FETCH→VALIDATE→NORMALIZE→CACHE→CALCULATE→RANK→STORE→PUBLISH` + lock `gemhunter:ranking:lock` (TTL anti-duplikat). Job state: RunID, Started/FinishedAt, RUNNING/SUCCESS/FAILED + universe/eligible/top10/durasi/latensi/error count.
- Redis keys: `sectors:company:{ticker}`, `sectors:financial:{ticker}`, `sectors:daily:{ticker}:{date}`, `gemhunter:ranking:latest`, `gemhunter:ranking:{run_id}`, `gemhunter:job:lock`. Pisahkan raw vs calculated; saat Sectors gagal pakai cache dalam freshness window + flag stale.
- Endpoint: `GET /, /stocks, /stocks/:ticker, /ranking, /ranking/top10, /health, /ready`; `GET /api/v1/ranking/top10` JSON (§40 task.md); `POST /admin/ranking/run` wajib auth.
- UI: SSR, tabel Top 10 (rank, ticker, company, price, GL, Graham, Lynch, MOS, PEG, EPS Growth, status), `hx-get="/ranking/top10" hx-trigger="every 60s"` baca cache, Alpine hanya expand/filter/modal/loading, detail saham + breakdown bar + penjelasan percentile + `Data updated … WIB` + flag stale + disclaimer.

## Error, Config, Keamanan

- Sectors: ctx timeout + backoff eksponensial + retry terbatas; no-retry 401/403; 429 hormati Retry-After; log JSON (timestamp, level, request_id, component, event, ticker, duration_ms, error) tanpa API key.
- Config 100% env (lihat harness.md). Docker: pin versi, healthcheck, app tunggu Redis.
- Security: validasi input, timeout, rate limit, security headers, admin auth, tanpa stack trace prod, tanpa secret di repo.

## Testing & Docs (wajib)

- Unit: `GrahamValue, MarginOfSafety, EPSGrowth, PEG, PercentileScore, GrahamScore, LynchScore, GLScore, Ranking, Eligibility` + edge: EPS/BVPS/PE/growth ≤0, nil, div-by-zero, duplikat ticker, universe kosong/satu saham.
- Provider: fixture JSON V2 (valid, missing/null/unexpected field, 401/429/500, timeout). Integration: API→service→repo→SQLite→Redis dengan mock provider, tidak hit API asli.
- Docs: `docs/{architecture,scoring,data-source,deployment,backtesting}.md` — tulis arah skor (higher: MOS/growth/ROE; lower: PE/PB/PEG), metode percentile, mapping field, dan fase ekstensi (multi-factor → risk parity → behavioral → backtest/paper-trading) sebagai interface saja.

## Pitfall Umum

- PEG pakai desimal (0.20) bukan persen (20) → salah 100x.
- Skor = raw value tanpa percentile → dilarang.
- Threshold absolut PE/PB sebagai satu-satunya filter → dilarang.
- Growth palsu saat denom ≤0; skor saat GrahamValue null; DER bank disamakan; recalc tiap request HTMX; klaim "pasti untung"; look-ahead bias di backtest.
