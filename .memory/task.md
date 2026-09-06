# GEM HUNTER — MASTER ENGINEERING PROMPT

Anda adalah **Principal Software Engineer, Quantitative Developer, dan Software Architect** yang juga memahami fundamental analysis saham Indonesia.

Bangun aplikasi bernama **Gem Hunter**, sebuah quantitative stock screening dan ranking engine untuk saham Bursa Efek Indonesia (BEI).

Tujuan utama aplikasi:

> Menghasilkan **Top 10 saham BEI terbaik berdasarkan kombinasi metode Benjamin Graham + Peter Lynch**, menggunakan data dari **Sectors.app Financial API V2**, dengan ranking otomatis dan pembaruan data setiap 6 jam.

Aplikasi harus production-oriented, modular, testable, observable, dan mudah dikembangkan menjadi multi-factor engine di masa depan.

---

# 1. TECHNOLOGY STACK

Gunakan stack berikut dan jangan menggantinya kecuali ada alasan teknis yang sangat kuat:

Backend:

* Go
* Fiber
* Go modules

Frontend:

* HTML server-rendered
* HTMX
* Alpine.js
* Tailwind CSS jika diperlukan
* Jangan menggunakan React/Vue/Next.js

Infrastructure:

* Docker
* Docker Compose
* Redis

Database:

* Gunakan SQLite (+ Litestream, nonaktif di dev) untuk persistent data.
* Redis digunakan sebagai cache, distributed lock, rate-limit protection, dan job coordination.

External data:

* Sectors.app Financial API V2

Testing:

* Go testing
* httptest
* integration tests
* Redis integration test
* API mock

Observability:

* structured logging
* request ID
* health check
* readiness check
* metrics jika memungkinkan

---

# 2. ARSITEKTUR

Gunakan Clean Architecture / Hexagonal Architecture yang pragmatis.

Struktur minimal:

cmd/
server/
main.go

internal/
domain/
application/
repository/
service/
handler/
middleware/
scheduler/
provider/
sectors/
scoring/
cache/
config/

web/
templates/
static/

migrations/

tests/

docker/

Jangan membuat business logic di Fiber handler.

Handler hanya:

HTTP Request
→ validation
→ application service
→ response/template

Business logic harus berada di application/domain/scoring layer.

---

# 3. DATA SOURCE — SECTORS.APP V2

Gunakan **Sectors.app Financial API V2** sebagai primary external data source.

API key HARUS berasal dari environment variable:

SECTORS_API_KEY

Jangan pernah hard-code API key.

Buat interface:

type MarketDataProvider interface {
GetCompanies(ctx context.Context) (...)
GetCompanyFinancials(ctx context.Context, ticker string) (...)
GetCompanyValuation(ctx context.Context, ticker string) (...)
GetDailyData(ctx context.Context, ticker string, ...) (...)
}

Implementasi:

SectorsProvider

Jangan membuat domain layer bergantung langsung kepada response JSON Sectors.

Buat DTO khusus provider kemudian mapping:

Sectors DTO
→ Domain Model

---

# 4. DATA NORMALIZATION

Jangan langsung memasukkan raw data Sectors ke formula.

Buat canonical model:

CompanyFinancialSnapshot

Minimal memiliki:

Ticker
CompanyName
Sector
Industry

Price
MarketCap

EPS
PreviousEPS

BookValuePerShare

PE
PB

ROE
DER

Revenue
PreviousRevenue

DividendYield

DataDate
FetchedAt

Pastikan setiap metric memiliki informasi freshness/data date.

Tangani:

* null
* zero
* negative EPS
* negative book value
* missing financial data
* stale data
* API error
* delisted/suspended ticker jika tersedia

---

# 5. GRAHAM ENGINE

Implementasikan Graham Number:

GrahamValue = sqrt(22.5 * EPS * BVPS)

Jika EPS <= 0 atau BVPS <= 0:

GrahamValue = null

Margin of Safety:

MOS = (GrahamValue - Price) / GrahamValue

atau:

MOS = 1 - Price / GrahamValue

Jangan menghasilkan score Graham jika intrinsic value tidak valid.

---

# 6. GRAHAM SCORE

Gunakan percentile ranking antar saham yang eligible.

