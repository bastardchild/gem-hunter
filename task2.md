# Formula & Deteksi Manipulasi Pasar (BEI / Sectors.app)

Rumus dan metodologi deteksi dini saham dengan kepemilikan terkonsentrasi tinggi (HSC), Unusual Market Activity (UMA), serta fitur Machine Learning untuk deteksi manipulasi saham di Bursa Efek Indonesia (BEI) menggunakan Sectors.app.

---

## 📐 1. Price Impact Ratio (Metode Resmi BEI)

BEI menggunakan rasio ini sebagai indikator deteksi dini saham dengan kepemilikan terkonsentrasi tinggi (HSC) yang berpotensi dimanipulasi.

### Rumus Lengkap:

**Langkah 1: Hitung Velocity**

$$Velocity = \frac{\text{Average Trading Volume}}{\text{Free Float}}$$

Keterangan:
- **Average Trading Volume**: rata-rata volume transaksi harian dalam periode tertentu (misal 30 hari)
- **Free Float**: jumlah saham yang beredar di publik (bukan saham treasury atau kepemilikan pengendali)

**Langkah 2: Hitung Price Impact Ratio**

$$Price\ Impact\ Ratio = \frac{\text{Percentage Price Change}}{\text{Velocity}}$$

atau

$$PIR = \frac{|\frac{P_t - P_{t-1}}{P_{t-1}} \times 100\%|}{Velocity}$$

**Interpretasi:**
- **Velocity rendah + Perubahan harga besar = PIR tinggi** -> indikasi saham sensitif terhadap transaksi kecil, berpotensi HSC/manipulasi
- BEI menerapkan screening ini untuk saham dengan kapitalisasi pasar > Rp 10 triliun
- Evaluasi dilakukan setiap kuartal

**Implementasi dengan Sectors.app:**

```python
def calculate_pir(ticker, period=30):
    # Ambil data dari Sectors API
    daily_data = sectors.get_daily(ticker)  # endpoint /daily/{ticker}
    free_float = sectors.get_free_float(ticker)  # dari /company/{ticker}
    
    # Hitung rata-rata volume
    avg_volume = daily_data['volume'].tail(period).mean()
    
    # Hitung velocity
    velocity = avg_volume / free_float
    
    # Hitung perubahan harga
    price_change_pct = (daily_data['close'].iloc[-1] / daily_data['close'].iloc[-period] - 1) * 100
    
    # Price Impact Ratio
    pir = abs(price_change_pct) / velocity if velocity > 0 else float('inf')
    
    return pir
```

---

## 📊 2. Kriteria Unusual Market Activity (UMA)

BEI menetapkan beberapa indikator yang memicu status UMA:

### a. Lonjakan Harga Ekstrem

$$Price\ Spike = \frac{P_t - P_{t-1}}{P_{t-1}} \times 100\%$$

**Ambang UMA:** Kenaikan hingga batas Auto Reject Atas (ARA) selama **> 2 hari berturut-turut**

| Harga Saham | ARA / ARB |
|:---|:---|
| > Rp 5.000 | 20% |
| Rp 200 - Rp 5.000 | 25% |
| Rp 50 - Rp 200 | 35% |

### b. Lonjakan Volume

$$Volume\ Spike = \frac{V_t}{\text{Avg Volume}_{20}}$$

**Ambang UMA:** Volume Spike > 3x - 5x dari rata-rata 20 hari

### c. Frekuensi Transaksi Melonjak

$$Frequency\ Spike = \frac{F_t}{\text{Avg Frequency}_{20}}$$

Frekuensi transaksi yang meningkat tajam tanpa informasi material

**Implementasi dengan Sectors.app:**

```python
def check_uma_indicators(ticker):
    daily = sectors.get_daily(ticker)
    
    # 1. Price Spike
    price_spike = (daily['close'].iloc[-1] / daily['close'].iloc[-2] - 1) * 100
    
    # 2. Volume Spike
    avg_volume_20 = daily['volume'].tail(20).mean()
    volume_spike = daily['volume'].iloc[-1] / avg_volume_20
    
    # 3. ARA detection (2 hari berturut-turut)
    ara_threshold = get_ara_threshold(daily['close'].iloc[-1])
    ara_days = sum(daily['change_pct'].tail(2) >= ara_threshold)
    
    return {
        'price_spike': price_spike,
        'volume_spike': volume_spike,
        'ara_consecutive_days': ara_days,
        'uma_suspected': price_spike >= ara_threshold and volume_spike >= 3
    }
```

---

## 🤖 3. Fitur-Fitur untuk Model Machine Learning (Deteksi Manipulasi)

Berdasarkan penelitian ilmiah untuk pasar IDX, berikut fitur-fitur yang digunakan untuk melatih model deteksi manipulasi:

### a. Fitur Pasar (Market Features)

| Fitur | Rumus | Periode |
|:---|:---|:---|
| **Return** | $R_t = \frac{P_t - P_{t-1}}{P_{t-1}}$ | 1, 3, 5, 10, 20 hari |
| **Volume Spike** | $VS_t = \frac{V_t}{\overline{V}_{20}}$ | 20 hari |
| **Volatilitas** | $\sigma = \sqrt{\frac{1}{n-1}\sum_{i=1}^{n}(R_i - \bar{R})^2}$ | 5, 10, 20 hari |
| **RSI** | $RSI = 100 - \frac{100}{1 + RS}$ <br>$RS = \frac{\text{Avg Gain}_{14}}{\text{Avg Loss}_{14}}$ | 14 hari |
| **Moving Average Ratio** | $MAR = \frac{P_t}{MA_{20}}$ | 20 hari |
| **Price Acceleration** | $PA = R_t - R_{t-1}$ | 1 hari |

