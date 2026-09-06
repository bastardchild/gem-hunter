package main

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
	"time"

	"gemhunter/internal/gemguard"
	"gemhunter/internal/service"
	"gemhunter/web"
)

var tmpl = template.Must(template.New("").Funcs(template.FuncMap{
	"r2":         func(v float64) string { return fmt.Sprintf("%.2f", v) },
	"rp":         rupiah,
	"pct":        func(v float64) string { return fmt.Sprintf("%.2f%%", v*100) },
	"pctPtr":     pctPtr,
	"fval":       fval,
	"pctW":       func(v float64) string { return fmt.Sprintf("%.1f", clamp(v, 0, 100)) },
	"badge":      badgeClass,
	"badgeLabel": badgeLabel,
	"gt":         func(a, b float64) bool { return a > b },
	"gte":        func(a, b float64) bool { return a >= b },
	"lt":         func(a, b float64) bool { return a < b },
	"lte":        func(a, b float64) bool { return a <= b },
	"eq":         func(a, b interface{}) bool { return a == b },
	"topGL": func(s []service.RankedStock) float64 {
		if len(s) == 0 {
			return 0
		}
		return s[0].GLScore
	},
	"shortID": func(id string) string {
		if len(id) > 8 {
			return id[:8]
		}
		return id
	},
}).ParseFS(web.FS, "templates/*.html"))

// DashboardData is the view-model for the dashboard page.
type DashboardData struct {
	Run        service.RunResult
	NextUpdate string
	Interval   int
}

var wib = time.FixedZone("WIB", 7*3600)

// nextScheduleSlot returns the next scheduled execution time among 00:00, 06:00, 12:00, 18:00 WIB.
func nextScheduleSlot(now time.Time) time.Time {
	nowWIB := now.In(wib)
	for _, h := range []int{0, 6, 12, 18} {
		slot := time.Date(nowWIB.Year(), nowWIB.Month(), nowWIB.Day(), h, 0, 0, 0, wib)
		if slot.After(nowWIB) {
			return slot
		}
	}
	return time.Date(nowWIB.Year(), nowWIB.Month(), nowWIB.Day()+1, 0, 0, 0, 0, wib)
}

func dashboardData(r service.RunResult, intervalHours int) DashboardData {
	next := ""
	if t, err := time.ParseInLocation("2006-01-02 15:04", r.CalculatedAt, wib); err == nil {
		next = nextScheduleSlot(t).Format("15:04")
	} else {
		next = nextScheduleSlot(time.Now()).Format("15:04")
	}
	return DashboardData{Run: r, NextUpdate: next, Interval: intervalHours}
}

func renderDashboard(r service.RunResult, intervalHours int) (string, error) {
	var b bytes.Buffer
	if err := tmpl.ExecuteTemplate(&b, "dashboard.html", dashboardData(r, intervalHours)); err != nil {
		return "", err
	}
	return b.String(), nil
}

func renderMethodology() (string, error) {
	var b bytes.Buffer
	if err := tmpl.ExecuteTemplate(&b, "methodology.html", nil); err != nil {
		return "", err
	}
	return b.String(), nil
}

func renderGemGuard(stocks []gemguard.StockSecurityAnalysis) (string, error) {
	var b bytes.Buffer
	high := 0
	for _, s := range stocks {
		if s.RiskLevel == "HIGH" {
			high++
		}
	}
	data := struct {
		Stocks   []gemguard.StockSecurityAnalysis
		HighRisk int
	}{
		Stocks:   stocks,
		HighRisk: high,
	}
	if err := tmpl.ExecuteTemplate(&b, "gemguard.html", data); err != nil {
		return "", err
	}
	return b.String(), nil
}

func renderRows(stocks []service.RankedStock) (string, error) {
	var b bytes.Buffer
	if err := tmpl.ExecuteTemplate(&b, "rows", stocks); err != nil {
		return "", err
	}
	return b.String(), nil
}

func rupiah(v float64) string {
	neg := v < 0
	if neg {
		v = -v
	}
	s := fmt.Sprintf("%.0f", v)
	n := len(s)
	var out strings.Builder
	for i, ch := range s {
		if i > 0 && (n-i)%3 == 0 {
			out.WriteByte('.')
		}
		out.WriteRune(ch)
	}
	if neg {
		return "-" + out.String()
	}
	return out.String()
}

func fval(p *float64) string {
	if p == nil {
		return "n/a"
	}
	return fmt.Sprintf("%.2f", *p)
}

func pctPtr(p *float64) string {
	if p == nil {
		return "n/a"
	}
	return fmt.Sprintf("%.2f%%", *p*100)
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func badgeClass(gl float64) string {
	switch {
	case gl >= 80:
		return "strong"
	case gl >= 70:
		return "attractive"
	case gl >= 60:
		return "neutral"
	default:
		return "weak"
	}
}

func badgeLabel(gl float64) string {
	switch {
	case gl >= 90:
		return "Exceptional"
	case gl >= 80:
		return "Strong"
	case gl >= 70:
		return "Attractive"
	case gl >= 60:
		return "Neutral"
	default:
		return "Weak"
	}
}
