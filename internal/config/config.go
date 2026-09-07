package config

import (
	"fmt"
	"os"
	"strconv"
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
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func Load() (Config, error) {
	interval, _ := strconv.Atoi(getenv("RANKING_INTERVAL_HOURS", "6"))
	maxAge, _ := strconv.Atoi(getenv("MAX_DATA_AGE_HOURS", "168"))
	minG, _ := strconv.ParseFloat(getenv("MIN_EPS_GROWTH", "0"), 64)
	smtpPort, _ := strconv.Atoi(getenv("SMTP_PORT", "587"))
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
	}
	if c.SectorsAPIKey == "" {
		return c, fmt.Errorf("SECTORS_API_KEY is required")
	}
	return c, nil
}
