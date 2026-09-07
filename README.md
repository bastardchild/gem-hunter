# Gem Hunter - Quantitative Stock Screening & Surveillance Platform

A full-stack quantitative equity screening, surveillance, and alerting platform for the **Indonesia Stock Exchange (IDX)**, built for the [Sectors Hackathon 2026](https://hackathon.sectors.app/). Gem Hunter combines classic value investing mathematics (Benjamin Graham + Peter Lynch) with modern AI-assisted analysis, anti-manipulation surveillance, financial distress detection, and automated email alerting into a single production-grade Go application.

> **Disclaimer:** Gem Hunter is a quantitative screening and research tool. It is **not** financial advice, a guarantee of returns, or a substitute for personal due diligence.

---

## Table of Contents

1. [Overview](#overview)
2. [Key Features](#key-features)
3. [Architecture](#architecture)
4. [The Quant Engine: Graham + Lynch Factor Model](#the-quant-engine-graham--lynch-factor-model)
5. [Modules](#modules)
   - [Gem Hunter (Top 10 Ranking)](#gem-hunter-top-10-ranking)
   - [Gem Guard (Anti-Manipulation Surveillance)](#gem-guard-anti-manipulation-surveillance)
   - [Gem Sentinel (Financial Distress Radar)](#gem-sentinel-financial-distress-radar)
   - [AI Agent Layer](#ai-agent-layer)
   - [Email Alert System](#email-alert-system)
   - [5-Year Proof Page](#5-year-proof-page)
6. [Screens & Routes](#screens--routes)
7. [REST API Reference](#rest-api-reference)
8. [Tech Stack](#tech-stack)
9. [Project Structure](#project-structure)
10. [Configuration (Environment Variables)](#configuration-environment-variables)
11. [Running Locally](#running-locally)
12. [Docker Deployment](#docker-deployment)
13. [Testing](#testing)
14. [Database Schema](#database-schema)
15. [Methodology & Scientific Basis](#methodology--scientific-basis)
16. [Roadmap](#roadmap)
17. [License & Attribution](#license--attribution)

---

## Overview

Gem Hunter answers one question with math: **which IDX stocks are objectively cheap relative to their fundamentals AND growing?**

Instead of relying on opinions, the platform:

1. **Screens** the universe with a hybrid Graham (value) + Lynch (growth-at-fair-price) percentile factor model.
2. **Ranks** candidates into a Top 10 list recomputed automatically 4x per day (00:00, 06:00, 12:00, 18:00 WIB).
3. **Explains** every score with AI-generated, guardrailed analysis (no hallucinated numbers).
4. **Watches** the market for manipulation patterns (Gem Guard) and bankruptcy risk (Gem Sentinel).
5. **Alerts** subscribers via email digests whenever a new screening cycle completes.
6. **Proves** the method with 5 years of real historical market data on the `/proof` page.

---

## Key Features

| Feature | Description |
|---|---|
| **Graham-Lynch Factor Model** | Deterministic 0-100 composite score: `GL = 0.55*Graham + 0.45*Lynch`, percentile-ranked across the eligible universe |
| **Automated Scheduler** | Ranking runs automatically at 00:00, 06:00, 12:00, 18:00 WIB (configurable via `RANKING_INTERVAL_HOURS`) |
| **AI Analyst Layer** | Multi-agent pipeline (Screener triage -> Analyst -> Risk Engine) with strict guardrails against score hallucination and advisory language |
| **Gem Guard** | Anti-manipulation surveillance: PIR, volume spikes, price spikes vs IDX Auto-Rejection boundaries, insider selling, RSI, volatility |
| **Gem Sentinel** | Springate Multiple Discriminant Analysis (MDA) bankruptcy/distress radar with Top 5 most vulnerable watchlist |
| **Email Alerts** | Per-user subscription modal (no auth required for demo): Top 10 Gems, Gem Guard High Risk, Gem Sentinel Distress digests via any SMTP provider |
| **Test Email** | One-click SMTP connectivity verification from the UI |
| **5-Year Proof** | Historical validation page: five real stocks that passed the screen in 2020 and their verified gains (Yahoo Finance public data) |
| **Bank-Aware Scoring** | Automatic renormalization for financial-sector issuers (missing DER/PB handled gracefully) |
| **Zero-Cost Fallback** | Fully functional deterministic AI fallback (no LLM key needed); mock universe when `SECTORS_API_KEY` is absent |
| **Production Database** | SQLite (WAL) with Litestream replication profile for disaster recovery |
| **HTMX Live Updates** | Rankings table auto-refreshes every 60s without full page reloads |
| **Modern Animations** | Spring-scale modal, backdrop blur transitions, bell-shake micro-interactions, ambient glow |

---

## Architecture

```
                        +---------------------------+
                        |      Sectors.app V2       |
                        |   (market data provider)  |
                        +------------+--------------+
                                     |
+-------------+          +-----------v------------+          +---------------+
|  Scheduler  |--------->|   Go Server (Fiber)    |<---------|  Admin Token  |
| 00/06/12/18 |          |  cmd/server            |          |  (protected)  |
+-------------+          +---+---------------+----+          +---------------+
                             |               |
              +--------------v--+        +---v-------------------+
              | Quant Engine    |        |  AI Agent Layer       |
              | internal/service|        |  internal/ai          |
              | internal/scoring|        |  screener/analyst/risk|
              +------+----------+        +---+-------------------+
                     |                       |
        +------------v------------+   +------v------+
        | SQLite (WAL) Repository |   | LLM Provider|
        | internal/repository     |   | (optional)  |
        +------------+------------+   +-------------+
                     |
     +---------------+----------------+
     |                                |
+----v---------+              +-------v--------+
| Web UI       |              | Email Notifier |
| HTMX+Alpine  |              | internal/notifier
| web/templates|              | SMTP (any)     |
+--------------+              +----------------+
```

**Data flow per scheduled run:**

1. Scheduler wakes at the next 00/06/12/18 WIB slot.
2. `service.Rank()` builds snapshots (Sectors API in production, mock universe in dev) and computes deterministic scores.
3. Results persist to SQLite (`ranking_runs`, `ranking_results`).
4. The AI worker enqueues the run: Screener triages, Analyst + Risk Engine produce per-ticker analyses (LLM if configured, deterministic templates otherwise).
5. The mailer dispatches personalized digest emails to every subscriber matching their alert preferences.
6. HTMX clients polling `/ranking/top10` receive the fresh table.

---

## The Quant Engine: Graham + Lynch Factor Model

All formulas live in `internal/scoring/scoring.go` (pure functions, fully unit-tested) and `internal/service/ranking.go` (universe pipeline).

### Step 1 - Candidate Hygiene Filter

A stock must pass all of the following to be scored (`service/ranking.go:64-96`):

- `Price > 0` and `EPS` present
- `BVPS` present (Book Value Per Share)
- `PE > 0`
- Data freshness: `now - DataDate <= MAX_DATA_AGE_HOURS` (skippable)
- Optional growth floor: `EPSGrowth >= MIN_EPS_GROWTH`

### Step 2 - Core Formulas

| Formula | Code | Meaning |
|---|---|---|
| `GrahamValue = sqrt(22.5 * EPS * BVPS)` | `scoring.go:14` | Benjamin Graham's number: the "fair price" implied by earnings power x asset backing |
| `MoS = 1 - Price / GrahamValue` | `scoring.go:22` | Margin of Safety: discount to fair value |
| `EPSGrowth = EPS / PrevEPS - 1` | `scoring.go:30` | Year-over-year earnings growth |
| `RevenueGrowth = Revenue / PrevRevenue - 1` | `scoring.go:38` | Year-over-year top-line growth |
| `PEG = PE / (EPSGrowth * 100)` | `scoring.go:46` | Peter Lynch's growth-adjusted valuation |

### Step 3 - Percentile Normalization

Every factor is converted to a 0-100 cross-sectional percentile via `PercentileScore` (`scoring.go:58`), using average-rank tie handling. "Lower is better" factors (PE, PB, PEG) are inverted. This makes scores **relative**: a score of 80 means "cheaper/safer/faster-growing than 80% of today's eligible universe", not an absolute valuation.

### Step 4 - Composite Scores

```
GrahamScore = 0.40*MoS + 0.20*PE + 0.15*PB + 0.15*EPSGrowth + 0.10*ROE     (scoring.go:85)
LynchScore  = 0.50*PEG + 0.25*EPSGrowth + 0.15*RevenueGrowth + 0.10*ROE    (scoring.go:89)
GLScore     = 0.55*GrahamScore + 0.45*LynchScore                           (scoring.go:94)
```

The 55/45 blend is normative (a 50/50 variant was explicitly rejected): value leads, growth confirms.

### Step 5 - Bank Renormalization

Banks and financial issuers report DER/PB differently. When `PB`/`ROE` percentile is unavailable, the engine assigns a neutral 50 and renormalizes weights (`service/ranking.go:119`), so financials compete fairly without penalizing structural differences.

### Score Bands

| GL Score | Label |
|---|---|
| >= 90 | Exceptional |
| >= 80 | Strong |
| >= 70 | Attractive |
| >= 60 | Neutral |
| < 60 | Weak |

---

## Modules

### Gem Hunter (Top 10 Ranking)

- Endpoint `/` renders the live Top 10 dashboard; rows refresh via HTMX every 60s.
- Each row exposes: rank, ticker, price, GL/Graham/Lynch sub-scores, Margin of Safety, PEG, EPS growth, and status badge.
- Row click expands an AI-generated thesis: Summary, Strengths, Risks, Why-it-ranked, Confidence, Missing data.
- Fully deterministic: `service.Rank()` is a pure function (no `time.Now()`/HTTP inside the formula), making backtests and unit tests reproducible.

### Gem Guard (Anti-Manipulation Surveillance)

Anti-manipulation heuristics inspired by IDX surveillance mechanics (`internal/gemguard/gemguard.go`):

| Metric | Formula / Rule | Alert |
|---|---|---|
| **ARA Boundary** | >Rp 5,000: 20% - Rp 200-5,000: 25% - Rp 50-200: 35% | price spike vs exchange auto-rejection ceiling |
| **Velocity** | `AvgDailyVolume / FreeFloat` | capital-flow normalization |
| **PIR** (Price Impact Ratio) | `|PriceChange%| / Velocity` | **> 2.0 = alert**: big move on thin float |
| **Volume Spike** | `V_t / AvgVolume20` | **> 3.0x = alert**: abnormal participation |
| **Price Spike** | `(P_t - P_{t-1}) / P_{t-1}` | directional pump detection |
| **Insider flows** | holding % change + selling % | distribution signals |
| **RSI / Volatility / DER / Current Ratio** | technical + leverage context | composite risk context |

Outputs a 0-100 composite **Risk Score** with HIGH (>=70) / MEDIUM (40-69) / LOW (<40) bands, plus `PrimaryRiskFactors` bullet list per stock.

### Gem Sentinel (Financial Distress Radar)

Springate (1978) Multiple Discriminant Analysis (`internal/gemsentinel/gemsentinel.go`):

```
Score = 1.03*X1 + 3.07*X2 + 0.66*X3 + 0.40*X4

X1 = WorkingCapital / TotalAssets
X2 = EBIT / TotalAssets
X3 = ProfitBeforeTax / CurrentLiabilities
X4 = Revenue / TotalAssets
```

Classification cutoffs:

- `Score < 0.500` -> **CRITICAL** financial distress (severe solvency/default risk)
- `0.500 <= Score < 0.862` -> **MODERATE** distress
- `Score >= 0.862` -> **HEALTHY**

The page surfaces the **Top 5 most vulnerable** names with per-ratio breakdown and an AI surveillance synthesis (`AISentinelSummary`).

### AI Agent Layer

Multi-agent pipeline in `internal/ai`:

1. **Screener Agent** (`screener.go`) - triages candidates before deep analysis.
2. **Analyst Agent** (`analyst.go`) - builds the structured thesis (Summary / Strengths / Risks / Why-it-ranked / Confidence / Missing data). Works **deterministically without any LLM** (zero cost, 100% reproducible); upgrades to LLM-generated prose when a provider key is present.
3. **Risk Engine** (`risk.go`) - categorizes red flags: `valuation_risk`, `earnings_deterioration`, `debt_risk`, `price_momentum_deterioration`, `sector_concentration`, `data_anomaly` with low/medium/high severity.
4. **Tools** (`tools.go`) - safe data access for the agents.
5. **Guardrails** (`guardrails.go` + `prompts.go`) - hard rules the LLM must obey:
   - Never invent or modify `gl_score`, `graham_score`, `lynch_score`, or `rank`.
   - Use screening vocabulary only; forbidden advisory phrases ("pasti naik", "guaranteed return", ...).
   - State percentile semantics explicitly; mark N/A bank fields as `N/A (financial profile)`.
   - Prefix stale data with a staleness warning.
6. **Worker** (`worker.go`) - asynchronous background queue so ranking runs never block on AI.

Supported LLM providers (OpenAI-compatible and native): OpenAI, Anthropic, Google, Mistral, Groq, plus any OpenAI-compatible endpoint (Ollama, OpenRouter, vLLM, LM Studio).

### Email Alert System

Demo-friendly per-user alerting without authentication (`internal/notifier/notifier.go`):

- Public modal in the header (available on every page): enter email + choose alert channels.
- Channels: **Top 10 Gems** (Graham-Lynch digest), **Gem Guard** (high-risk alerts), **Gem Sentinel** (critical distress).
- Preferences persist in SQLite (`email_subscriptions`); every send is audited in `email_logs`.
- **Test Email** button verifies SMTP connectivity instantly (`POST /api/v1/email/test`).
- Digests dispatch automatically after each scheduled ranking run, personalized per subscription.
- SMTP engine supports **STARTTLS (587)** and **implicit TLS (465)** using only the Go standard library (`net/smtp`) - works with Brevo, Resend SMTP, Gmail, Mailgun, SES, Mailtrap, etc.
- Responsive dark-theme HTML digest with per-channel sections; subscribe/test flow fully in English.

### 5-Year Proof Page

`/proof` is the "show, don't tell" page: five real IDX stocks that the Graham-Lynch screen would have surfaced in early 2020, with verified Yahoo Finance price history (2 Jan 2020 vs 7 Sep 2026), price-only gains, and a plain-English walkthrough of why each passed the formula (MoS, PEG, percentile context). Includes honest "what this proves and what it does not" framing plus reproducibility instructions.

---

## Screens & Routes

| Route | Screen | Description |
|---|---|---|
| `/` | Gem Hunter Dashboard | Live Top 10 rankings, stats strip, HTMX auto-refresh, search |
| `/gemguard` | Gem Guard | Anti-manipulation surveillance table with expandable risk detail |
| `/gemsentinel` | Gem Sentinel | Springate distress watchlist with ratio breakdowns |
| `/methodology` | Methodology | Full human-friendly math explanation with KaTeX-rendered formulas |
| `/proof` | 5-Year Proof | Historical validation narrative (period, gain, method per stock) |

Shared UI shell (Go template partials): `head.html` (Inter font via Google Fonts, htmx, Alpine.js), `header.html` (brand, nav, context pills, email modal), `nav.html` (SVG icons), `modal_email.html` (alert subscription modal with animated spring-scale transitions).

---

## REST API Reference

Base URL: `http://localhost:3000`

### Health

| Method | Path | Description |
|---|---|---|
| GET | `/health` | Liveness probe |
| GET | `/ready` | Readiness probe |

### Ranking

| Method | Path | Description |
|---|---|---|
| GET | `/ranking` | Full latest `RunResult` JSON |
| GET | `/ranking/top10` | Latest top 10 - JSON, or HTML fragment when `HX-Request: true` |
| GET | `/api/v1/ranking/top10` | Latest top 10 JSON (versioned alias) |
| GET | `/stocks` | Ranked stocks array only |
| GET | `/stocks/:ticker` | Single ranked stock detail (404 if absent) |
| POST | `/admin/ranking/run` | Force a ranking run. Header: `Authorization: Bearer $ADMIN_TOKEN` |

### Surveillance

| Method | Path | Description |
|---|---|---|
| GET | `/gemguard` | Gem Guard HTML page |
| GET | `/api/v1/gemguard` | Gem Guard JSON array |
| GET | `/gemsentinel` | Gem Sentinel HTML page |
| GET | `/api/v1/gemsentinel` | Gem Sentinel JSON array (Top 5 distress) |

### Email Alerts

| Method | Path | Body | Description |
|---|---|---|---|
| POST | `/api/v1/subscribe` | `{"email":"a@b.c","notify_gems":true,"notify_guard":true,"notify_sentinel":true}` | Create/update subscription |
| POST | `/api/v1/email/test` | `{"email":"a@b.c"}` | Send a verification test email |

### AI Analysis

| Method | Path | Description |
|---|---|---|
| GET | `/api/v1/ai/analysis/:ticker` | Latest per-ticker AI analysis (404 if pending) |
| GET | `/api/v1/ai/run/:run_id` | All analyses for a ranking run |
| POST | `/admin/ai/run` | Re-enqueue latest run for AI processing (Bearer token) |

### Example

```bash
curl http://localhost:3000/api/v1/ranking/top10 | jq '.stocks[0]'

curl -X POST http://localhost:3000/api/v1/subscribe \
  -H "Content-Type: application/json" \
  -d '{"email":"investor@example.com","notify_gems":true,"notify_guard":true,"notify_sentinel":false}'

curl -X POST http://localhost:3000/admin/ranking/run \
  -H "Authorization: Bearer change-me"
```

---

## Tech Stack

| Layer | Choice | Why |
|---|---|---|
| Language | Go 1.25 | Single static binary, strong concurrency for scheduled runs + AI workers |
| Web framework | Fiber v2.52 | Fasthttp-based performance, minimal routing ergonomics |
| Database | SQLite via `modernc.org/sqlite` (pure Go, CGO-free) | Zero-ops persistence, WAL mode, cross-compile friendly |
| Templates | Go `html/template` with `ParseFS` | Compile-time embedded templates, XSS-safe auto-escaping |
| Interactivity | HTMX 1.9 + Alpine.js 3.13 | No build step, partial swaps, reactive modals |
| Fonts | Inter (Google Fonts) | Consistent modern typography |
| Math rendering | KaTeX 0.16 | Methodology formula display |
| Email | `net/smtp` (stdlib) | STARTTLS + implicit TLS, zero third-party deps |
| Caching | Redis 7 (Alpine) | Optional cache layer via `REDIS_URL` |
| Replication | Litestream 0.3 (profile) | Streaming SQLite backup for disaster recovery |
| Container | Distroless static image (nonroot) | Minimal attack surface, ~15MB runtime |

---

## Project Structure

```
gem-hunter/
|- cmd/
|  `- server/
|     |- main.go            # wiring: config, store, agents, mailer, scheduler, routes
|     |- views.go           # template FuncMap, view models, page renderers
|     `- migrations.sql     # embedded SQLite schema (also mirrored in /migrations)
|- internal/
|  |- ai/                   # AI agent layer
|  |  |- agent.go           #   orchestrator (screener -> analyst -> risk)
|  |  |- screener.go        #   triage agent
|  |  |- analyst.go         #   thesis builder (deterministic + LLM)
|  |  |- risk.go            #   risk flag engine
|  |  |- tools.go           #   data access for agents
|  |  |- guardrails.go      #   advisory-language firewall
|  |  |- prompts.go         #   system prompts (strict no-hallucination rules)
|  |  |- types.go           #   RiskFlag, ScoreBreakdown, Analysis, ...
|  |  |- worker.go          #   async background worker
|  |  `- ai_test.go
|  |- config/config.go      # env loader (all runtime knobs)
|  |- domain/domain.go      # canonical Snapshot + Sentinel types
|  |- gemguard/             # anti-manipulation math (ARA, PIR, velocity, spikes)
|  |  |- gemguard.go
|  |  `- gemguard_test.go
|  |- gemsentinel/          # Springate MDA distress model
|  |  |- gemsentinel.go
|  |  `- gemsentinel_test.go
|  |- notifier/notifier.go  # SMTP mailer, digest builder, test email, scheduler hook
|  |- provider/sectors/     # Sectors.app V2 client (retry, 429 backoff, report mapping)
|  |- repository/store.go   # SQLite store (runs, results, subscriptions, email logs)
|  |- scoring/              # pure formula library (Graham/Lynch/percentiles)
|  |  |- scoring.go
|  |  `- scoring_test.go
|  `- service/              # ranking pipeline + mock universe
|     |- ranking.go         #   Rank(): pure, dedup, eligibility, scoring, Top N
|     |- ranking_test.go
|     `- mock.go            # 12-issuer synthetic universe for offline dev
|- migrations/001_init.sql  # standalone schema reference
|- web/
|  |- web.go                # //go:embed for templates + static assets
|  |- templates/            # dashboard, gemguard, gemsentinel, methodology, proof,
|  |                        # head/header/nav/modal_email/rows partials
|  `- static/css/app.css    # Stone/Spectral dark theme (sectors.app palette)
|- docker-compose.yml       # app + redis (+ optional litestream profile)
|- Dockerfile               # multi-stage: golang:1.25 -> distroless/static
|- litestream.yml           # replication config
|- docs/                    # all.md, implementation-plan.md
`- task3.md
```

---

## Configuration (Environment Variables)

All configuration is environment-driven (`internal/config/config.go`), typically via `.env` (see `.env.example`).

### Core

| Variable | Default | Description |
|---|---|---|
| `APP_ENV` | `development` | Environment label |
| `PORT` | `3000` | HTTP listen port |
| `SQLITE_PATH` | `data/gemhunter.db` | SQLite file path (WAL) |
| `LITESTREAM_ENABLED` | `false` | Litestream replication flag |
| `REDIS_URL` | (empty) | Optional Redis cache endpoint |
| `TZ` | - | Recommended: `Asia/Jakarta` for scheduler alignment |

### Market Data & Quant Engine

| Variable | Default | Description |
|---|---|---|
| `SECTORS_API_KEY` | (required) | Sectors.app V2 API key. Without it the server logs a warning and boots with the mock universe |
| `SECTORS_BASE_URL` | `https://api.sectors.app` | Sectors API base URL |
| `RANKING_INTERVAL_HOURS` | `6` | Ranking cadence (scheduler also snaps to 00/06/12/18 WIB) |
| `MIN_EPS_GROWTH` | `0` | Eligibility floor for EPS growth (fraction: `0.1` = 10%) |
| `MAX_DATA_AGE_HOURS` | `168` | Max snapshot age before a stock is skipped |
| `MIN_MARKET_CAP` | (empty) | Optional liquidity floor |
| `MIN_AVG_VALUE_20D` | (empty) | Optional 20-day traded-value floor |
| `ADMIN_TOKEN` | `change-me` | Bearer token for `/admin/*` endpoints |

### AI Layer

| Variable | Description |
|---|---|
| `OPENAI_API_KEY` / `ANTHROPIC_API_KEY` / `GOOGLE_API_KEY` / `MISTRAL_API_KEY` / `GROQ_API_KEY` | Native provider keys (any one is enough) |
| `OPENAI_COMPATIBLE_API_KEY` / `OPENAI_COMPATIBLE_BASE_URL` | Any OpenAI-compatible endpoint (Ollama `http://host.docker.internal:11434/v1`, OpenRouter, vLLM, LM Studio) |
| `LLM_DEFAULT_PROVIDER` | `openai` (default provider id) |
| `LLM_DEFAULT_MODEL` | `gpt-4o-mini` |

> With no keys at all, the AI layer falls back to deterministic template analysis - the app remains fully functional and free to run.

### Email Alerts (SMTP)

| Variable | Default | Description |
|---|---|---|
| `SMTP_HOST` | (empty) | e.g. `smtp-relay.brevo.com` (Brevo), `smtp.resend.com` (Resend), `smtp.gmail.com` |
| `SMTP_PORT` | `587` | `587` = STARTTLS, `465` = implicit TLS |
| `SMTP_USER` | (empty) | SMTP login |
| `SMTP_PASSWORD` | (empty) | SMTP key/password |
| `SMTP_FROM` | `Gem Hunter <alerts@gemhunter.app>` | Verified sender header |
| `EMAIL_ALERTS_ENABLED` | `false` | Master switch for digest dispatch |

Example (Brevo free tier, 300 emails/day):

```env
SMTP_HOST=smtp-relay.brevo.com
SMTP_PORT=587
SMTP_USER=your-login@example.com
SMTP_PASSWORD=your-smtp-key
SMTP_FROM=Gem Hunter <alerts@yourdomain.com>
EMAIL_ALERTS_ENABLED=true
```

---

## Running Locally

### Prerequisites

- Docker + Docker Compose (recommended), **or**
- Go 1.25+ (native)

### Option A - Docker (recommended)

```bash
cp .env.example .env          # fill SECTORS_API_KEY + SMTP (optional)
docker compose up -d --build
```

App: http://localhost:3000 - Redis runs automatically; SQLite persists in the `appdata` volume.

> Template/static asset changes are embedded at build time (`web/web.go` `//go:embed`): re-run `docker compose up -d --build` after editing UI files.

### Option B - Native Go

```bash
export SECTORS_API_KEY=your_key       # optional; mock universe without it
export SQLITE_PATH=data/gemhunter.db
go run ./cmd/server
```

### Litestream replication (optional profile)

```bash
docker compose --profile replication up -d
```

---

## Docker Deployment

The multi-stage `Dockerfile`:

1. **Builder**: `golang:1.25-bookworm` -> `CGO_ENABLED=0 go build -o /server ./cmd/server` (pure-Go SQLite, no libc dependency).
2. **Runtime**: `gcr.io/distroless/static-debian12:nonroot` - only the static binary + embedded assets, runs as nonroot, `VOLUME /data`, `EXPOSE 3000`.

Services (`docker-compose.yml`):

- `app` - the Gem Hunter server (`env_file: .env`, volume `appdata:/data`)
- `redis` - with AOF persistence and healthcheck
- `litestream` (profile `replication`) - streams WAL backups per `litestream.yml`

Useful commands:

```bash
docker compose ps                 # status
docker compose logs --tail=50 app # application logs (JSON-structured events)
docker compose up -d --build      # rebuild after code/UI changes
docker compose down               # stop (volumes preserved)
```

---

## Testing

```bash
go test ./...
```

Test coverage by package:

- `internal/scoring` - Graham/Lynch formula invariants, percentile ties, zero-division guards
- `internal/service` - ranking pipeline: dedup, eligibility filters, bank renormalization, Top N ordering
- `internal/gemguard` - ARA boundaries, PIR, velocity, spike math
- `internal/gemsentinel` - Springate ratios, cutoff classification
- `internal/ai` - guardrails and analysis structure

Build check without a local Go toolchain:

```bash
docker run --rm -v "${PWD}:/src" -w /src golang:1.25-alpine go build ./...
```

---

## Database Schema

Embedded via `//go:embed migrations.sql` and applied on boot (`CREATE TABLE IF NOT EXISTS`).

| Table | Purpose |
|---|---|
| `companies` | Universe registry (ticker, name, sector, industry) |
| `financial_snapshots` | Historical per-ticker fundamentals (price, eps, bvps, pe, pb, roe + timestamps) |
| `daily_prices` | Daily OHLCV cache for backtesting (`PRIMARY KEY(ticker,date)`) |
| `ranking_runs` | One row per scheduled/manual ranking run (status, counts) |
| `ranking_results` | Per-stock scores for each run (all factor percentiles + raw values) |
| `email_subscriptions` | Per-email channel preferences (`notify_gems`, `notify_guard`, `notify_sentinel`) |
| `email_logs` | Audit trail: type, subject, status, error, sent_at |

SQLite is opened in WAL mode with `busy_timeout=5000`, single-connection pooling, and an in-memory cache of the latest run for hot reads.

---

## Methodology & Scientific Basis

The full step-by-step explanation (with KaTeX-rendered math) lives at `/methodology` and covers:

1. **Why combine Graham & Lynch** - avoiding both the value trap and the growth-at-any-price trap.
2. **Graham Value & Margin of Safety** - `sqrt(22.5*EPS*BVPS)` intuition and downside protection.
3. **PEG** - Lynch's growth-at-a-reasonable-price discipline.
4. **Percentile normalization** - why scores are cross-sectional ranks, not absolute valuations.
5. **Composite weighting** - the normative 55/45 Graham/Lynch blend and factor-by-factor rationale.
6. **Bank renormalization** - fair treatment of financial issuers.
7. **Gem Guard framework** - ARA boundaries, PIR, velocity, volume spikes (IDX anti-manipulation mechanics).
8. **Gem Sentinel / Springate MDA** - the 1.03/3.07/0.66/0.40 discriminant with 0.862 cutoff, validated for IDX issuers.

The `/proof` page complements this with five tickers of historical evidence (BRIS, ANTM, HRUM, ESSA, ITMG) from public Yahoo Finance data, each explained through the exact formulas above.

---

## Roadmap

- [x] Core Graham-Lynch ranking engine + scheduler
- [x] Gem Guard anti-manipulation surveillance
- [x] Gem Sentinel Springate distress radar
- [x] AI agent layer with guardrails + deterministic fallback
- [x] Per-user email alert subscriptions + test email
- [x] 5-Year Proof page (Yahoo Finance historical validation)
- [ ] Full Sectors live-universe wiring (replace mock when API key active)
- [ ] Quarterly-rebalance backtest engine vs IHSG benchmark (CAGR, Sharpe, hit rate, max drawdown)
- [ ] Signal Confluence Score fusing all three engines
- [ ] Intraday anomaly alerts (volume/price spikes during market hours)
- [ ] Telegram/Discord webhook delivery
- [ ] Point-in-time fundamentals cache for bias-free backtests

---

## License & Attribution

Built for the Sectors Hackathon 2026.

- **Market data**: [Sectors.app Financial API V2](https://sectors.app) (live screening) + Yahoo Finance public chart API (historical proof).
- **Fonts**: Inter by Rasmus Andersson (Google Fonts).
- **Formulas**: Benjamin Graham's number, Peter Lynch's PEG, Springate (1978) MDA, IDX Auto-Rejection mechanics.
- All screening output is research information, not investment advice.
