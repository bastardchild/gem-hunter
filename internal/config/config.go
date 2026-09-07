package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Env                  string
	Port                 string
	DBPath               string
	LitestreamEnabled    bool
	RedisURL             string
	SectorsAPIKey        string
	SectorsBaseURL       string
	RankingIntervalHours int
	MinEPSGrowth         float64
	MaxDataAgeHours      int
	AdminToken           string
	SMTPHost             string
	SMTPPort             int
	SMTPUser             string
	SMTPPassword         string
	SMTPFrom             string
	EmailAlertsEnabled   bool

	LLMProvider  string
	LLMModel     string
	LLMAPIKey    string
	LLMBaseURL   string
	LLMEnabled   bool

	LiveSectorsEnabled   bool
	UniverseMinMarketCap float64
	UniverseMaxStocks    int
	CacheTTLHours        int
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func isEnabled(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "true", "1", "yes", "on":
		return true
	default:
		return false
	}
}

func Load() (Config, error) {
	interval, _ := strconv.Atoi(getenv("RANKING_INTERVAL_HOURS", "6"))
	maxAge, _ := strconv.Atoi(getenv("MAX_DATA_AGE_HOURS", "168"))
	minG, _ := strconv.ParseFloat(getenv("MIN_EPS_GROWTH", "0"), 64)
	smtpPort, _ := strconv.Atoi(getenv("SMTP_PORT", "587"))
	minMC, _ := strconv.ParseFloat(getenv("UNIVERSE_MIN_MARKET_CAP", "50000000000000"), 64)
	maxStk, _ := strconv.Atoi(getenv("UNIVERSE_MAX_TICKERS", "20"))
	cacheTTL, _ := strconv.Atoi(getenv("SECTORS_CACHE_TTL_HOURS", "168"))
	c := Config{
		Env:                  getenv("APP_ENV", "development"),
		Port:                 getenv("PORT", "3000"),
		DBPath:               getenv("SQLITE_PATH", "data/gemhunter.db"),
		LitestreamEnabled:    getenv("LITESTREAM_ENABLED", "false") == "true",
		RedisURL:             getenv("REDIS_URL", ""),
		SectorsAPIKey:        os.Getenv("SECTORS_API_KEY"),
		SectorsBaseURL:       getenv("SECTORS_BASE_URL", "https://api.sectors.app"),
		RankingIntervalHours: interval,
		MinEPSGrowth:         minG,
		MaxDataAgeHours:      maxAge,
		AdminToken:           os.Getenv("ADMIN_TOKEN"),
		SMTPHost:             os.Getenv("SMTP_HOST"),
		SMTPPort:             smtpPort,
		SMTPUser:             os.Getenv("SMTP_USER"),
		SMTPPassword:         os.Getenv("SMTP_PASSWORD"),
		SMTPFrom:             getenv("SMTP_FROM", "Gem Hunter <alerts@gemhunter.app>"),
		EmailAlertsEnabled:   getenv("EMAIL_ALERTS_ENABLED", "false") == "true",

		LLMProvider:          getenv("LLM_DEFAULT_PROVIDER", "openai"),
		LLMModel:             getenv("LLM_DEFAULT_MODEL", "gpt-4o-mini"),
		LLMAPIKey:            os.Getenv("OPENAI_COMPATIBLE_API_KEY"),
		LLMBaseURL:           getenv("OPENAI_COMPATIBLE_BASE_URL", "https://api.openai.com/v1"),
		LLMEnabled:           isEnabled(getenv("LLM_ENABLED", "true")) && os.Getenv("OPENAI_COMPATIBLE_API_KEY") != "",

		LiveSectorsEnabled:   os.Getenv("SECTORS_API_KEY") != "" && isEnabled(getenv("LIVE_SECTORS_ENABLED", "true")),
		UniverseMinMarketCap: minMC,
		UniverseMaxStocks:    maxStk,
		CacheTTLHours:        cacheTTL,
	}
	return c, nil
}
