# Architecture — Gem Hunter
Clean/Hexagonal pragmatis. `cmd/server` (Fiber) → `internal/service` (ranking) → `internal/scoring` (formula-logic.md) + `internal/provider/sectors` (DTO→snapshot) + `internal/repository` (store). Handler tanpa business logic.
# Scoring — ringkas
Lihat `.memory/formula-logic.md` normatif + `docs/implementation-plan.md`. GL=0.55 Graham+0.45 Lynch. Vector ABC: 83.35/86.55/84.79.
# Data Source — Sectors V2
`GET /v2/companies/` universe, `GET /v2/company/report/{sym}/?sections=overview,valuation,financials,dividend`, `GET /v2/daily/{sym}/`, `GET /v2/financials/quarterly/{sym}/`. BVPS dihitung `total_equity/outstanding_shares`. Auth `Authorization: SECTORS_API_KEY`.
# Deployment
`docker compose up --build` (app + redis; SQLite di volume `appdata`, WAL mode).
Litestream NONAKTIF di dev (`LITESTREAM_ENABLED=false`); replikasi hanya via
`docker compose --profile replication up -d` (lihat `litestream.yml`).
Env via `.env` (lihat `.env.example`). Health `/health`, ready `/ready`.
# Backtesting
`service.Rank` pure atas snapshot; syarat `available_at<=ranking_time`; biaya fee/pajak/slippage wajib dihitung di fase backtest.
