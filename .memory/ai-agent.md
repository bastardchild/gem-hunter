# AI-AGENT — Analysis Layer di atas Quant Engine (Referensi Implementasi)

> Prinsip non-negosiasi: **AI tidak pernah menghitung/mengubah `GLScore`**.
> Quant engine (`internal/scoring` + `internal/service`, lihat `.memory/formula-logic.md`) tetap
> deterministik, reproducible, dan bisa di-backtest. AI adalah **reasoning / explanation / risk layer**.

## 1. Arsitektur

```text
Sectors.app v2
      ↓
Data Ingestion (internal/provider/sectors)
      ↓
SQLite (WAL) ───── Redis
      ↓
Quant Engine (Graham + Lynch + future factors)
      ↓
Top 10 Candidates (service.RunResult)
      ↓
AI Analyst Layer (Screener → Analyst → Risk)
      ↓
FINAL REPORT → HTMX Dashboard
```

```
                 GEM HUNTER
                      │
          ┌───────────┴───────────┐
          │                       │
    QUANT ENGINE              AI AGENTS
          │                       │
 Graham + Lynch          ┌────────┼────────┐
 Multi Factor            │        │        │
 Risk Model           Screener Analyst   Risk
          │               │        │        │
          └───────────────┴────────┴────────┘
                          │
                     FINAL REPORT
                          │
                    Top 10 BEI
```

## 2. Kontrak input (dari quant → AI)

AI menerima `service.RankedStock` apa adanya, contoh:

```json
{
  "ticker": "BBCA",
  "rank": 1,
  "gl_score": 91.4,
  "graham_score": 94.1,
  "lynch_score": 88.2,
  "mos": 0.27,
  "pe": 18.2,
  "pb": 3.1,
  "eps_growth": 0.18,
  "revenue_growth": 0.14,
  "roe": 0.23
}
```

Plus konteks run: `run_id`, `calculated_at`, `data_as_of`, `score_breakdown`
(`S(MOS)`, `S(PE)`, `S(PB)`, `S(PEG)`, `S(EPSGrowth)`, `S(RevenueGrowth)`, `S(ROE)` +
flag renormalisasi bank), `previous_rank` / `score_delta_6h`, dan `data_quality`.

## 3. Tiga agent

### 3.1 Screener Agent — "layak dianalisis lebih lanjut?"
- Input: Top 10 + universe stats.
- Tugas: triase. Tandai kandidat yang skornya didorong **satu faktor saja**,
  data tipis (komponen N/A / renormalisasi), atau stagnan (tidak ada perubahan
  2 run terakhir). Tidak mengeliminasi dari ranking — hanya memberi label
  `deep_dive: true/false` + alasan.
- Output: `[{ticker, deep_dive, reason}]`.

### 3.2 Analyst Agent — "why this stock?"
- Input per ticker: snapshot + breakdown + histori + perbandingan sektor.
- Tugas, urutan tetap:
  1. Apakah valuation menarik? (MOS + S(PE)/S(PB) relatif universe.)
  2. Apakah growth mendukung valuation? (EPSGrowth vs PEG.)
  3. Apakah score seimbang atau satu faktor? (lihat breakdown.)
  4. Apa yang berubah vs ranking 6 jam sebelumnya?
  5. Bull case / bear case / data yang kurang.
- Contoh output (format baku, dilarang klaim profit):

```text
BBCA — Rank #1

AI Summary
Strong quality-growth profile with attractive Graham margin
of safety relative to the current universe.

Strengths
✓ EPS growth 18%
✓ ROE 23%
✓ Positive Graham MOS
✓ Strong Lynch score

Risks
⚠ PE remains relatively high
⚠ Valuation depends on continued earnings growth

Why it ranked #1
The stock scores strongly across both value and growth
dimensions rather than relying on a single factor.

AI Confidence: 84/100
```

### 3.3 Risk Agent — red flags
- Kategori tetap: `valuation_risk`, `earnings_deterioration`, `debt_risk`,
  `price_momentum_deterioration`, `sector_concentration`, `data_anomaly`.
