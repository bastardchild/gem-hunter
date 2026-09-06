# HARNESS — Gem Hunter (Go + Fiber + HTMX + SQLite + Redis)

Acuan kerja agen/developer untuk stack ini. Prinsip: **CORRECTNESS > SIMPLICITY > PERFORMANCE > FEATURES**.

## 1. Prereqs

- Go (cek `go version`, target ≥ 1.21, pin di `go.mod`)
- Docker + Docker Compose v2
- `sqlite3`, `redis-cli` (opsional, untuk debug)
- `SECTORS_API_KEY` dari env, tidak pernah hard-code

## 2. Struktur Repo (wajib)

```
cmd/server/main.go
internal/{domain,application,repository,service,handler,middleware,scheduler}
internal/provider/sectors/   # SectorsProvider + DTO + mapping
internal/{scoring,cache,config}
web/{templates,static}
migrations/
tests/                       # fixtures + integration
docker/
docs/{architecture,scoring,data-source,deployment,backtesting}.md
.env.example
Dockerfile (multi-stage: builder → runtime)
docker-compose.yml (app, redis + litestream profil opsional, nonaktif di dev)
```

Aturan: logic bisnis HANYA di `application/domain/scoring`. Handler Fiber hanya validasi → service → response/template.

## 3. Env & Config

```bash
APP_ENV=development
SQLITE_PATH=data/gemhunter.db
LITESTREAM_ENABLED=false
REDIS_URL=redis://redis:6379/0
SECTORS_API_KEY=            # wajib, tanpa default
RANKING_INTERVAL_HOURS=6
MIN_MARKET_CAP=
MIN_EPS_GROWTH=
MAX_DATA_AGE_HOURS=
MIN_AVG_VALUE_20D=
ADMIN_TOKEN=                # untuk POST /admin/ranking/run
TZ=Asia/Jakarta
```

- Semua threshold via env/config, tidak hard-code.
- Sediakan `.env.example` tanpa secret.
- Presentasi pakai WIB (`Asia/Jakarta`), simpan DB dalam UTC.

## 4. Quickstart

```bash
# 1. init (sekali)
go mod init gemhunter   # jika belum ada
cp .env.example .env    # isi SECTORS_API_KEY + ADMIN_TOKEN

# 2. infra
docker compose up -d redis
docker compose ps

# 3. run lokal
go run ./cmd/server
# atau via compose:
docker compose up --build app

# 4. cek
curl localhost:3000/health
curl localhost:3000/ready
curl localhost:3000/ranking/top10
```

Healthcheck: `app` wajib `depends_on: redis` + `condition: service_healthy`, tunggu dependency siap (retry, bukan `sleep` buta). SQLite berupa file di volume `appdata` (WAL mode); Litestream nonaktif di dev.

## 5. Loop Kerja Per-Phase (wajib setelah tiap phase task.md §44)

```bash
gofmt -l .
go vet ./...
go build ./...
go test ./... -count=1
```

- Jangan lanjut phase berikutnya jika build/test merah.
- Ringkas: file berubah + formula yang disentuh + status test.

Urutan phase: arsitektur → Go+Docker+SQLite+Redis → provider/DTO/normalisasi → Graham/Lynch/scoring → ranking → scheduler/lock/cache → Fiber API → HTMX+Alpine → tests → docs.

## 6. Perintah Penting

```bash
# test spesifik
go test ./internal/scoring/ -run 'TestGraham|TestLynch|TestPEG|TestPercentile' -v
go test ./internal/provider/sectors/ -v
go test ./tests/ -tags=integration -v   # butuh SQLite+Redis, mock Sectors

# lint cepat
go vet ./...; gofmt -l .

# db (SQLite file di volume appdata)
docker run --rm -v gem-hunter_appdata:/data golang:1.25-bookworm ls -la /data
# redis
docker compose exec redis redis-cli KEYS 'gemhunter:*'
```

## 7. Kontrak Kunci (jangan dilanggar)

