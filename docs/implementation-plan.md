# IMPLEMENTATION PLAN — Gem Hunter MVP (hasil PHASE 1)

## 1. Hasil inspeksi
- Repo: kosong kecuali `.memory/` (task, harness, skill, formula-logic). Belum ada `go.mod`, Dockerfile, compose, SQLite/Redis lokal.
- Host: Windows, **Go tidak terinstal** (`go: not recognized`), Docker 29.7.2 + Compose v5.5.0 tersedia. Strategi: **semua perintah Go via `docker run --rm golang:1.25-bookworm`**, runtime app via Compose. Pin `golang:1.25-bookworm`, `redis:7-alpine`. DB: SQLite file (WAL, pure-Go), tanpa service DB terpisah.
- SQLite/Redis: belum ada container/file. Dibuat di PHASE 2 via compose (`redis`, `app` + healthcheck; SQLite di volume `appdata`, Litestream nonaktif di dev).
- Sectors V2: v1 discontinued 2026-05-11 (410 Gone). Auth: header `Authorization: <API_KEY>` dari `SECTORS_API_KEY`. Billing: 2xx/404 bayar kredit, 400/401/403/429/5xx gratis (kecuali `?q=` 400 pasca-LLM = 1 kredit).

## 2. Endpoint V2 yang dipakai (terverifikasi dari docs resmi)
| Kebutuhan | Endpoint | Catatan kredit |
|---|---|---|
| Universe + snapshot massal | `GET /v2/companies/` (structured `where`/`order_by`, paginasi) | 1 kredit/query; JANGAN pakai `?q=` (3 kredit) |
| Detail per ticker (overview/valuation/financials/dividend) | `GET /v2/company/report/{symbol}/?sections=overview,valuation,financials,dividend` | 1 kredit per section → 4 kredit/ticker |
| EPS/revenue historis & YoY | `financials.historical_eps`, `historical_financials[]` (revenue/earnings per year) dari report; fallback `GET /v2/financials/quarterly/{symbol}/?n_quarters=8` | 1 kredit per quarter |
| Harga + marketCap harian | `GET /v2/daily/{symbol}/?start=&end=` (max 90 hari/request) | 1 kredit |
| Suspensi/delisted | suspensions endpoint (filter `symbol`) | — |
| BVPS | **tidak ada field langsung** → hitung `total_equity / outstanding_shares` (dari `historical_financials` + `outstanding_shares[YYYY]`) | — |

## 3. Mapping eksak Sectors → snapshot (final, bukan tebakan)
| Snapshot | Sumber V2 |
|---|---|
| ticker/companyName/sector/industry | `symbol`, `company_name`, `overview.sector/sub_sector/industry` |
| price/marketCap | `overview.last_close_price` / `market_cap` (fallback `/v2/daily/` latest `close`/`market_cap`) |
| eps / prevEps | `financials.eps` (atau `historical_eps[YYYY]`), prev = tahun N-1 |
| bvps | `total_equity[Y] / outstanding_shares[Y]` (dihitung, bukan raw) |
| pe / pb | `pe_ttm` (fallback `pe[Y]`), `pb_mrq` (fallback `pb[Y]`) |
| roe / der | `roe_ttm` (fallback `roe[Y]`); `der_mrq` (fallback `debt_to_equity_ratio[Y]`; bank → N/A) |
| revenue / prevRevenue | `historical_financials[].revenue` Y vs Y-1 |
| dividendYield | `yield_ttm` |
| dataDate/publishedAt | `latest_close_date` / tahun fiskal report; `fetchedAt` = waktu fetch |

## 4. Formula (kunci dari formula-logic.md, normatif)
`GrahamValue=sqrt(22.5·EPS·BVPS)` (null bila EPS/BVPS≤0); `MOS=1−Price/GrahamValue`;
`EPSGrowth=EPS_t/EPS_prev−1`, `RevenueGrowth` analog (null bila denom≤0);
`PEG=PE/(EPSGrowth·100)` (null bila PE≤0/growth≤0 — jangan pakai 0.20);
percentile percent-rank + average-ties (`n==1→50`), lower-is-better di-invers;
`Graham=0.40MOS+0.20PE+0.15PB+0.15EPSg+0.10ROE`;
`Lynch=0.50PEG+0.25EPSg+0.15RevG+0.10ROE`;
`GL=0.55Graham+0.45Lynch`. Vector ABC: Graham 83.35, Lynch 86.55, GL 84.79.

## 5. Struktur direktori (dibuat PHASE 2)
`cmd/server/main.go`, `internal/{domain,config,scoring,provider/sectors,repository,service,handler,middleware,scheduler,cache}`,
`web/{templates,static}`, `migrations/`, `tests/`, `docker/`, `docs/`, `Dockerfile`, `docker-compose.yml`, `.env.example`.

## 6. Ambiguitas / risiko
1. BVPS tidak native → hitung sendiri; bila `outstanding_shares` null → GrahamValue null (exclude). 2. Screener `where` tanpa tahun = data "most recent" (Jan–Apr bisa tahun N-2, smart-FY) → simpan `periodEnd/publishedAt`, hormati `available_at<=ranking_time`. 3. Kredit mahal bila report full 8 section → batasi 4 section + cache Redis agresif + refresh 6 jam ber-jitter. 4. `GET /v2/companies/` paginasi ratusan emiten → butuh loop halaman + rate-limit/backoff + hormati Retry-After. 5. Bank: field loan/deposit ≠ DER → profil finansial terpisah.