- Setiap flag: `{category, severity: low|medium|high, evidence, metric}`.
- Bank (`isFinancial`): DER tidak boleh jadi flag (sudah N/A di quant).

## 4. Tools yang boleh dipanggil agent ( investigation, bukan prompt→jawaban )

```text
AI Analyst
├── get_stock_snapshot(ticker, run_id)
├── get_historical_financials(ticker, n_quarters)
├── get_price_history(ticker, days)
├── get_previous_ranking(run_id, offset)
├── get_sector_comparison(ticker)
├── get_score_breakdown(ticker, run_id)
├── detect_anomalies(ticker)
└── generate_analysis(ticker, run_id)
```

Alur investigasi baku:

```text
Top 10 → pilih kandidat (deep_dive) → ambil historical →
bandingkan sektor → cek perubahan score → cek anomaly →
buat investment thesis
```

Aturan tool:
- Semua tool **read-only** terhadap ranking; tidak ada `set_score` / `rerank`.
- `get_*` baca dari SQLite (persisten) atau Redis (cache); tidak hit
  Sectors langsung dari agent (hemat kredit, hindari inkonsistensi dengan quant run).
- `detect_anomalies` deterministik (aturan statistik, bukan LLM): lompatan
  EPSGrowth > 3σ sektor, MOS tanda berbalik, komponen hilang antar run.

## 5. Prompt sistem (template, ditempel di code sebagai konstanta)

```
You are the Gem Hunter Analyst Agent, a quantitative equity analysis layer
for IDX stocks. You receive deterministic scores from a Graham+Lynch engine.
Rules:
- NEVER invent or modify gl_score, graham_score, lynch_score, or rank.
  Quote them exactly as given.
- Scores are cross-sectional percentiles (relative to the eligible universe),
  not absolute valuations. Say so when relevant.
- Use only "ranking", "signal", "screening", "score". NEVER "pasti naik",
  "pasti untung", "jaminan profit", "buy recommendation".
- Always include: Summary, Strengths, Risks, Why-it-ranked, Confidence (0-100),
  Missing data.
- If a breakdown component is N/A (e.g. bank DER), say "N/A (financial profile)"
  instead of treating it as weakness.
- Cite every factual claim with [ticker, metric, value, run_id].
- If data is stale (flag set), prefix output with "⚠ Data may be stale".
```

## 6. Guardrails

1. **Otoritas**: AI dilarang output berisi skor/rank baru. Validator menolak
   respons yang mengandung pola `gl_score\s*[:=]\s*\d` selain mengutip input.
2. **Bahasa**: filter kata terlarang (`pasti naik|pasti untung|jaminan profit|rekomendasi beli`)
   → tolak + log.
3. **Grounding**: setiap angka di output harus cocok dengan input/tool output;
   angka tanpa sitasi → tolak.
4. **Stale**: bila `stale=true` atau umur data > `MAX_DATA_AGE_HOURS`, analisis
   wajib mencantumkan peringatan, dilarang menyajikan sebagai real-time.
5. **Biaya**: agent tidak boleh memicu fetch Sectors per analisis; hanya baca
   SQLite/Redis. Budget LLM per run dibatasi (mis. Top 10 → max 10 analisis).
6. **Audit**: simpan prompt hash, model, input `run_id`, output, confidence.

## 7. Redis job flow (koordinasi dengan scheduler 6 jam)

Pipeline quant yang ada (`FETCH→…→PUBLISH`, lock `gemhunter:ranking:lock`)
diperpanjang — agent jalan **setelah** `PUBLISH`, tidak memblokir ranking:

```text
quant run SUCCESS (run_id=R)
  → LPUSH gemhunter:ai:queue {"run_id": R, "tickers": [...Top10]}
  → AI worker (1 consumer, grup `ai-analyst`) BRPOPLUSH queue→processing
  → SET gemhunter:ai:{run_id}:{ticker} <analysis JSON> EX 6h
  → SET gemhunter:ai:{run_id}:status {pending|partial|done|failed}
  → dashboard baca cache analisis; bila kosong → tampilkan quant saja + badge "AI pending"
```

Keys:

```text
gemhunter:ai:queue                 # list antrean run
gemhunter:ai:processing            # list backup BRPOPLUSH
gemhunter:ai:{run_id}:status       # string
gemhunter:ai:{run_id}:{ticker}     # JSON analisis, TTL 6 jam
gemhunter:ai:latest:{ticker}       # pointer ke analisis terbaru (untuk /stocks/:ticker)
```

Lock terpisah `gemhunter:ai:lock` (TTL) bila multi-replica worker.
Kegagalan agent tidak boleh mengubah status quant run.

## 8. Struktur folder yang disarankan

```text
internal/ai/
├── agent.go        # interface Analyst { Analyze(ctx, CandidateInput) (Analysis, error) }
├── screener.go     # Screener Agent (triase deep_dive)
├── analyst.go      # Analyst Agent (why-stock)
├── risk.go         # Risk Agent (red flags)
├── tools.go        # implementasi 8 tools (baca repository/store, bukan Sectors)
├── prompts.go      # konstanta system prompt §5 + template output
├── guardrails.go   # validator skor/bahasa/sitasi/stale
├── worker.go       # consumer gemhunter:ai:queue + penulis cache Redis
└── types.go        # CandidateInput, Analysis, RiskFlag, ScoreBreakdown
web/templates/
├── ranking.html    # tambah kolom/section AI summary + confidence + risks
└── stock.html      # tambah Bull/Bear, Why-ranked, Missing data
```

`CandidateInput` minimal: `RankedStock + ScoreBreakdown + PreviousDelta + SectorStats + DataQuality + Stale`.
`Analysis` minimal: `{ticker, run_id, model, summary, strengths[], risks[], why_ranked,
bull_case, bear_case, confidence, missing_data[], risk_flags[], created_at}`.

## 9. API & UI (tambahan, quant endpoint tidak berubah)

- `GET /api/v1/ai/analysis/:ticker` → analisis terbaru (dari `gemhunter:ai:latest:{ticker}`).
- `GET /api/v1/ai/run/:run_id` → status + daftar analisis run itu.
- `POST /admin/ai/run` (auth sama dengan `/admin/ranking/run`) → enqueue ulang analisis
  untuk latest run. Rate-limit ketat (mis. 1x/5 menit).
- Dashboard: badge `AI confidence 84`, section Risks, link "why ranked".
  HTMX poll terpisah (`hx-get="/api/v1/ai/run/{run_id}" hx-trigger="every 120s"`)
  agar analisis yang lambat tidak menghambat tabel Top 10.

## 10. Testing AI layer (tanpa LLM asli di CI)

- Golden test: input `RankedStock` tetap → mock LLM → snapshot output
  (format section lengkap, angka dikutip persis, kata terlarang absen).
- Guardrail test: injeksikan respons berisi `gl_score: 99` baru / "pasti untung" /
  angka tak bersitasi → validator menolak.
- Tool test: `detect_anomalies` dengan fixture (lonjakan EPS 3σ, MOS flip,
  komponen hilang) → flag benar.
- Contract test: analisis merujuk `run_id` yang salah / ticker tak ada di run → error.
- E2E (opsional, tag `integration`): enqueue → worker mock → Redis cache terisi →
  endpoint serve JSON.

## 11. Rollout

1. Fase A (tanpa LLM): tampilkan `Why it ranked` deterministik dari breakdown
   (template, bukan LLM) + Risk flags aturan statistik. Nol biaya, langsung berguna.
2. Fase B: tambah LLM hanya untuk `summary` + `bull/bear` dengan guardrails §6.
3. Fase C: investigasi penuh (8 tools) + confidence terkalibrasi dari histori
   (apakah Top 10 outperform? → `docs/backtesting`).

## 12. Definisi selesai (DoD) untuk AI layer

- [ ] Quant output tidak berubah (test `service.Rank` tetap hijau).
- [ ] Analisis selalu mencantumkan `run_id` + peringatan stale bila perlu.
- [ ] Tidak ada angka baru yang tidak bisa ditelusur ke input/tool.
- [ ] Kegagalan AI tidak merusak ranking/dashboard (fallback quant-only).
- [ ] Budget: ≤ 10 pemanggilan LLM per run 6 jam; tidak ada hit Sectors dari agent.