Semua komponen dinormalisasi menjadi 0–100.

Untuk faktor yang semakin tinggi semakin baik:

Score(x) = PercentileRank(x) * 100

Untuk faktor yang semakin rendah semakin baik:

Score(x) = (1 - PercentileRank(x)) * 100

GrahamScore:

GrahamScore =
0.40 * MOSScore
+ 0.20 * PEScore
+ 0.15 * PBScore
+ 0.15 * EPSGrowthScore
+ 0.10 * ROEScore

Catatan:

PE dan PB adalah lower-is-better.

EPS Growth dan ROE adalah higher-is-better.

MOS adalah higher-is-better.

Jangan menggunakan threshold PE/PB secara absolut sebagai satu-satunya mekanisme seleksi.

---

# 7. EPS GROWTH

Jika tersedia EPS historis yang cukup:

EPSGrowth =
(EPS_t / EPS_t-n)^(1/n) - 1

Untuk YoY sederhana:

EPSGrowth =
EPS_t / EPS_previous - 1

Jika denominator <= 0:

EPSGrowth = null

Jangan memberikan growth score palsu.

---

# 8. REVENUE GROWTH

RevenueGrowth =
Revenue_t / Revenue_previous - 1

Jika previous revenue <= 0:

RevenueGrowth = null

---

# 9. PETER LYNCH ENGINE

Gunakan:

PEG = PE / EPSGrowthPercent

PENTING:

Jika EPSGrowthPercent <= 0:
PEG = null
LynchScore harus turun / tidak eligible.

Jika PE <= 0:
PEG = null.

Karena growth dinyatakan dalam percentage points.

Contoh:

PE = 15
EPS Growth = 20%

PEG = 15 / 20
PEG = 0.75

Jangan menggunakan 0.20 sebagai denominator dalam formula PEG tersebut.

---

# 10. LYNCH SCORE

Gunakan:

LynchScore =
0.50 * PEGScore
+ 0.25 * EPSGrowthScore
+ 0.15 * RevenueGrowthScore
+ 0.10 * ROEScore

PEG adalah lower-is-better.

EPS Growth adalah higher-is-better.

Revenue Growth adalah higher-is-better.

ROE adalah higher-is-better.

---

# 11. GRAHAM + LYNCH COMPOSITE

Buat composite score:

GLScore =
0.55 * GrahamScore
+ 0.45 * LynchScore

Range:

0 <= GLScore <= 100

Interpretasi:

90–100 = Exceptional
80–89.99 = Strong
70–79.99 = Attractive
60–69.99 = Neutral
<60 = Weak

Jangan menyebut score sebagai "jaminan keuntungan".

Gunakan istilah:

"quantitative screening score"

atau

"ranking signal".

---

# 12. ELIGIBILITY FILTER

Sebelum ranking:

Stock harus memenuhi:

* valid ticker
* valid price
* valid EPS
* valid fundamental data
* bukan data stale
* EPS growth > 0 untuk Lynch
* PE > 0
* GrahamValue valid

Tambahkan konfigurasi threshold melalui environment/config sehingga tidak hard-coded.

Contoh:

MIN_EPS_GROWTH
MAX_DATA_AGE
MIN_MARKET_CAP
MIN_AVG_VALUE_20D

Jangan membuat semua filter mandatory jika menyebabkan universe terlalu kecil.

---

# 13. SPECIAL HANDLING UNTUK BANK / FINANCIAL SECTOR

Jangan menganggap DER perusahaan bank sama dengan DER perusahaan non-financial.

Buat:

ScoringProfile

misalnya:

GrahamLynchNonFinancialProfile
GrahamLynchFinancialProfile

Untuk tahap pertama, jika data/rumus bank belum tervalidasi:

* jangan memaksakan DER filter kepada bank
* tandai metric sebagai N/A
* renormalisasi bobot komponen yang tersedia

Jangan membuat bank otomatis mendapat score rendah hanya karena karakteristik balance sheet-nya berbeda.

---

# 14. RANKING ENGINE

Setelah seluruh saham mendapat GLScore:

sort descending berdasarkan:

1. GLScore
2. GrahamScore
3. LynchScore
4. MOS
5. EPSGrowth

Kemudian ambil:

TOP 10

Buat model:

RankedStock