### b. Fitur Fundamental

Berdasarkan penelitian Sergi, Wongkar & Suhariono (2025) di *American Economist*, fitur-fitur berikut terbukti memiliki korelasi dengan risiko manipulasi:

| Fitur | Rumus | Sumber Data Sectors |
|:---|:---|:---|
| **Insider Holdings** | $\frac{\text{Saham Insider}}{\text{Total Saham}} \times 100\%$ | `/insider/{ticker}` |
| **Debt to Equity (DER)** | $\frac{\text{Total Utang}}{\text{Total Ekuitas}}$ | `/financials/{ticker}` |
| **Current Ratio** | $\frac{\text{Aset Lancar}}{\text{Utang Lancar}}$ | `/financials/{ticker}` |
| **Return on Equity (ROE)** | $\frac{\text{Laba Bersih}}{\text{Total Ekuitas}} \times 100\%$ | `/financials/{ticker}` |
| **Earnings Per Share (EPS)** | $\frac{\text{Laba Bersih}}{\text{Jumlah Saham Beredar}}$ | `/financials/{ticker}` |

### c. Label Data (Ground Truth)

Label untuk training dibuat berdasarkan **pengumuman UMA resmi BEI** dengan **event window**:

$$Label_t = \begin{cases} 1 & \text{jika } t \in [UMA\_date - 5, UMA\_date + 5] \\ 0 & \text{lainnya} \end{cases}$$

---

## 🧠 4. Model yang Direkomendasikan

Berdasarkan riset untuk pasar IDX:

| Model | Akurasi | Keunggulan | Referensi |
|:---|:---|:---|:---|
| **Random Forest** | **97%** (data 3 hari) | Tertinggi untuk data UMA | IEEE Xplore 2023 |
| **Random Forest** | MCC 0.3405, F1 0.3583 | Terbaik di antara 6 model | GitHub - ikhsanmn |
| **LightGBM** | Cepat, efisien | Alternatif Random Forest | - |

**Penanganan Data Tidak Seimbang:** Gunakan **SMOTE** (Synthetic Minority Oversampling Technique) karena kasus manipulasi sangat langka (~2.8% positive rate).

---

## 💻 5. Implementasi Lengkap dengan Sectors.app

```python
import pandas as pd
import numpy as np
from sklearn.ensemble import RandomForestClassifier
from imblearn.over_sampling import SMOTE

class GorenganDetector:
    def __init__(self, sectors_client):
        self.sectors = sectors_client
        self.model = RandomForestClassifier(n_estimators=100, random_state=42)
    
    def extract_features(self, ticker):
        """Ekstrak semua fitur untuk satu saham"""
        daily = self.sectors.get_daily(ticker)
        financials = self.sectors.get_financials(ticker)
        insider = self.sectors.get_insider(ticker)
        
        features = {}
        
        # Market Features
        prices = daily['close'].values
        volumes = daily['volume'].values
        
        features['return_1d'] = (prices[-1] / prices[-2] - 1) * 100
        features['return_3d'] = (prices[-1] / prices[-4] - 1) * 100
        features['return_5d'] = (prices[-1] / prices[-6] - 1) * 100
        
        features['volume_spike'] = volumes[-1] / np.mean(volumes[-20:-1])
        features['volatility'] = np.std(np.diff(np.log(prices[-20:])))
        
        # Price Impact Ratio
        avg_volume = np.mean(volumes[-30:])
        free_float = self.sectors.get_free_float(ticker)
        velocity = avg_volume / free_float
        price_change = (prices[-1] / prices[-30] - 1) * 100
        features['pir'] = abs(price_change) / velocity if velocity > 0 else float('inf')
        
        # Fundamental Features
        features['der'] = financials['debt_to_equity']
        features['current_ratio'] = financials['current_ratio']
        features['roe'] = financials['roe']
        features['eps'] = financials['eps']
        features['insider_holdings'] = insider['total_insider_percentage']
        
        return features
    
    def predict(self, ticker):
        """Prediksi apakah saham terindikasi manipulasi"""
        features = self.extract_features(ticker)
        X = pd.DataFrame([features])
        proba = self.model.predict_proba(X)[0][1]
        return {
            'ticker': ticker,
            'risk_score': proba * 100,
            'risk_level': 'HIGH' if proba > 0.7 else 'MEDIUM' if proba > 0.4 else 'LOW',
            'features': features
        }
```

---

## 📋 6. Ringkasan Indikator & Ambang Batas

| Indikator | Rumus | Ambang Peringatan | Sumber Data Sectors |
|:---|:---|:---|:---|
| **Price Impact Ratio** | $\frac{\Delta P\%}{Velocity}$ | > 2.0 (saham besar) | `/daily/`, `/company/` |
| **Volume Spike** | $\frac{V_t}{\bar{V}_{20}}$ | > 3x - 5x | `/daily/` |
| **Price Spike** | $\frac{P_t - P_{t-1}}{P_{t-1}}$ | Mendekati ARA (20-35%) | `/daily/` |
| **Insider Selling** | $\frac{\text{Saham Dijual}}{\text{Total Saham}}$ | > 5% dalam 1 bulan | `/insider/` |
| **DER** | $\frac{\text{Utang}}{\text{Ekuitas}}$ | > 2.0 | `/financials/` |
