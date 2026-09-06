# FORMULA-LOGIC — Dasar Seluruh Perhitungan Gem Hunter

> Sumber: `task.md` §5–14 + §38, `formula-hard.md`, `skill.md`, `harness.md`.
> Dokumen ini NORMATIF. Jika ada konflik, dokumen ini yang menang untuk implementasi.
> Bahasa skor: "quantitative screening score" / "ranking signal". Bukan rekomendasi investasi.

## 0. Konvensi

- `null` = tidak valid / tidak dihitung / tidak eligible untuk komponen itu. Di Go: gunakan `*float64` (nil) untuk semua metric turunan, bukan 0.
- Satuan: `Price, EPS, BVPS, GrahamValue` dalam Rupiah per lembar (konsisten). `PE, PB, PEG` rasio murni.
- Growth disimpan sebagai **fraksi desimal** (`0.20` = 20%), TETAPI denominator PEG memakai **persen** (`20`). Lihat §4.
- Semua perbandingan float pakai toleransi `eps = 1e-9`; `x > 0` berarti `x > eps`.
- Waktu: input engine = snapshot (tanpa `time.Now()`, tanpa HTTP). `available_at <= ranking_time` untuk backtest.

## 1. Input Kanonis

### 1.1 CompanyFinancialSnapshot (per ticker, per run)

```
ticker, companyName, sector, industry, isFinancial (bank/finance → profile khusus)
price, marketCap
eps, prevEps              # TTM / tahun berjalan vs periode pembanding
bvps                      # book value per share
pe, pb
roe                       # fraksi desimal, cth 0.18
der                       # hanya dipakai non-financial; bank = N/A
revenue, prevRevenue
dividendYield
dataDate, periodEnd, publishedAt, fetchedAt
```

### 1.2 Mapping Sectors V2 → snapshot (diisi setelah baca dok resmi V2, jangan karang field)

| Rumus | Butuh | Kandidat field Sectors V2 |
|---|---|---|
| GrahamValue | EPS + BVPS | financials.eps / valuation.bvps (verifikasi) |
| MOS | GrahamValue + Price | daily.close |
| PE score | PE | valuation.pe |
| PB score | PB | valuation.pb |
| EPSGrowth | EPS historis | financials.eps_t, eps_t-1 |
| RevenueGrowth | revenue historis | financials.revenue_t, revenue_t-1 |
| ROE | netIncome + equity | financials.roe atau hitung |
| PEG | PE + EPSGrowth | turunan |

Aturan mapping: (1) cari endpoint V2 lain, (2) hitung dari field tersedia, (3) jika tetap tak ada → `null` + tandai `unavailable`.

## 2. Graham Engine

### 2.1 GrahamValue

```
IF eps == null OR bvps == null OR eps <= 0 OR bvps <= 0:
  GrahamValue = null
ELSE:
  GrahamValue = sqrt(22.5 * eps * bvps)
```

### 2.2 Margin of Safety (MOS, fraksi desimal)

```
IF GrahamValue == null OR price == null OR price <= 0 OR GrahamValue <= 0:
  MOS = null
ELSE:
  MOS = 1 - price / GrahamValue
      = (GrahamValue - price) / GrahamValue
```

MOS bisa negatif (overvalued) — tetap valid selama input valid. Arah: **higher-is-better**.

### 2.3 Contoh (§6 formula-hard, diverifikasi)

price=5000, eps=500, bvps=4000 → `GrahamValue = sqrt(22.5*500*4000) = 6708.20…` → `MOS = 1 - 5000/6708.20 = 0.2547` (25.47%).

## 3. Growth Engine

### 3.1 EPSGrowth (fraksi desimal)

```
YoY sederhana:
IF eps == null OR prevEps == null OR prevEps <= 0:
  EPSGrowth = null
ELSE:
  EPSGrowth = eps / prevEps - 1

CAGR multi-periode (n tahun, bila tersedia):
IF eps_t-n <= 0: EPSGrowth = null
ELSE: EPSGrowth = (eps_t / eps_t-n)^(1/n) - 1
```

Jangan beri skor palsu bila denominator ≤ 0. Arah: **higher-is-better**.

### 3.2 RevenueGrowth (fraksi desimal)

```
IF revenue == null OR prevRevenue == null OR prevRevenue <= 0:
  RevenueGrowth = null
ELSE:
  RevenueGrowth = revenue / prevRevenue - 1
```

Arah: **higher-is-better**.

## 4. Lynch Engine (PEG)