yang berisi:

Rank
Ticker
CompanyName
Sector
Price

GLScore
GrahamScore
LynchScore

GrahamValue
MarginOfSafety

PE
PB
PEG

EPSGrowth
RevenueGrowth
ROE

ScoreBreakdown
DataDate
CalculatedAt

---

# 15. SIX-HOUR REFRESH ENGINE

Gem Hunter harus melakukan recalculation setiap 6 jam.

Gunakan scheduler internal atau worker architecture.

Jangan membuat HTTP request menjadi scheduler.

Pipeline:

FETCH
→ VALIDATE
→ NORMALIZE
→ CACHE
→ CALCULATE
→ RANK
→ STORE
→ PUBLISH

Gunakan Redis distributed lock:

gemhunter:ranking:lock

TTL harus mencegah duplicate execution.

Jika aplikasi dijalankan menggunakan multiple container replicas, hanya satu worker yang boleh menjalankan ranking job.

---

# 16. JOB STATES

Setiap ranking execution mempunyai:

RunID
StartedAt
FinishedAt
Status

Status:

RUNNING
SUCCESS
FAILED

Simpan metadata:

UniverseSize
EligibleCount
Top10Count
Duration
ProviderLatency
ErrorCount

---

# 17. REDIS

Gunakan Redis untuk:

1. API response cache
2. calculated ranking cache
3. distributed lock
4. rate-limit protection
5. short-lived market data cache

Contoh keys:

sectors:company:{ticker}
sectors:financial:{ticker}
sectors:daily:{ticker}:{date}

gemhunter:ranking:latest
gemhunter:ranking:{run_id}

gemhunter:job:lock

Jangan menggunakan Redis sebagai satu-satunya persistent database.

---

# 18. CACHE STRATEGY

Pisahkan:

RAW DATA CACHE

dan

CALCULATED RESULT CACHE.

Jika Sectors API gagal:

* gunakan cached data jika masih dalam acceptable freshness window
* tandai result sebagai stale
* jangan diam-diam menyajikan data lama sebagai data real-time

UI harus menampilkan:

Data updated:
YYYY-MM-DD HH:mm WIB

dan jika stale:

⚠ Data may be stale

---

# 19. API ENDPOINTS

Minimal:

GET /
GET /stocks
GET /stocks/:ticker
GET /ranking
GET /ranking/top10
GET /health
GET /ready

Admin/internal:

POST /admin/ranking/run

POST endpoint harus protected.

Jangan expose trigger job tanpa authentication jika production.

---

# 20. HTMX FRONTEND

Jangan membuat SPA.

Gunakan server-side rendered HTML.

Dashboard harus menampilkan:

GEM HUNTER

Last update:
06 Sep 2026 18:00 WIB

Next update:
00:00 WIB

TOP 10

Rank
Ticker
Company
Price
GL Score
Graham
Lynch
MOS
PEG
EPS Growth
Risk/Data status

Gunakan HTMX untuk refresh fragment:

GET /ranking/top10

Contoh:

hx-get="/ranking/top10"
hx-trigger="every 60s"
hx-swap="innerHTML"

Tetapi jangan menghitung ulang ranking setiap request.

HTMX hanya membaca cached latest result.

---

# 21. ALPINE.JS

Gunakan Alpine.js hanya untuk UI state ringan:

* expandable stock detail
* score breakdown
* filter
* modal
* loading state

Jangan memindahkan business logic scoring ke JavaScript.

Scoring harus 100% berada di backend.

---

# 22. STOCK DETAIL

Ketika user klik saham:

Tampilkan:

Company
Ticker
Sector

Current Price

Graham Value
Margin of Safety

PE
PB
PEG

EPS Growth
Revenue Growth
ROE

Graham Score
Lynch Score
GL Score

Score breakdown:

MOS       ██████████ 82
PE        █████████  78
PB        █████████  75
Growth    █████████  81
ROE       ████████   72

Jelaskan bahwa score adalah relative percentile terhadap eligible universe.

---

# 23. API ERROR HANDLING

Sectors API dapat mengalami:

* timeout
* rate limit
* 401
* 403
* 404
* 429
* 500
* malformed response

Implementasikan:

context timeout
exponential backoff
bounded retry
structured logging

