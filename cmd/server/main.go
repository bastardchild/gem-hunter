package main

import (
	"context"
	"embed"
	"log"
	"os"
	"sort"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"gemhunter/internal/ai"
	"gemhunter/internal/config"
	"gemhunter/internal/gemguard"
	"gemhunter/internal/notifier"
	"gemhunter/internal/repository"
	"gemhunter/internal/service"
	"gemhunter/web"
)

//go:embed migrations.sql
var migrationsFS embed.FS

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Printf("WARN config: %v (lanjut dengan mock universe)", err)
		cfg = fallbackConfig()
	}
	schema, err := migrationsFS.ReadFile("migrations.sql")
	if err != nil {
		log.Fatalf("migrations.sql: %v", err)
	}
	store, err := repository.New(cfg.DBPath, string(schema))
	if err != nil {
		log.Fatalf("sqlite open %q: %v", cfg.DBPath, err)
	}
	defer store.Close()

	aiAgent := ai.NewAgent(store)
	aiWorker := ai.NewWorker(aiAgent)
	aiWorker.Start(context.Background())
	defer aiWorker.Stop()

	mailer := notifier.New(cfg, store)

	run := func() service.RunResult {
		nowWIB := time.Now().In(wib)
		r := service.Rank(service.MockUniverse(), nowWIB, cfg.MaxDataAgeHours, cfg.MinEPSGrowth)
		r.RunID = uuid.NewString()
		r.CalculatedAt = nowWIB.Format("2006-01-02 15:04")
		r.DataAsOf = nowWIB.Format("2006-01-02 15:04")
		if err := store.Save(context.Background(), r); err != nil {
			log.Printf(`{"level":"error","event":"ranking_save","error":%q}`, err.Error())
		}
		log.Printf(`{"level":"info","event":"ranking_run","run_id":%q,"count":%d,"db":%q}`, r.RunID, r.Count, cfg.DBPath)
		aiWorker.Enqueue(r)

		// Dispatch email alerts ke subscribers
		go mailer.DispatchScheduledDigest(
			context.Background(),
			r,
			defaultGemGuardSurveillance(),
			service.GetTop5SpringateDistress(service.MockUniverse()),
		)

		return r
	}
	run()
	// Scheduler: jalankan otomatis tepat pada jam 00:00, 06:00, 12:00, 18:00 WIB
	go func() {
		for {
			next := nextScheduleSlot(time.Now())
			sleepDur := time.Until(next)
			log.Printf(`{"level":"info","event":"scheduler_sleep","next_slot":%q,"duration":%q}`, next.Format("2006-01-02 15:04 WIB"), sleepDur.String())
			time.Sleep(sleepDur)
			run()
		}
	}()

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Set("X-Request-ID", uuid.NewString())
		c.Set("X-Content-Type-Options", "nosniff")
		return c.Next()
	})
	app.Get("/health", func(c *fiber.Ctx) error { return c.SendString("ok") })
	app.Get("/ready", func(c *fiber.Ctx) error { return c.SendString("ready") })
	app.Get("/ranking/top10", func(c *fiber.Ctx) error {
		if c.Get("HX-Request") != "" {
			html, err := renderRows(mustLatest(store, cfg, run).Stocks)
			if err != nil {
				return fiber.NewError(500, err.Error())
			}
			return c.Type("html").SendString(html)
		}
		return c.JSON(mustLatest(store, cfg, run))
	})
	app.Get("/api/v1/ranking/top10", func(c *fiber.Ctx) error {
		return c.JSON(mustLatest(store, cfg, run))
	})
	app.Get("/ranking", func(c *fiber.Ctx) error { return c.JSON(mustLatest(store, cfg, run)) })
	app.Get("/stocks", func(c *fiber.Ctx) error { return c.JSON(mustLatest(store, cfg, run).Stocks) })
	app.Get("/stocks/:ticker", func(c *fiber.Ctx) error {
		r := mustLatest(store, cfg, run)
		for _, s := range r.Stocks {
			if s.Ticker == c.Params("ticker") {
				return c.JSON(s)
			}
		}
		return c.Status(404).JSON(fiber.Map{"error": "not found"})
	})
	app.Post("/admin/ranking/run", func(c *fiber.Ctx) error {
		if cfg.AdminToken == "" || c.Get("Authorization") != "Bearer "+cfg.AdminToken {
			return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
		}
		return c.JSON(run())
	})
	app.Get("/", func(c *fiber.Ctx) error {
		html, err := renderDashboard(mustLatest(store, cfg, run), cfg.RankingIntervalHours)
		if err != nil {
			return fiber.NewError(500, err.Error())
		}
		return c.Type("html").SendString(html)
	})
	app.Get("/methodology", func(c *fiber.Ctx) error {
		html, err := renderMethodology()
		if err != nil {
			return fiber.NewError(500, err.Error())
		}
		return c.Type("html").SendString(html)
	})
	app.Get("/proof", func(c *fiber.Ctx) error {
		html, err := renderProof()
		if err != nil {
			return fiber.NewError(500, err.Error())
		}
		return c.Type("html").SendString(html)
	})
	app.Get("/gemguard", func(c *fiber.Ctx) error {
		stocks := defaultGemGuardSurveillance()
		html, err := renderGemGuard(stocks)
		if err != nil {
			return fiber.NewError(500, err.Error())
		}
		return c.Type("html").SendString(html)
	})
	app.Get("/api/v1/gemguard", func(c *fiber.Ctx) error {
		stocks := defaultGemGuardSurveillance()
		return c.JSON(stocks)
	})
	app.Get("/gemsentinel", func(c *fiber.Ctx) error {
		stocks := service.GetTop5SpringateDistress(service.MockUniverse())
		html, err := renderGemSentinel(stocks)
		if err != nil {
			return fiber.NewError(500, err.Error())
		}
		return c.Type("html").SendString(html)
	})
	app.Get("/api/v1/gemsentinel", func(c *fiber.Ctx) error {
		stocks := service.GetTop5SpringateDistress(service.MockUniverse())
		return c.JSON(stocks)
	})
	app.Post("/api/v1/subscribe", func(c *fiber.Ctx) error {
		var req repository.Subscription
		if err := c.BodyParser(&req); err != nil || req.Email == "" {
			return c.Status(400).JSON(fiber.Map{"error": "Format request/email tidak valid"})
		}
		if err := store.SaveSubscription(c.Context(), req); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Gagal menyimpan preferensi email: " + err.Error()})
		}
		return c.JSON(fiber.Map{"status": "success", "message": "Subscription berhasil disimpan"})
	})
	app.Post("/api/v1/email/test", func(c *fiber.Ctx) error {
		var req struct {
			Email string `json:"email"`
		}
		if err := c.BodyParser(&req); err != nil || req.Email == "" {
			return c.Status(400).JSON(fiber.Map{"error": "Format email tidak valid"})
		}
		if err := mailer.SendTestEmail(c.Context(), req.Email); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Gagal mengirim email uji coba: " + err.Error()})
		}
		return c.JSON(fiber.Map{"status": "success", "message": "Email tes berhasil dikirim"})
	})
	app.Get("/api/v1/ai/analysis/:ticker", func(c *fiber.Ctx) error {
		ticker := c.Params("ticker")
		an, ok := aiAgent.GetAnalysis(ticker)
		if !ok {
			return c.Status(444).Status(404).JSON(fiber.Map{"error": "analysis pending or not found"})
		}
		return c.JSON(an)
	})
	app.Get("/api/v1/ai/run/:run_id", func(c *fiber.Ctx) error {
		runID := c.Params("run_id")
		an, ok := aiAgent.GetRunAnalyses(runID)
		if !ok {
			return c.Status(404).JSON(fiber.Map{"error": "run analysis not found"})
		}
		return c.JSON(an)
	})
	app.Post("/admin/ai/run", func(c *fiber.Ctx) error {
		if cfg.AdminToken == "" || c.Get("Authorization") != "Bearer "+cfg.AdminToken {
			return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
		}
		r := mustLatest(store, cfg, run)
		aiWorker.Enqueue(r)
		return c.JSON(fiber.Map{"status": "enqueued", "run_id": r.RunID})
	})
	app.Get("/static/*", func(c *fiber.Ctx) error {
		p := c.Params("*")
		data, err := web.FS.ReadFile("static/" + p)
		if err != nil {
			return fiber.ErrNotFound
		}
		if len(p) >= 4 && p[len(p)-4:] == ".css" {
			c.Type("css")
		}
		return c.Send(data)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	log.Fatal(app.Listen(":" + port))
}

func fallbackConfig() config.Config {
	dbPath := os.Getenv("SQLITE_PATH")
	if dbPath == "" {
		dbPath = "data/gemhunter.db"
	}
	return config.Config{Env: getenvDefault("APP_ENV", "development"), DBPath: dbPath, RankingIntervalHours: 6, MaxDataAgeHours: 168, AdminToken: os.Getenv("ADMIN_TOKEN")}
}

func getenvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func defaultGemGuardSurveillance() []gemguard.StockSecurityAnalysis {
	list := []gemguard.StockSecurityAnalysis{
		{
			Ticker:            "PUMP",
			CompanyName:       "Pratama Unggul Makmur PT",
			Sector:            "Consumer Non-Cyclicals",
			Price:             450,
			MarketCap:         150000000000,
			FreeFloat:         50000000,
			Velocity:          0.005,
			PriceImpactRatio:  4.50,
			VolumeSpike:       5.50,
			PriceSpikePct:     34.50,
			ARAThreshold:      35.0,
			ARACountConsec:    2,
			DER:               3.20,
			CurrentRatio:      0.65,
			ROE:               2.10,
			EPS:               1.20,
			InsiderHoldingPct: 78.5,
			InsiderSellingPct: 7.50,
			Volatility:        0.18,
			RSI:               88.0,
		},
		{
			Ticker:            "GOTO",
			CompanyName:       "GoTo Gojek Tokopedia Tbk",
			Sector:            "Technology",
			Price:             68,
			MarketCap:         81000000000000,
			FreeFloat:         75000000000,
			Velocity:          0.045,
			PriceImpactRatio:  1.42,
			VolumeSpike:       3.10,
			PriceSpikePct:     12.50,
			ARAThreshold:      35.0,
			ARACountConsec:    0,
			DER:               0.25,
			CurrentRatio:      1.80,
			ROE:               -5.4,
			EPS:               -8.50,
			InsiderHoldingPct: 42.0,
			InsiderSellingPct: 1.20,
			Volatility:        0.06,
			RSI:               62.0,
		},
		{
			Ticker:            "BBCA",
			CompanyName:       "Bank Central Asia Tbk",
			Sector:            "Financials",
			Price:             9800,
			MarketCap:         1208000000000000,
			FreeFloat:         45000000000,
			Velocity:          0.012,
			PriceImpactRatio:  0.25,
			VolumeSpike:       0.95,
			PriceSpikePct:     0.80,
			ARAThreshold:      20.0,
			ARACountConsec:    0,
			DER:               0.0,
			CurrentRatio:      1.20,
			ROE:               21.80,
			EPS:               395.0,
			InsiderHoldingPct: 54.9,
			InsiderSellingPct: 0.00,
			Volatility:        0.015,
			RSI:               54.0,
		},
		{
			Ticker:            "ASII",
			CompanyName:       "Astra International Tbk",
			Sector:            "Industrially Cyclical",
			Price:             5150,
			MarketCap:         208480000000000,
			FreeFloat:         20000000000,
			Velocity:          0.018,
			PriceImpactRatio:  0.55,
			VolumeSpike:       1.20,
			PriceSpikePct:     1.80,
			ARAThreshold:      20.0,
			ARACountConsec:    0,
			DER:               1.02,
			CurrentRatio:      1.45,
			ROE:               16.20,
			EPS:               780.0,
			InsiderHoldingPct: 50.1,
			InsiderSellingPct: 0.00,
			Volatility:        0.022,
			RSI:               48.0,
		},
		{
			Ticker:            "TLKM",
			CompanyName:       "Telkom Indonesia Tbk",
			Sector:            "Infrastructure",
			Price:             3750,
			MarketCap:         371480000000000,
			FreeFloat:         47000000000,
			Velocity:          0.021,
			PriceImpactRatio:  0.42,
			VolumeSpike:       1.05,
			PriceSpikePct:     -0.50,
			ARAThreshold:      25.0,
			ARACountConsec:    0,
			DER:               0.78,
			CurrentRatio:      0.95,
			ROE:               18.50,
			EPS:               245.0,
			InsiderHoldingPct: 52.1,
			InsiderSellingPct: 0.00,
			Volatility:        0.018,
			RSI:               42.0,
		},
	}

	for i := range list {
		gemguard.ComputeRiskScore(&list[i])
	}
	// Sort descending by risk score so the highest-risk (most urgent) rows appear first.
	sort.Slice(list, func(i, j int) bool {
		if list[i].RiskScore != list[j].RiskScore {
			return list[i].RiskScore > list[j].RiskScore
		}
		return list[i].Ticker < list[j].Ticker
	})
	for i := range list {
		list[i].Rank = i + 1
	}
	return list
}

func mustLatest(s *repository.Store, cfg config.Config, run func() service.RunResult) service.RunResult {
	if r, ok := s.Latest(); ok {
		return r
	}
	return run()
}
