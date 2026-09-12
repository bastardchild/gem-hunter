# Gem Hunter

### Gem Hunter: Find the signal. Check the risk.

**Powered by Sectors.app**

Gem Hunter is a decision-support platform for the Indonesia Stock Exchange (IDX). It uses **Sectors.app Financial API V2** as a core market-data source and turns company fundamentals into a recurring, explainable workflow:

1. Fetch a focused universe of Indonesian issuers from Sectors.
2. Filter unusable or stale fundamentals.
3. Rank candidates with a deterministic Graham + Lynch factor model.
4. Explain each result with a guardrailed analysis layer.
5. Monitor abnormal price/volume behaviour with Gem Guard.
6. Surface potential financial distress with Gem Sentinel.
7. Deliver the latest signals through a dashboard, JSON API, and optional email digests.

> **Sectors Hackathon 2026:** declared track **Market Intelligence**. The product also demonstrates the **Automation & Workflows** pattern through scheduled ranking, asynchronous analysis, and alert delivery.

> **Disclaimer:** Gem Hunter is a quantitative research and screening tool. It is not financial advice, does not guarantee returns, and does not place or automate trades.

## Why this matters

Indonesian investors can access data, but turning a large market into a repeatable research shortlist is still slow. A raw quote or financial profile is not a decision workflow. Gem Hunter adds the missing layer:

- **A consistent definition of “interesting”:** value and growth are scored with published formulas instead of opaque opinions.
- **Context before conviction:** every candidate is accompanied by its factor scores, source dates, risks, and missing-data notes.
- **Protection against false positives:** a cheap stock can still be exposed to thin-float price impact, abnormal volume, or balance-sheet distress.
- **A workflow that keeps running:** rankings refresh on a schedule and can notify subscribers without a manual cycle-by-cycle process.

The result is not a buy list. It is a compact research queue: **what deserves attention, why it ranked, and what could invalidate the signal.**

## Product tour

| Surface | What a user gets |
|---|---|
| **Gem Hunter Dashboard** | Live Top 10 IDX ranking with price, composite score, Graham/Lynch components, Margin of Safety, PEG, EPS growth, and expandable analysis. |
| **Gem Guard** | A surveillance view for abnormal price impact, volume spikes, Auto-Rejection Atas proximity, consecutive ARA days, insider selling, leverage, RSI, and volatility. |
| **Gem Sentinel** | A Top 5 financial-distress watchlist using Springate MDA, with ratio-level explanations and a primary vulnerability for each issuer. |
| **Methodology** | Human-readable formulas, weights, percentile semantics, eligibility rules, and treatment of financial-sector data. |
| **5-Year Proof** | A transparent historical illustration using BRIS, ANTM, HRUM, ESSA, and ITMG, with public price history and explicit limitations. |
| **Email alerts** | Optional Top 10, high-risk Gem Guard, and critical Gem Sentinel digests through SMTP. |
| **JSON API** | Consume rankings, surveillance results, ticker details, and AI analyses from integrations or scripts. |

## The core insight: value plus growth, with risk beside it

Gem Hunter deliberately avoids a one-factor screen:

- A **value-only** screen can find businesses that are cheap because fundamentals are deteriorating.
- A **growth-only** screen can find excellent businesses priced for perfection.
- A **single ranking** can hide market-structure and solvency risks.

Gem Hunter combines a value-led Graham score with a growth-at-a-reasonable-price Lynch score, then presents surveillance and distress signals alongside the result. This makes the product useful at the moment a researcher asks: **“Why is this stock interesting, and what should I be cautious about?”**

## How the ranking works

All ranking mathematics is implemented as pure, unit-tested Go functions in `internal/scoring` and `internal/service`.

### 1. Candidate hygiene

A snapshot must have usable, positive inputs before it can enter the ranking:

- price, EPS, BVPS, and PE must be available and valid;
- the snapshot must be within `MAX_DATA_AGE_HOURS` when freshness filtering is enabled;
- EPS growth must clear `MIN_EPS_GROWTH`;
- duplicate tickers are reduced to the most recently published snapshot.

### 2. Fundamental features

```text
GrahamValue = sqrt(22.5 * EPS * BVPS)
MarginOfSafety = 1 - Price / GrahamValue
EPSGrowth = EPS / PreviousEPS - 1
RevenueGrowth = Revenue / PreviousRevenue - 1
PEG = PE / (EPSGrowth * 100)
```