Jangan retry 401/403 secara membabi buta.

Untuk 429:

respect Retry-After jika tersedia.

---

# 24. DATABASE

Gunakan SQLite (WAL mode, pure-Go via modernc.org/sqlite).

Minimal tabel:

companies
financial_snapshots
daily_prices
ranking_runs
ranking_results

ranking_results:

id
run_id
ticker
rank
gl_score
graham_score
lynch_score
graham_value
margin_of_safety
peg
eps_growth
revenue_growth
roe
price
data_date
calculated_at

Tambahkan indexes untuk:

ticker
run_id
rank
calculated_at

---

# 25. HISTORICAL RANKING

Jangan hanya menyimpan Top 10 terbaru.

Simpan setiap ranking run.

Dengan demikian kita dapat membuat:

* ranking history
* score history
* stock movement
* backtesting
* model evaluation

Nantinya dapat menjawab:

"Apakah saham yang masuk Top 10 benar-benar outperform?"

---

# 26. BACKTESTING FOUNDATION

Walaupun backtesting belum menjadi fitur utama versi pertama, desain harus memungkinkan.

Buat interface:

RankingEngine

dan:

BacktestEngine

Jangan membuat ranking engine bergantung pada current time atau HTTP.

Input ranking engine harus berupa data snapshot.

Dengan demikian kita bisa melakukan:

Historical Snapshot
→ Graham/Lynch
→ Ranking
→ Forward Return

untuk menguji model.

---

# 27. AVOID LOOK-AHEAD BIAS

Ini WAJIB.

Jika laporan keuangan baru tersedia pada tanggal tertentu, jangan menggunakan data tersebut untuk ranking pada tanggal sebelum data tersedia.

Simpan:

period_end
published_at
fetched_at

Untuk backtesting:

available_at <= ranking_time

harus menjadi syarat.

---

# 28. TRANSACTION COST

Backtest nantinya harus memperhitungkan:

broker fee
tax
slippage
bid/ask spread jika data tersedia

Jangan membandingkan raw return tanpa transaction cost dan mengklaim strategi profitable.

---

# 29. TIMEZONE

Gunakan:

Asia/Jakarta

untuk business logic.

Simpan timestamp database dalam UTC jika memungkinkan.

Presentation:

WIB

Jangan menggunakan local server timezone sebagai sumber kebenaran.

---

# 30. CONFIGURATION

Gunakan environment variables.

Contoh:

APP_ENV=development

SQLITE_PATH=...
LITESTREAM_ENABLED=false
REDIS_URL=...

SECTORS_API_KEY=...

RANKING_INTERVAL_HOURS=6

MIN_MARKET_CAP=...
MIN_EPS_GROWTH=...
MAX_DATA_AGE_HOURS=...

Tidak ada secret di source code.

Buat:

.env.example

tanpa secret asli.

---

# 31. DOCKER

Buat Dockerfile multi-stage.

Stage:

builder
→ runtime

docker-compose.yml minimal:

app
redis
(litestream hanya profil opsional, nonaktif di dev)

Gunakan healthcheck.

App harus menunggu dependency siap.

Jangan menggunakan:

latest

untuk production image jika bisa dihindari.

Pin major/minor versions yang masuk akal.

---

# 32. SECURITY

Implementasikan minimal:

* API key tidak masuk log
* request ID
* input validation
* timeout
* rate limiting
* security headers
* protected admin endpoint
* no stack traces in production response

Jangan menyimpan credential di repository.

---

# 33. OBSERVABILITY

Structured log JSON:

timestamp
level
request_id
component
event
ticker
duration_ms
error

Metrics minimal:

ranking_run_total
ranking_run_success
ranking_run_failure
ranking_duration
sectors_api_requests
sectors_api_errors
cache_hit
cache_miss

---

# 34. TESTING

WAJIB buat unit test untuk:

GrahamValue()

MarginOfSafety()

EPSGrowth()

PEG()

PercentileScore()

GrahamScore()

LynchScore()

GLScore()

Ranking()

Eligibility()

Test edge cases:

EPS <= 0
BVPS <= 0
PE <= 0
growth <= 0
nil data
division by zero
duplicate ticker
empty universe
only one eligible stock

Tambahkan integration test:

API
→ service
→ repository
→ SQLite
→ Redis