```
EPSGrowthPercent = EPSGrowth * 100          # 0.20 → 20

IF pe == null OR pe <= 0: PEG = null
ELSE IF EPSGrowth == null OR EPSGrowth <= 0: PEG = null
ELSE IF EPSGrowthPercent <= 0: PEG = null
ELSE: PEG = pe / EPSGrowthPercent
```

Contoh normatif: PE=15, growth=20% → `PEG = 15/20 = 0.75`. Contoh ABC: PE=10 → `PEG = 10/20 = 0.50`.
Kesalahan fatal: memakai `0.20` sebagai denominator (hasil 75, salah 100x). Arah PEG: **lower-is-better**.

## 5. Percentile S(Metric) — satu-satunya metode yang dipakai

Metode: **percent-rank dengan average-rank untuk ties, dinormalisasi ke 0–100**.

```
Diberikan v_1..v_n (hanya nilai non-null yang eligible untuk metric itu):
1. Urutkan ascending. Ties (selisih <= 1e-9) dapat rank rata-rata.
2. Untuk nilai v dengan average rank r (1-indexed):
     p = (r - 1) / (n - 1)      # n > 1
     p = 0.5                    # n == 1 → skor 50
3. higher-is-better: S = p * 100
   lower-is-better:  S = (1 - p) * 100
```

Sifat: min→0 (higher) / 100 (lower), max→100/0, median ≈ 50, ties sama. Nilai `null` tidak ikut universe dan tidak dapat skor komponen itu.

Arah per metric:

| higher-is-better | lower-is-better |
|---|---|
| MOS, EPSGrowth, RevenueGrowth, ROE | PE, PB, PEG |

Contoh: ROE percentile-90 → `S(ROE)=90`. PE percentile-10 (termurah) → `S(PE)=90`.

## 6. GrahamScore (0–100)

```
Butuh: S(MOS), S(PE), S(PB), S(EPSGrowth), S(ROE)
Bobot tetap:
  GrahamScore = 0.40*S(MOS) + 0.20*S(PE) + 0.15*S(PB)
              + 0.15*S(EPSGrowth) + 0.10*S(ROE)
```

Kandidat §2 formula-hard (`PE<15, PB<1.5, DER<1`) TIDAK dipakai sebagai hard filter (alasan diversifikasi, temuan Kartikasari). Mereka hanya memengaruhi skor via percentile.

### 6.1 Verifikasi ABC

`S = {MOS:82, PE:90, PB:85, EPSGrowth:80, ROE:78}` →
`0.40*82 + 0.20*90 + 0.15*85 + 0.15*80 + 0.10*78 = 32.8+18+12.75+12+7.8 = 83.35` → **83.35**
(Catatan: `formula-hard.md` menulis 83.3 — pembulatan; nilai eksak 83.35.)

## 7. LynchScore (0–100)

```
LynchScore = 0.50*S(PEG) + 0.25*S(EPSGrowth)
           + 0.15*S(RevenueGrowth) + 0.10*S(ROE)
```

Verifikasi ABC: `S = {PEG:95, EPSGrowth:80, RevenueGrowth:75, ROE:78}` →
`47.5+20+11.25+7.8 = 86.55` → **86.55**
(Catatan: `formula-hard.md` menulis 86.05 — salah hitung; 86.55 yang benar.)

## 8. Composite GLScore (0–100) — FINAL

```
GLScore = 0.55 * GrahamScore + 0.45 * LynchScore
```

Varian 50/50 di §5 formula-hard DITOLAK untuk MVP. Kunci: **55/45** (murah + bertumbuh, bobot ke value).
Verifikasi ABC dengan nilai koreksi: `0.55*83.35 + 0.45*86.55 = 45.8425 + 38.9475 = 84.79` → **84.79**
(Jika memakai angka lama yang keliru 83.3/86.05 → 84.54; pakai 84.79 sebagai test vector baru.)

Label: 90–100 Exceptional, 80–89.99 Strong, 70–79.99 Attractive, 60–69.99 Neutral, <60 Weak.

## 9. Profil Bank / Finansial

- `isFinancial == true` → pakai `GrahamLynchFinancialProfile`: **abaikan DER sepenuhnya**, tampilkan `DER = N/A`.
- Jika suatu komponen null karena karakteristik bank (mis. BVPS tidak bermakna): **renormalisasi bobot proporsional** atas komponen yang tersedia, bukan memberi 0.
  ```
  w'_i = w_i / Σ(w_j tersedia); Score = Σ(w'_j * S_j)
  ```
- Jangan biarkan bank sistematis rendah hanya karena neraca berbeda. Tandai di `ScoreBreakdown` komponen mana yang N/A + apakah renormalisasi terjadi.