- `MarketDataProvider` interface di domain; `SectorsProvider` implementasi. Domain tidak import JSON Sectors langsung — via DTO → `CompanyFinancialSnapshot`.
- Null/invalid handling: EPS≤0 atau BVPS≤0 → `GrahamValue=null`, no Graham score; `EPSGrowth` denom≤0 → null; `PE≤0` atau `EPSGrowthPercent≤0` → `PEG=null`, Lynch tidak eligible.
- Percentile cross-sectional, satu metode konsisten (nearest-rank atau interpolated), didokumentasikan. Lower-is-better (PE, PB, PEG): `score=(1-p)*100`.
- Bobot: Graham `0.40 MOS + 0.20 PE + 0.15 PB + 0.15 EPSGrowth + 0.10 ROE`; Lynch `0.50 PEG + 0.25 EPSGrowth + 0.15 RevenueGrowth + 0.10 ROE`; Composite `0.55 Graham + 0.45 Lynch` (0–100).
- Bank/finansial: jangan filter DER ke bank; tandai N/A; renormalisasi bobot.
- Ranking sort: GLScore ↓, GrahamScore ↓, LynchScore ↓, MOS ↓, EPSGrowth ↓ → ambil Top 10.
- Scheduler internal (bukan HTTP trigger): `FETCH→VALIDATE→NORMALIZE→CACHE→CALCULATE→RANK→STORE→PUBLISH`, lock `gemhunter:ranking:lock` + TTL, 1 worker menang bila multi-replica.
- Cache keys: `sectors:company:{ticker}`, `sectors:financial:{ticker}`, `sectors:daily:{ticker}:{date}`, `gemhunter:ranking:latest`, `gemhunter:ranking:{run_id}`, `gemhunter:job:lock`. Pisahkan raw vs calculated. Stale → tampilkan `⚠ Data may be stale` + `Data updated: YYYY-MM-DD HH:mm WIB`.
- Endpoint: `GET /, /stocks, /stocks/:ticker, /ranking, /ranking/top10, /health, /ready`, `POST /admin/ranking/run` (wajib auth token). `GET /api/v1/ranking/top10` JSON sesuai §40.
- Frontend: SSR + HTMX (`hx-get="/ranking/top10" hx-trigger="every 60s"` baca cache, bukan recalc) + Alpine hanya UI state. Scoring 100% backend.
- Sectors error: timeout + exponential backoff + bounded retry; jangan retry 401/403; 429 hormati `Retry-After`; simpan structured log JSON.
- Backtest-ready: `RankingEngine`/`BacktestEngine` interface, input = snapshot (no `time.Now()`, no HTTP di engine); simpan `period_end, published_at, fetched_at`; syarat `available_at <= ranking_time`.

## 8. Observability & Security Checklist

- Log JSON: timestamp, level, request_id, component, event, ticker, duration_ms, error. API key tidak masuk log.
- Metrics: `ranking_run_total/success/failure`, `ranking_duration`, `sectors_api_requests/errors`, `cache_hit/miss`.
- Security: request ID, validasi input, timeout ctx, rate limit, security headers, admin auth, no stack trace di prod, no secret di repo.

## 9. Troubleshooting

| Gejala | Cek |
|---|---|
| `401/403 Sectors` | `echo ${#SECTORS_API_KEY}` kosong/salah; jangan retry loop |
| `429` | cek log `Retry-After`, backoff, cache fallback |
| Lock tidak lepas | `TTL` lock, `redis-cli GET gemhunter:ranking:lock` |
| Universe kosong | threshold env terlalu ketat; longgarkan `MIN_*`, cek eligibility log |
| Skor bank anjlok | pastikan profile finansial aktif, DER diabaikan |
| HTMX tidak refresh | pastikan fragment `GET /ranking/top10` return HTML parsial, bukan full page |
| Waktu salah | `TZ`, konversi UTC→WIB hanya di presentasi |

## 10. Acceptance Gate (sebelum klaim MVP done)

`go build` ✓, Fiber up ✓, SQLite+Redis connected ✓, compose up ✓, provider+mapping ✓, Graham/MOS/EPSGrowth/PEG/GrahamScore/LynchScore/GLScore ✓, percentile ✓, Top10 ✓, persist+cache ✓, scheduler 6h + lock ✓, dashboard HTMX+Alpine ✓, unit+integration hijau ✓, healthcheck ✓, docs + `.env.example` ✓, no secret ✓.