Gunakan mock provider untuk Sectors.

Jangan membuat test bergantung pada API Sectors sungguhan.

---

# 35. DATA PROVIDER TESTING

Buat fixture JSON berdasarkan response Sectors V2.

Test:

valid response
missing field
null field
unexpected field
HTTP 401
HTTP 429
HTTP 500
timeout

Jangan membuat production calculation bergantung langsung pada JSON provider.

---

# 36. API DOCUMENTATION

Buat dokumentasi:

docs/architecture.md
docs/scoring.md
docs/data-source.md
docs/deployment.md
docs/backtesting.md

Jelaskan setiap formula secara matematis.

---

# 37. SCORING DOCUMENTATION

Dokumentasikan:

Graham:

GrahamValue = sqrt(22.5 × EPS × BVPS)

MOS = (GrahamValue - Price) / GrahamValue

GrahamScore =
0.40 MOS

* 0.20 PE
* 0.15 PB
* 0.15 EPS Growth
* 0.10 ROE

Lynch:

PEG = PE / EPSGrowthPercent

LynchScore =
0.50 PEG

* 0.25 EPS Growth
* 0.15 Revenue Growth
* 0.10 ROE

Composite:

GLScore =
0.55 GrahamScore
+ 0.45 LynchScore

Pastikan dokumentasi menjelaskan arah score:

Higher is better:
MOS
EPS Growth
Revenue Growth
ROE

Lower is better:
PE
PB
PEG

---

# 38. IMPORTANT — PERCENTILE IMPLEMENTATION

Jangan melakukan:

score = raw_value

Gunakan cross-sectional percentile.

Contoh:

Jika saham berada pada percentile ke-90 untuk ROE:

ROEScore = 90

Jika PE berada pada percentile ke-10:

PEScore = 90

karena PE lebih rendah lebih baik.

Pastikan metode percentile konsisten.

Dokumentasikan metode yang digunakan:

nearest-rank atau interpolated percentile.

Gunakan satu metode secara konsisten.

---

# 39. FINANCIAL DATA QUALITY

Buat DataQualityScore.

Minimal:

Completeness
Freshness
Consistency

Jika data fundamental tidak lengkap:

jangan memberikan score palsu.

Contoh:

DataQuality < threshold
→ stock excluded

atau:

score = unavailable

sesuai konfigurasi.

---

# 40. OUTPUT TOP 10

API:

GET /api/v1/ranking/top10

Return JSON:

{
"run_id": "...",
"calculated_at": "...",
"data_as_of": "...",
"count": 10,
"stocks": [
{
"rank": 1,
"ticker": "...",
"company_name": "...",
"price": 0,
"gl_score": 0,
"graham_score": 0,
"lynch_score": 0,
"graham_value": 0,
"margin_of_safety": 0,
"peg": 0,
"eps_growth": 0,
"revenue_growth": 0,
"roe": 0
}
]
}

Jangan mengembalikan internal API key atau raw provider credentials.

---

# 41. UI PRINCIPLE

Dashboard harus sederhana.

Prioritas visual:

1. Top 10
2. GL Score
3. Graham Score
4. Lynch Score
5. MOS
6. PEG
7. Growth
8. Data freshness

Gunakan badge:

🟢 Strong
🟡 Neutral
🔴 Weak

Tetapi jangan menggunakan warna sebagai satu-satunya informasi.

---

# 42. INVESTMENT DISCLAIMER

Dashboard harus memiliki disclaimer:

"Gem Hunter adalah alat quantitative screening dan bukan rekomendasi investasi, jaminan keuntungan, atau pengganti analisis pribadi."

Jangan menggunakan bahasa:

"pasti naik"
"pasti untung"
"jaminan profit"

Gunakan:

"ranking"
"signal"
"screening"
"score"

---

# 43. FUTURE EXTENSION

Arsitektur harus memungkinkan penambahan:

Phase 2:

* Simons-style multi-factor
* momentum
* quality
* value
* low volatility

Phase 3:

* Risk Parity
* HRP
* portfolio allocation

Phase 4:

* behavioral finance
* herding
* foreign flow
* abnormal volume
* anomaly detection

Phase 5:

* backtesting
* paper trading
* performance attribution

