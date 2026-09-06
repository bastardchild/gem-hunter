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
	}
	if c.SectorsAPIKey == "" {
		return c, fmt.Errorf("SECTORS_API_KEY is required")
	}
	return c, nil
}