### 3. Relative percentile scoring

Raw metrics are converted to 0-100 cross-sectional percentiles across the eligible universe. Lower-is-better metrics such as PE, PB, and PEG are inverted. A score of 80 means “better than roughly 80% of this run’s eligible universe,” not “80% likely to go up.”

### 4. Composite model

```text
GrahamScore = 0.40*MoS + 0.20*PE + 0.15*PB + 0.15*EPSGrowth + 0.10*ROE
LynchScore  = 0.50*PEG + 0.25*EPSGrowth + 0.15*RevenueGrowth + 0.10*ROE
GLScore     = 0.55*GrahamScore + 0.45*LynchScore
```

The 55/45 blend intentionally gives value a small lead while requiring growth confirmation. Financial issuers can have structurally different disclosures; when PB or ROE is unavailable, the engine uses a neutral percentile and preserves fair comparison rather than treating missing data as poor performance.

| GL score | Label |
|---:|---|
| 90+ | Exceptional |
| 80-89.9 | Strong |
| 70-79.9 | Attractive |
| 60-69.9 | Neutral |
| Below 60 | Weak |

## Gem Guard: do not confuse cheap with safe

Gem Guard is a surveillance layer inspired by IDX market mechanics. It is not a claim that manipulation has occurred; it is a transparent set of indicators that tells the user when a price move deserves investigation.

| Signal | Calculation or rule | Why it matters |
|---|---|---|
| ARA boundary | 20%, 25%, or 35% depending on price band | Puts a move in the context of IDX Auto-Rejection Atas limits. |
| Trading velocity | `AverageDailyVolume / FreeFloat` | Normalizes participation by available float. |
| Price Impact Ratio | `abs(PriceChange%) / Velocity` | High values indicate a large move on relatively thin participation. |
| Volume spike | `LatestVolume / AverageVolume20` | Flags unusual participation. |
| Price spike | `((P_t - P_t-1) / P_t-1) * 100` | Identifies abrupt directional movement. |
| Other context | insider selling, DER, current ratio, RSI, volatility, moving-average ratio | Separates a market anomaly from a broader risk picture. |

The output is a 0-100 composite risk score:

- **LOW:** below 40
- **MEDIUM:** 40-69
- **HIGH:** 70+

The UI also exposes the primary risk factors and an `UMA suspected` flag when configured threshold combinations are met. These are screening indicators for human review, not accusations or regulatory determinations.

## Gem Sentinel: financial distress radar

Gem Sentinel applies the Springate Multiple Discriminant Analysis model to available financial-statement inputs:

```text
Score = 1.03*X1 + 3.07*X2 + 0.66*X3 + 0.40*X4

X1 = WorkingCapital / TotalAssets
X2 = EBIT / TotalAssets
X3 = ProfitBeforeTax / CurrentLiabilities
X4 = Revenue / TotalAssets
```

| Springate score | Zone |
|---:|---|
| Below 0.500 | Critical |
| 0.500 to below 0.862 | Moderate |
| 0.862+ | Healthy |

The product ranks the most vulnerable names, shows the four ratios, and explains the main weakness such as working-capital deficit, operating loss, liability coverage, or low asset turnover. A low score is a monitoring signal, not a prediction of bankruptcy.

## AI that explains the numbers without owning the numbers

Gem Hunter’s AI layer is an application-owned pipeline, not a chat client with a prompt:

```text
Ranking run -> Screener triage -> Analyst thesis -> Risk flags -> Async cache/API/UI
```

The analyst produces structured output:

- summary;
- strengths;
- risks;
- why the stock ranked;
- bull case and bear case;
- confidence;
- missing data;
- risk flags.

The system is safe to demo and cheap to run:

- with no model key, deterministic `gemhunter-deterministic-v1` analysis keeps the app fully functional;
- with an OpenAI-compatible provider configured, the deterministic analysis can be enriched with qualitative prose;
- guardrails reject altered scores, invented numbers, unsupported certainty, and direct buy/sell language;
- stale data is explicitly marked;
- the asynchronous worker prevents slow LLM requests from blocking a ranking run.