Jangan implementasikan seluruh phase tersebut sekarang.

Buat interface dan boundaries agar mudah ditambahkan.

---

# 44. DEVELOPMENT WORKFLOW

Jangan langsung menghasilkan ribuan baris kode.

Kerjakan bertahap:

PHASE 1
→ inspect requirements
→ propose architecture
→ identify Sectors V2 endpoints/fields needed

PHASE 2
→ initialize Go project
→ Docker
→ SQLite (+ Litestream, dev nonaktif)
→ Redis

PHASE 3
→ Sectors provider
→ DTO
→ normalization

PHASE 4
→ Graham engine
→ Lynch engine
→ scoring engine

PHASE 5
→ ranking engine

PHASE 6
→ scheduler
→ Redis lock
→ caching

PHASE 7
→ Fiber API

PHASE 8
→ HTMX + Alpine dashboard

PHASE 9
→ tests

PHASE 10
→ documentation

Setelah setiap phase:

1. run tests
2. run gofmt
3. run go vet
4. verify build
5. inspect changed files
6. summarize what changed

Jangan melanjutkan jika build/test rusak tanpa memperbaikinya.

---

# 45. VERY IMPORTANT — SECTORS V2 DOCUMENTATION

Sebelum menulis integration code, pelajari dokumentasi resmi Sectors.app V2.

Jangan menebak nama endpoint atau JSON field.

Buat mapping:

Sectors V2 field
→ canonical domain field
→ formula

Contoh:

provider field
→ EPS
→ GrahamValue

provider field
→ BVPS
→ GrahamValue

provider field
→ PE
→ PEG

provider field
→ historical EPS
→ EPSGrowth

Jika suatu data tidak tersedia secara langsung di Sectors V2:

1. cari endpoint V2 lain yang relevan
2. hitung dari field yang tersedia
3. jika tetap tidak tersedia, tandai sebagai unavailable

Jangan mengarang field API.

---

# 46. SOURCE OF TRUTH

Untuk API behavior:

Sectors.app official V2 documentation.

Untuk formula:

Dokumentasikan sumber akademik yang relevan.

Pisahkan:

DATA SOURCE
dan
ACADEMIC MODEL

Jangan mengklaim bahwa formula gabungan GLScore adalah formula resmi Benjamin Graham atau Peter Lynch.

GLScore adalah:

"Gem Hunter composite scoring model based on Graham and Lynch principles."

---

# 47. ACCEPTANCE CRITERIA

Aplikasi dianggap selesai untuk MVP jika:

[ ] Go application build
[ ] Fiber server running
[ ] SQLite connected (file DB + WAL aktif)
[ ] Redis connected
[ ] Docker Compose works
[ ] Sectors V2 provider implemented
[ ] Raw data normalized
[ ] Graham Number calculated
[ ] MOS calculated
[ ] EPS Growth calculated
[ ] PEG calculated
[ ] Graham Score calculated
[ ] Lynch Score calculated
[ ] GL Score calculated
[ ] percentile ranking implemented
[ ] Top 10 generated
[ ] ranking persisted
[ ] Redis cache working
[ ] six-hour scheduler working
[ ] distributed lock working
[ ] HTMX dashboard working
[ ] Alpine interactions working
[ ] API endpoints working
[ ] unit tests passing
[ ] integration tests passing
[ ] Docker healthchecks passing
[ ] documentation complete
[ ] no secrets committed

---

# 48. FIRST TASK

Jangan langsung membuat seluruh aplikasi.

Mulai dengan:

1. Inspect repository.
2. Inspect existing files.
3. Inspect current Go version.
4. Inspect existing Docker configuration.
5. Determine SQLite path / existing DB file.
6. Determine whether Redis already exists.
7. Read official Sectors.app V2 documentation.
8. Identify exact endpoints and fields required for Graham + Lynch.
9. Produce an implementation plan.
10. Show proposed directory structure.
11. Show exact data mapping.
12. Show scoring formulas.
13. Identify ambiguities or unavailable data.

Setelah itu implementasikan MVP secara incremental.

Prioritas utama:

**CORRECTNESS > SIMPLICITY > PERFORMANCE > FEATURES**

Jangan menambahkan fitur yang belum diperlukan.

# END MASTER PROMPT
