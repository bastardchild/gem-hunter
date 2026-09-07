# Graph Report - gem-hunter  (2026-09-07)

## Corpus Check
- Corpus is ~13,311 words - fits in a single context window. You may not need a graph.

## Summary
- 181 nodes · 326 edges · 15 communities (12 shown, 3 thin omitted)
- Extraction: 91% EXTRACTED · 9% INFERRED · 0% AMBIGUOUS · INFERRED: 30 edges (avg confidence: 0.83)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- LLM Agent & AI Analysis Pipeline
- Server Setup & Configuration
- Quantitative Scoring Engine
- Async Worker & Database Store
- Risk Assessment & Stock Screening
- Web Dashboard & Methodology Views
- Gem Guard Surveillance & PIR
- Market Data Provider & HTTP Client
- Docker & Litestream Infrastructure
- Sectors V2 API & Rate Limiting
- Architecture & MVP Strategy
- Backtest Discipline
- Core GemHunter Module

## God Nodes (most connected - your core abstractions)
1. `main()` - 15 edges
2. `Agent` - 15 edges
3. `RunResult` - 15 edges
4. `Rank()` - 15 edges
5. `Store` - 14 edges
6. `NewAgent()` - 8 edges
7. `Analysis` - 8 edges
8. `toolsImpl` - 7 edges
9. `CandidateInput` - 7 edges
10. `Worker` - 7 edges

## Surprising Connections (you probably didn't know these)
- `Normative Valuation Formulas & Vector ABC` --semantically_similar_to--> `GL Score Composition & ABC Normalization`  [INFERRED] [semantically similar]
  docs/implementation-plan.md → web/templates/methodology.html
- `main()` --calls--> `NewAgent()`  [EXTRACTED]
  cmd/server/main.go → internal/ai/agent.go
- `main()` --calls--> `NewWorker()`  [EXTRACTED]
  cmd/server/main.go → internal/ai/worker.go
- `main()` --calls--> `New()`  [EXTRACTED]
  cmd/server/main.go → internal/repository/store.go
- `main()` --calls--> `MockUniverse()`  [EXTRACTED]
  cmd/server/main.go → internal/service/mock.go

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **Graham & Lynch Factor Valuation Pipeline** — web_templates_methodology_graham_intrinsic_value, web_templates_methodology_lynch_peg_ratio, web_templates_methodology_percentile_ranking, web_templates_methodology_gl_composite_score [EXTRACTED 1.00]
- **Gem Guard Anti-Manipulation Surveillance Pipeline** — web_templates_methodology_price_impact_ratio, web_templates_methodology_uma_trigger_criteria, web_templates_methodology_ml_surveillance_architecture, web_templates_gemguard_html [EXTRACTED 1.00]
- **Docker SQLite Litestream Persistence Topology** — docker_compose_service_app, docker_compose_service_litestream, docker_compose_volume_appdata, litestream_replication_config [EXTRACTED 1.00]

## Communities (15 total, 3 thin omitted)

### Community 0 - "LLM Agent & AI Analysis Pipeline"
Cohesion: 0.11
Nodes (19): Agent, Analysis, Analyst, analystImpl, RiskEngine, riskEngineImpl, RunAnalyses, Screener (+11 more)

### Community 1 - "Server Setup & Configuration"
Cohesion: 0.13
Nodes (16): defaultGemGuardSurveillance(), fallbackConfig(), getenvDefault(), main(), mustLatest(), dashboardData(), nextScheduleSlot(), renderDashboard() (+8 more)

### Community 2 - "Quantitative Scoring Engine"
Cohesion: 0.17
Nodes (20): testing.T, EPSGrowth(), f(), GLScore(), GrahamScore(), GrahamValue(), LynchScore(), MarginOfSafety() (+12 more)

### Community 3 - "Async Worker & Database Store"
Cohesion: 0.15
Nodes (9): Worker, database/sql.DB, database/sql.NullFloat64, sync.RWMutex, NewWorker(), f64(), Store, New() (+1 more)

### Community 4 - "Risk Assessment & Stock Screening"
Cohesion: 0.22
Nodes (10): CandidateInput, RiskCategory, RiskFlag, RiskSeverity, ScoreBreakdown, screenerImpl, ScreenerResult, toolsImpl (+2 more)

### Community 5 - "Web Dashboard & Methodology Views"
Cohesion: 0.12
Nodes (18): GL Composite Scoring Specification, Normative Valuation Formulas & Vector ABC, Top 10 IDX Dashboard View, Gem Guard Surveillance View, Asness, Frazzini & Pedersen (2019) Quality Minus Junk, Fama & French (1992) Cross-Section of Stock Returns, Fargher, Weigand & Belghitar (2011) Efficacy of PEG Ratio, GL Score Composition & ABC Normalization (+10 more)

### Community 6 - "Gem Guard Surveillance & PIR"
Cohesion: 0.18
Nodes (13): ARABoundary(), CalculatePIR(), CalculateVelocity(), ComputeRiskScore(), PriceSpikePct(), RSI(), TestARABoundary(), TestCalculateVelocityAndPIR() (+5 more)

### Community 7 - "Market Data Provider & HTTP Client"
Cohesion: 0.19
Nodes (11): MarketDataProvider, net/http.Client, net/http.Header, net/url.Values, Snapshot, fptr(), MapReport(), New() (+3 more)

### Community 8 - "Docker & Litestream Infrastructure"
Cohesion: 0.47
Nodes (6): Docker Service App, Docker Service Litestream, Docker Service Redis, Docker Volume Appdata, Gem Hunter Deployment & Litestream Strategy, Litestream SQLite Replication Config

### Community 9 - "Sectors V2 API & Rate Limiting"
Cohesion: 0.67
Nodes (3): Sectors.app V2 API Integration, Data & Rate Limit Risk Mitigations, Sectors API V2 Endpoint Catalog & Credit Optimization

## Knowledge Gaps
- **16 isolated node(s):** `gemhunter`, `MarketDataProvider`, `screenerRow`, `Docker Service Redis`, `Pragmatic Hexagonal Architecture` (+11 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **3 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Rank()` connect `Quantitative Scoring Engine` to `LLM Agent & AI Analysis Pipeline`, `Server Setup & Configuration`, `Async Worker & Database Store`, `Market Data Provider & HTTP Client`?**
  _High betweenness centrality (0.181) - this node is a cross-community bridge._
- **Why does `main()` connect `Server Setup & Configuration` to `LLM Agent & AI Analysis Pipeline`, `Quantitative Scoring Engine`, `Async Worker & Database Store`?**
  _High betweenness centrality (0.150) - this node is a cross-community bridge._
- **Why does `RunResult` connect `Async Worker & Database Store` to `LLM Agent & AI Analysis Pipeline`, `Server Setup & Configuration`, `Quantitative Scoring Engine`, `Risk Assessment & Stock Screening`?**
  _High betweenness centrality (0.101) - this node is a cross-community bridge._
- **Are the 5 inferred relationships involving `main()` (e.g. with `nextScheduleSlot()` and `renderDashboard()`) actually correct?**
  _`main()` has 5 INFERRED edges - model-reasoned connections that need verification._
- **Are the 3 inferred relationships involving `Rank()` (e.g. with `f()` and `TestRankEmpty()`) actually correct?**
  _`Rank()` has 3 INFERRED edges - model-reasoned connections that need verification._
- **What connects `gemhunter`, `MarketDataProvider`, `screenerRow` to the rest of the system?**
  _16 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `LLM Agent & AI Analysis Pipeline` be split into smaller, more focused modules?**
  _Cohesion score 0.11494252873563218 - nodes in this community are weakly interconnected._