Supported integration patterns include OpenAI-compatible endpoints, OpenAI, Anthropic, Google, Mistral, Groq, Ollama, OpenRouter, vLLM, and LM Studio, subject to the configured client/provider settings.

## Sectors integration

Sectors is part of the product’s core workflow, not a decorative data call.

When live mode is enabled, Gem Hunter uses `SECTORS_API_KEY` and the Sectors V2 client in `internal/provider/sectors` to build the screening universe. The integration includes:

- company and financial snapshot mapping into a canonical domain model;
- configurable market-cap and universe-size limits;
- SQLite-backed caching with a configurable TTL;
- live refresh before scheduled ranking cycles;
- a fallback mock universe for local development and zero-credit demos;
- isolated provider code so the scoring engine remains deterministic and testable.

By default, the example configuration targets a focused universe of up to 20 issuers with a 7-day cache TTL. This makes API-credit usage visible and intentional during a hackathon demo. Set `LIVE_SECTORS_ENABLED=false` to run locally without making live Sectors requests.

## End-to-end workflow

```text
Sectors V2 API
      |
      v
Cached issuer snapshots -> eligibility filter -> percentile factors -> Top 10
                                                               |
                         +-------------------------------------+------------------+
                         |                                                        |
                         v                                                        v
                 AI explanation                                      email digest / JSON API

Market and fundamental context -> Gem Guard surveillance
Financial statement ratios     -> Gem Sentinel distress radar
```

The server runs an initial ranking at startup, then schedules new cycles at 00:00, 06:00, 12:00, and 18:00 WIB. Each cycle persists its results to SQLite, enqueues AI analysis, and dispatches matching alert digests.

## Quick start

### Requirements

- Docker and Docker Compose, or Go 1.25+;
- a Sectors API key for live data;
- an LLM key only if qualitative model enrichment is desired;
- SMTP credentials only if email alerts are desired.

### Run with Docker

```bash
cp .env.example .env
docker compose up -d --build
```

Open `http://localhost:3000`.

For a free offline demo, leave `SECTORS_API_KEY` empty and set:

```env
LIVE_SECTORS_ENABLED=false
LLM_ENABLED=false
EMAIL_ALERTS_ENABLED=false
```

The app then boots against its built-in mock universe and deterministic analysis fallback.

### Run natively

```bash
go run ./cmd/server
```

The application creates and migrates the SQLite database at `SQLITE_PATH` on startup.

### Optional replication profile

```bash
docker compose --profile replication up -d
```

This starts Litestream replication alongside the app and Redis services. Configure the destination in `litestream.yml` before using it in production.

## Screens and API

### Web routes

| Route | Purpose |
|---|---|
| `/` | Gem Hunter Top 10 dashboard; the ranking table refreshes through HTMX. |
| `/gemguard` | Anti-manipulation surveillance view. |
| `/gemsentinel` | Springate distress watchlist. |
| `/methodology` | Formula and model documentation. |
| `/proof` | Historical illustration and reproducibility context. |
| `/health` | Liveness check. |
| `/ready` | Readiness check. |

### JSON endpoints

| Method | Endpoint | Purpose |
|---|---|---|
| `GET` | `/api/v1/ranking/top10` | Latest ranked Top 10. |
| `GET` | `/ranking` | Latest complete run. |
| `GET` | `/stocks` | Ranked stock array. |
| `GET` | `/stocks/:ticker` | Ranked details for one ticker; `.JK` is added when omitted. |
| `GET` | `/api/v1/gemguard` | Gem Guard surveillance array. |
| `GET` | `/api/v1/gemsentinel` | Gem Sentinel watchlist. |
| `GET` | `/api/v1/ai/analysis/:ticker` | Latest structured analysis for a ticker. |
| `GET` | `/api/v1/ai/run/:run_id` | Analyses generated for a ranking run. |
| `POST` | `/api/v1/subscribe` | Create or update email channel preferences. |
| `POST` | `/api/v1/email/test` | Verify SMTP delivery. |
| `POST` | `/admin/ranking/run` | Force a ranking run with a bearer token. |
| `POST` | `/admin/ai/run` | Re-enqueue the latest run for analysis. |

Example:

```bash
curl http://localhost:3000/api/v1/ranking/top10

curl -X POST http://localhost:3000/api/v1/subscribe \
  -H "Content-Type: application/json" \
  -d '{"email":"investor@example.com","notify_gems":true,"notify_guard":true,"notify_sentinel":false}'

curl -X POST http://localhost:3000/admin/ranking/run \
  -H "Authorization: Bearer change-me"
```

## Configuration

Copy `.env.example` to `.env`. The most important settings are:

| Variable | Default | Description |
|---|---|---|
| `PORT` | `3000` | HTTP port. |
| `SQLITE_PATH` | `data/gemhunter.db` | SQLite database path. |
| `SECTORS_API_KEY` | empty | Enables the Sectors data source when live mode is on. |
| `SECTORS_BASE_URL` | `https://api.sectors.app` | Sectors API base URL. |
| `LIVE_SECTORS_ENABLED` | `true` | Live Sectors switch; an API key is also required. |
| `UNIVERSE_MIN_MARKET_CAP` | `50000000000000` | Minimum market cap for the live universe. |
| `UNIVERSE_MAX_TICKERS` | `20` | Maximum live issuers fetched per refresh. |
| `SECTORS_CACHE_TTL_HOURS` | `168` | Live-universe cache TTL. |
| `RANKING_INTERVAL_HOURS` | `6` | Ranking cadence; schedule is aligned to WIB slots. |
| `MIN_EPS_GROWTH` | `0` | Minimum EPS growth as a fraction, e.g. `0.10` for 10%. |
| `MAX_DATA_AGE_HOURS` | `168` | Maximum accepted snapshot age. |
| `ADMIN_TOKEN` | `change-me` | Bearer token for admin endpoints; replace it outside local demos. |
| `LLM_ENABLED` | `true` | Enables model enrichment when a compatible key is present. |
| `OPENAI_COMPATIBLE_API_KEY` | empty | Optional LLM API key. |
| `OPENAI_COMPATIBLE_BASE_URL` | empty | Optional OpenAI-compatible endpoint. |
| `LLM_DEFAULT_MODEL` | `gpt-4o-mini` | Model name sent to the configured endpoint. |
| `EMAIL_ALERTS_ENABLED` | `true` | Enables scheduled digests when SMTP is configured. |
| `SMTP_HOST` / `SMTP_PORT` | empty / `587` | SMTP server and port. |
| `SMTP_USER` / `SMTP_PASSWORD` | empty | SMTP credentials. |
| `SMTP_FROM` | `Gem Hunter <alerts@gemhunter.app>` | Verified sender header. |
| `TZ` | empty | Set to `Asia/Jakarta` for local scheduler/log alignment. |

Never commit `.env`, API keys, SMTP passwords, or `cookies.txt`. The repository’s `.gitignore` is intended to keep local secrets and generated data out of version control; review the final public repository before submission.

## Architecture

```text
                    +----------------------+
                    | Sectors.app V2 API  |
                    +----------+-----------+
                               |
                    +----------v-----------+
                    | Provider + cache    |
                    +----------+-----------+
                               |
                    +----------v-----------+
                    | Go server / Fiber   |
                    | scheduler + routes  |
                    +-----+----------+-----+
                          |          |
              +-----------v--+    +-v----------------+
              | Quant engine |    | AI worker         |
              | scoring      |    | screener/analyst  |
              | ranking      |    | risk + guardrails |
              +------+-------+    +---------+----------+
                     |                       |
              +------v-----------------------v------+
              | SQLite WAL repository               |
              | runs, results, snapshots, emails    |
              +----------------+--------------------+
                               |
               +---------------+----------------+
               |                                |
        HTMX + Go templates                SMTP digests
```

The code follows a pragmatic hexagonal boundary:

- `cmd/server`: composition root, scheduler, HTTP routes, and embedded assets;
- `internal/provider/sectors`: Sectors API client, DTO mapping, retry/cache boundary;
- `internal/domain`: canonical financial snapshot and distress types;
- `internal/scoring`: pure Graham, Lynch, PEG, percentile, and composite functions;
- `internal/service`: eligibility, deduplication, ranking, and Springate orchestration;
- `internal/gemguard`: surveillance calculations and risk scoring;
- `internal/gemsentinel`: Springate calculations and vulnerability summaries;
- `internal/ai`: screener, analyst, risk engine, tools, guardrails, and async worker;
- `internal/repository`: SQLite persistence;
- `internal/notifier`: SMTP test flow and scheduled digests;
- `web`: embedded templates and static CSS.

