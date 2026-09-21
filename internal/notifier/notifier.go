package notifier

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"

	"gemhunter/internal/config"
	"gemhunter/internal/domain"
	"gemhunter/internal/gemguard"
	"gemhunter/internal/repository"
	"gemhunter/internal/service"
)

type Mailer struct {
	cfg   config.Config
	store *repository.Store
}

func New(cfg config.Config, store *repository.Store) *Mailer {
	return &Mailer{cfg: cfg, store: store}
}

func envelopeAddress(from string) string {
	from = strings.TrimSpace(from)
	if from == "" {
		return "alerts@gemhunter.app"
	}
	parsed, err := mail.ParseAddress(from)
	if err != nil || parsed.Address == "" {
		return from
	}
	return parsed.Address
}

// SendRaw mengirim email via SMTP (support TLS / STARTTLS).
func (m *Mailer) SendRaw(to []string, subject, htmlBody string) error {
	if m.cfg.SMTPHost == "" {
		return fmt.Errorf("SMTP_HOST belum dikonfigurasi di .env")
	}

	fromHeader := strings.TrimSpace(m.cfg.SMTPFrom)
	if fromHeader == "" {
		fromHeader = "Gem Hunter <alerts@gemhunter.app>"
	}
	fromAddr := envelopeAddress(fromHeader)

	// Format MIME Header
	headers := make(map[string]string)
	headers["From"] = fromHeader
	headers["To"] = strings.Join(to, ", ")
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"
	headers["Date"] = time.Now().Format(time.RFC1123Z)

	var msg bytes.Buffer
	for k, v := range headers {
		msg.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	msg.WriteString("\r\n")
	msg.WriteString(htmlBody)

	addr := fmt.Sprintf("%s:%d", m.cfg.SMTPHost, m.cfg.SMTPPort)

	// Auth jika user/pass tersedia
	var auth smtp.Auth
	if m.cfg.SMTPUser != "" && m.cfg.SMTPPassword != "" {
		auth = smtp.PlainAuth("", m.cfg.SMTPUser, m.cfg.SMTPPassword, m.cfg.SMTPHost)
	}

	// Jika port 465 (SMTPS / SSL langsung)
	if m.cfg.SMTPPort == 465 {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: false,
			ServerName:         m.cfg.SMTPHost,
		}
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("tls dial: %w", err)
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, m.cfg.SMTPHost)
		if err != nil {
			return fmt.Errorf("smtp new client: %w", err)
		}
		defer client.Close()

		if auth != nil {
			if ok, _ := client.Extension("AUTH"); ok {
				if err = client.Auth(auth); err != nil {
					return fmt.Errorf("smtp auth: %w", err)
				}
			}
		}

		if err = client.Mail(fromAddr); err != nil {
			return fmt.Errorf("smtp mail from: %w", err)
		}
		for _, addr := range to {
			if err = client.Rcpt(addr); err != nil {
				return fmt.Errorf("smtp rcpt to (%s): %w", addr, err)
			}
		}

		w, err := client.Data()
		if err != nil {
			return fmt.Errorf("smtp data: %w", err)
		}
		_, err = w.Write(msg.Bytes())
		if err != nil {
			return fmt.Errorf("write email body: %w", err)
		}
		return w.Close()
	}

	// Default: Standard STARTTLS (Port 587 / 25)
	host, _, _ := net.SplitHostPort(addr)
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         host,
	}

	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("tcp dial: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("smtp new client: %w", err)
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); ok {
		if err = client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("starttls: %w", err)
		}
	}

	if auth != nil {
		if ok, _ := client.Extension("AUTH"); ok {
			if err = client.Auth(auth); err != nil {
				return fmt.Errorf("smtp auth: %w", err)
			}
		}
	}

	if err = client.Mail(fromAddr); err != nil {
		return fmt.Errorf("mail from: %w", err)
	}
	for _, addr := range to {
		if err = client.Rcpt(addr); err != nil {
			return fmt.Errorf("rcpt to (%s): %w", addr, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("data: %w", err)
	}
	if _, err = w.Write(msg.Bytes()); err != nil {
		return fmt.Errorf("write data: %w", err)
	}
	return w.Close()
}

// SendTestEmail mengirim email uji coba format ringkas.
func (m *Mailer) SendTestEmail(ctx context.Context, email string) error {
	subject := "◆ [Gem Hunter] Test Alert & Notification Verification"
	html := fmt.Sprintf(`
	<!DOCTYPE html>
	<html>
	<head>
		<meta charset="utf-8">
	</head>
	<body style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;background-color:#0c0a09;color:#ffffff;padding:20px;margin:0;">
		<div style="background-color:#1c1917;border:1px solid #27272a;border-radius:8px;padding:24px;max-width:540px;margin:0 auto;color:#ffffff;">
			<div style="margin-bottom:12px;">
				<span style="font-size:20px;color:#d53e4f;">◆</span>
				<strong style="letter-spacing:1.5px;font-size:16px;color:#ffffff;">GEM HUNTER QUANT ALERTS</strong>
			</div>
			<p style="color:#ffffff;">Hello,</p>
			<p style="color:#ffffff;">This is a confirmation message that your <strong style="color:#ffffff;">Gem Hunter</strong> email alert integration is now active!</p>
			<p style="color:#ffffff;">You will receive automated surveillance digests tailored to your preferences during scheduled screening runs (00:00, 06:00, 12:00, 18:00 WIB) or upon critical market anomalies.</p>
			<div style="margin:16px 0;">
				<span style="background-color:#1f3d2a;color:#abdda4;padding:4px 10px;border-radius:999px;font-weight:bold;font-size:12px;">SMTP Connection: Verified &amp; Ready</span>
			</div>
			<div style="font-size:11px;color:#a1a1aa;margin-top:20px;text-align:center;">
				Sent automatically at %s WIB · Gem Hunter Quantitative Engine
			</div>
		</div>
	</body>
	</html>
	`, time.Now().Format("2006-01-02 15:04:05"))

	err := m.SendRaw([]string{email}, subject, html)
	status := "SUCCESS"
	errMsg := ""
	if err != nil {
		status = "FAILED"
		errMsg = err.Error()
	}
	_ = m.store.LogEmail(ctx, email, "TEST", subject, status, errMsg)
	return err
}

type DigestData struct {
	CalculatedAt string
	TopGems      []service.RankedStock
	GuardStocks  []gemguard.StockSecurityAnalysis
	Sentinel     []domain.SentinelStock
}

var digestTmpl = template.Must(template.New("digest").Funcs(template.FuncMap{
	"mul": func(a, b float64) float64 {
		return a * b
	},
}).Parse(`
<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
</head>
<body style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;background:#0c0a09;color:#ffffff;margin:0;padding:24px 12px;">
<div style="max-width:600px;margin:0 auto;background:#1c1917;border:1px solid #27272a;border-radius:10px;overflow:hidden;color:#ffffff;">
	<div style="padding:20px 24px;border-bottom:1px solid #27272a;background:#141210;">
		<div style="font-weight:800;font-size:16px;letter-spacing:1.5px;color:#ffffff;">◆ GEM HUNTER · SURVEILLANCE & QUANT DIGEST</div>
		<div style="font-size:12px;color:#a1a1aa;margin-top:4px;">Scheduled Screening Update: {{.CalculatedAt}} WIB</div>
	</div>

	{{if .TopGems}}
	<div style="padding:20px 24px;border-bottom:1px solid #292524;color:#ffffff;">
		<h3 style="margin-top:0;font-size:14px;text-transform:uppercase;letter-spacing:1px;color:#abdda4;">◆ Top Graham-Lynch Undervalued Gems</h3>
		<table style="width:100%;border-collapse:collapse;font-size:13px;margin-top:10px;color:#ffffff;">
			<thead>
				<tr>
					<th style="text-align:left;font-size:11px;text-transform:uppercase;color:#a1a1aa;padding:6px 8px;border-bottom:1px solid #27272a;">Rank</th>
					<th style="text-align:left;font-size:11px;text-transform:uppercase;color:#a1a1aa;padding:6px 8px;border-bottom:1px solid #27272a;">Ticker</th>
					<th style="text-align:left;font-size:11px;text-transform:uppercase;color:#a1a1aa;padding:6px 8px;border-bottom:1px solid #27272a;">Price</th>
					<th style="text-align:left;font-size:11px;text-transform:uppercase;color:#a1a1aa;padding:6px 8px;border-bottom:1px solid #27272a;">GL Score</th>
					<th style="text-align:left;font-size:11px;text-transform:uppercase;color:#a1a1aa;padding:6px 8px;border-bottom:1px solid #27272a;">Margin Safety</th>
				</tr>
			</thead>
			<tbody>
				{{range .TopGems}}
				<tr>
					<td style="padding:8px;border-bottom:1px solid #292524;color:#ffffff;">#{{.Rank}}</td>
					<td style="padding:8px;border-bottom:1px solid #292524;color:#ffffff;"><b style="color:#ffffff;">{{.Ticker}}</b></td>
					<td style="padding:8px;border-bottom:1px solid #292524;color:#ffffff;">Rp {{.Price}}</td>
					<td style="padding:8px;border-bottom:1px solid #292524;color:#abdda4;font-weight:bold;">{{printf "%.1f" .GLScore}}</td>
					<td style="padding:8px;border-bottom:1px solid #292524;color:#ffffff;">{{printf "%.1f%%" (mul .MarginOfSafety 100.0)}}</td>
				</tr>
				{{end}}
			</tbody>
		</table>
	</div>
	{{end}}

	{{if .GuardStocks}}
	<div style="padding:20px 24px;border-bottom:1px solid #292524;color:#ffffff;">
		<h3 style="margin-top:0;font-size:14px;text-transform:uppercase;letter-spacing:1px;color:#fdae61;">Gem Guard High-Risk Surveillance</h3>
		<table style="width:100%;border-collapse:collapse;font-size:13px;margin-top:10px;color:#ffffff;">
			<thead>
				<tr>
					<th style="text-align:left;font-size:11px;text-transform:uppercase;color:#a1a1aa;padding:6px 8px;border-bottom:1px solid #27272a;">Ticker</th>
					<th style="text-align:left;font-size:11px;text-transform:uppercase;color:#a1a1aa;padding:6px 8px;border-bottom:1px solid #27272a;">Risk Score</th>
					<th style="text-align:left;font-size:11px;text-transform:uppercase;color:#a1a1aa;padding:6px 8px;border-bottom:1px solid #27272a;">PIR</th>
					<th style="text-align:left;font-size:11px;text-transform:uppercase;color:#a1a1aa;padding:6px 8px;border-bottom:1px solid #27272a;">UMA Spike</th>
				</tr>
			</thead>
			<tbody>
				{{range .GuardStocks}}
				<tr>
					<td style="padding:8px;border-bottom:1px solid #292524;color:#ffffff;"><b style="color:#ffffff;">{{.Ticker}}</b></td>
					<td style="padding:8px;border-bottom:1px solid #292524;color:#d53e4f;font-weight:bold;">{{printf "%.1f" .RiskScore}} ({{.RiskLevel}})</td>
					<td style="padding:8px;border-bottom:1px solid #292524;color:#ffffff;">{{printf "%.2f" .PriceImpactRatio}}</td>
					<td style="padding:8px;border-bottom:1px solid #292524;color:#ffffff;">{{printf "%.2fx" .VolumeSpike}}</td>
				</tr>
				{{end}}
			</tbody>
		</table>
	</div>
	{{end}}

	{{if .Sentinel}}
	<div style="padding:20px 24px;border-bottom:1px solid #292524;color:#ffffff;">
		<h3 style="margin-top:0;font-size:14px;text-transform:uppercase;letter-spacing:1px;color:#d53e4f;">Gem Sentinel Distress Radar</h3>
		<table style="width:100%;border-collapse:collapse;font-size:13px;margin-top:10px;color:#ffffff;">
			<thead>
				<tr>
					<th style="text-align:left;font-size:11px;text-transform:uppercase;color:#a1a1aa;padding:6px 8px;border-bottom:1px solid #27272a;">Ticker</th>
					<th style="text-align:left;font-size:11px;text-transform:uppercase;color:#a1a1aa;padding:6px 8px;border-bottom:1px solid #27272a;">Springate Score</th>
					<th style="text-align:left;font-size:11px;text-transform:uppercase;color:#a1a1aa;padding:6px 8px;border-bottom:1px solid #27272a;">Status</th>
				</tr>
			</thead>
			<tbody>
				{{range .Sentinel}}
				<tr>
					<td style="padding:8px;border-bottom:1px solid #292524;color:#ffffff;"><b style="color:#ffffff;">{{.Ticker}}</b></td>
					<td style="padding:8px;border-bottom:1px solid #292524;color:#d53e4f;font-weight:bold;">{{printf "%.2f" .Springate.Score}}</td>
					<td style="padding:8px;border-bottom:1px solid #292524;color:#ffffff;">{{.Springate.Zone}}</td>
				</tr>
				{{end}}
			</tbody>
		</table>
	</div>
	{{end}}

	<div style="padding:16px 24px;font-size:11px;color:#a1a1aa;text-align:center;background:#141210;">
		Quantitative research digest · GEM HUNTER Framework
	</div>
</div>
</body>
</html>
`))

func (m *Mailer) DispatchScheduledDigest(ctx context.Context, r service.RunResult, guard []gemguard.StockSecurityAnalysis, sentinel []domain.SentinelStock) {
	subs, err := m.store.ListSubscriptions(ctx)
	if err != nil {
		log.Printf(`{"level":"error","event":"email_dispatch","error":%q}`, err.Error())
		return
	}
	if len(subs) == 0 {
		return
	}

	top5Gems := r.Stocks
	if len(top5Gems) > 5 {
		top5Gems = top5Gems[:5]
	}

	highRiskGuard := make([]gemguard.StockSecurityAnalysis, 0)
	for _, g := range guard {
		if g.RiskLevel == "HIGH" || g.RiskScore >= 70 {
			highRiskGuard = append(highRiskGuard, g)
		}
	}

	criticalSentinel := make([]domain.SentinelStock, 0)
	for _, s := range sentinel {
		if s.Springate.Score < 0.50 {
			criticalSentinel = append(criticalSentinel, s)
		}
	}

	funcMap := template.FuncMap{
		"mul": func(a, b float64) float64 { return a * b },
	}
	t, _ := template.New("digest").Funcs(funcMap).Parse(digestTmpl.Tree.Root.String())

	for _, sub := range subs {
		data := DigestData{
			CalculatedAt: r.CalculatedAt,
		}
		if sub.NotifyGems {
			data.TopGems = top5Gems
		}
		if sub.NotifyGuard {
			data.GuardStocks = highRiskGuard
		}
		if sub.NotifySentinel {
			data.Sentinel = criticalSentinel
		}

		if len(data.TopGems) == 0 && len(data.GuardStocks) == 0 && len(data.Sentinel) == 0 {
			continue
		}

		var buf bytes.Buffer
		if err := t.Execute(&buf, data); err != nil {
			log.Printf(`{"level":"error","event":"email_tmpl_exec","email":%q,"error":%q}`, sub.Email, err.Error())
			continue
		}

		subject := fmt.Sprintf("◆ [Gem Hunter] Surveillance Digest (%s WIB)", r.CalculatedAt)
		err := m.SendRaw([]string{sub.Email}, subject, buf.String())
		status := "SUCCESS"
		errMsg := ""
		if err != nil {
			status = "FAILED"
			errMsg = err.Error()
			log.Printf(`{"level":"error","event":"email_send_fail","email":%q,"error":%q}`, sub.Email, err.Error())
		}
		_ = m.store.LogEmail(ctx, sub.Email, "DIGEST", subject, status, errMsg)
	}
}