## 10. Eligibility (sebelum ranking)

Saham masuk universe hitung skor bila:

```
valid ticker, price > 0, eps valid
GrahamValue != null            # eps>0 & bvps>0
pe != null AND pe > 0
data tidak stale: (ranking_time - dataDate) <= MAX_DATA_AGE_HOURS
DataQuality >= threshold (completeness, freshness, consistency)
```

Untuk komponen Lynch (bukan eliminasi total bila hanya Lynch yang gagal — lihat §11):

```
EPSGrowth != null AND EPSGrowth > MIN_EPS_GROWTH (default 0)
PEG != null AND PEG > 0
```

Threshold via env: `MIN_EPS_GROWTH, MAX_DATA_AGE_HOURS, MIN_MARKET_CAP, MIN_AVG_VALUE_20D`. Jangan set terlalu ketat sampai universe < ~20.

## 11. Aturan Skor Parsial (anti skor palsu)

- `GrahamValue null` → `MOS null` → saham **tidak dapat GrahamScore maupun GLScore** (exclude dari ranking).
- `PEG null` → **tidak dapat LynchScore**. Kebijakan MVP: saham tanpa LynchScore **tidak dapat GLScore** (karena GL butuh keduanya) → exclude. Alternatif masa depan: GL = Graham saja dengan flag `lynch_unavailable` — belum dipakai sekarang.
- Komponen `S(x)` hanya dihitung dari universe non-null metric itu. Saham dengan metric null tidak mendapat kontribusi palsu.
- `DataQuality < threshold` → exclude, atau `score = unavailable` sesuai config. Tidak pernah isi 0 diam-diam.

## 12. Ranking → Top 10

```
Urutkan eligible desc: GLScore → GrahamScore → LynchScore → MOS → EPSGrowth
Ambil 10 pertama → RankedStock{
  rank, ticker, companyName, sector, price,
  glScore, grahamScore, lynchScore,
  grahamValue, marginOfSafety, pe, pb, peg,
  epsGrowth, revenueGrowth, roe,
  scoreBreakdown, dataDate, calculatedAt }
```

Tie-break deterministik: bila semua kunci sama → ticker ascending.

## 13. Tabel Keputusan Edge-Case (wajib lulus test)

| Input | Hasil |
|---|---|
| eps ≤ 0 atau bvps ≤ 0 | GrahamValue=null, MOS=null, exclude |
| price ≤ 0 / null | MOS=null, exclude |
| prevEps ≤ 0 / null | EPSGrowth=null |
| prevRevenue ≤ 0 / null | RevenueGrowth=null |
| pe ≤ 0 / null | PEG=null, no LynchScore, exclude dari GL |
| EPSGrowth ≤ 0 / null | PEG=null, no LynchScore, exclude dari GL |
| n == 1 di percentile | S = 50 untuk metric itu |
| semua null satu metric | tidak ada skor komponen itu untuk siapa pun |
| universe kosong | ranking = [] (bukan error panic), job SUCCESS dengan eligible=0 |
| duplikat ticker | dedup: pakai snapshot dengan `publishedAt` terbaru |
| bank | DER=N/A, renormalisasi bila komponen hilang |

## 14. Signature Go (acuan implementasi)

```go
func GrahamValue(eps, bvps float64) *float64
func MarginOfSafety(price, grahamValue float64) *float64
func EPSGrowth(eps, prevEps float64) *float64
func RevenueGrowth(rev, prevRev float64) *float64
func PEG(pe, epsGrowthFrac float64) *float64   // terima fraksi, kali 100 di dalam
func PercentileScore(values []float64, v float64, lowerIsBetter bool) float64
func GrahamScore(mos, pe, pb, epsG, roe float64) float64
func LynchScore(peg, epsG, revG, roe float64) float64
func GLScore(graham, lynch float64) float64   // 0.55/0.45
```

Semua `*float64` constructor mengembalikan nil bila invalid (§2–4). `PercentileScore` mengasumsikan `values` sudah non-null.

## 15. Test Vector Normatif

```
ABC: price=5000 eps=500 bvps=4000 pe=10 pb=1.25
     epsGrowth=0.20 revGrowth=0.15 roe=0.18
→ GrahamValue=6708.20 MOS=0.2547 PEG=0.50
→ S Graham {82,90,85,80,78} → GrahamScore=83.35
→ S Lynch {95,80,75,78} → LynchScore=86.55
→ GLScore=84.79 → label Strong
```

Test wajib: tiap fungsi §14 + tiap baris §13 + vector ABC + konsistensi `S(lower)` vs `S(higher)`.