## Technology

- Go 1.25
- Fiber v2
- SQLite through `modernc.org/sqlite` with WAL mode and no CGO requirement
- HTMX and Alpine.js for lightweight interactive UI
- Go `html/template` with embedded assets
- Redis 7 as an optional cache service
- Litestream as an optional SQLite replication profile
- Distroless non-root container runtime
- Go standard-library SMTP client

## Testing and reproducibility

Run the full test suite:

```bash
go test ./...
```

Coverage includes:

- Graham, Lynch, PEG, percentile, and zero-division invariants;
- eligibility filtering, deduplication, ordering, and bank-aware scoring;
- ARA boundaries, PIR, velocity, spikes, volatility, and RSI;
- Springate ratios and distress cutoffs;
- AI output structure and guardrail validation.

The ranking function accepts snapshots, a clock value, and configuration explicitly. It does not perform HTTP or call `time.Now()` inside the scoring formula, which makes unit tests, repeatable demos, and future point-in-time backtests possible.

## Historical proof, honestly framed

The `/proof` page demonstrates the model against five IDX names using public Yahoo Finance chart data and historical financial inputs: BRIS, ANTM, HRUM, ESSA, and ITMG. It shows the period, price-only gain, formula rationale, and what the illustration cannot establish.

This is intentionally not presented as a guarantee or a fully bias-free portfolio backtest. A production research version should add point-in-time fundamentals, survivorship controls, transaction costs, dividends, benchmark comparison, and a pre-registered rebalance rule. The product makes that limitation visible instead of hiding it behind a headline return.

## Project status

### Implemented

- Sectors V2 provider boundary and cached live-universe flow
- Graham + Lynch Top 10 ranking engine
- Automated WIB-aligned scheduler
- Gem Guard surveillance calculations
- Gem Sentinel Springate distress radar
- Deterministic AI analysis with optional LLM enrichment
- AI guardrails and asynchronous processing
- Email subscriptions, test email, and scheduled digests
- Dashboard, methodology, proof page, and JSON API
- SQLite persistence, Docker deployment, Redis service, and optional Litestream profile

### Next improvements

- Point-in-time fundamentals cache for bias-free historical research
- Quarterly rebalance backtest versus IHSG with CAGR, Sharpe, hit rate, and drawdown
- A unified confluence score across ranking, Guard, and Sentinel
- Intraday anomaly alerts during market hours
- Telegram and Discord delivery
- Stronger authentication and subscription verification for production use

## Hackathon submission notes

The product is designed to satisfy the Sectors Hackathon’s core product test:

- **Sectors is essential:** removing Sectors snapshots removes the live screening universe and the core market-intelligence workflow.
- **Derived insight is the product:** the app produces custom scores, rankings, anomaly indicators, distress classifications, and synthesized explanations rather than merely re-displaying raw data.
- **The core workflow is end to end:** fetch, filter, score, explain, persist, display, and alert.
- **Automation is visible:** scheduled runs, timestamps, persisted run IDs, async processing, and email dispatch are part of the server lifecycle.
- **It is decision support, not execution:** Gem Hunter never places or automates buy/sell orders.

Recommended judging-video sequence:

1. Start with the problem: Indonesian market data is available, but a researcher needs a repeatable shortlist with context.
2. Show the live Sectors-backed dashboard and open one ranked row.
3. Point to the exact Graham/Lynch scores, percentile meaning, and AI guardrails.
4. Open Gem Guard to show how a “cheap” stock can carry market-structure warnings.
5. Open Gem Sentinel to show the separate solvency lens.
6. Show the scheduler/log or a forced run, then show the resulting JSON/API or email workflow.
7. Close with the disclaimer and the product’s honest limits.

## Attribution and license

Built for the [Sectors Hackathon 2026](https://hackathon.sectors.app/).

- Market data: [Sectors.app Financial API V2](https://sectors.app)
- Historical chart data for the proof page: Yahoo Finance public chart API
- Financial concepts: Benjamin Graham’s number, Peter Lynch’s PEG discipline, Springate MDA, and IDX Auto-Rejection mechanics
- UI font: Inter by Rasmus Andersson

No license file is currently included. Add the project license that matches the team’s intended distribution before publishing a long-term public release.
