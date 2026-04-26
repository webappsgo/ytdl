// Package service - Email/SMTP notification service with auto-detection.
// See AI.md PART 18 for email and notification specifications.
// Auto-detects local SMTP on first run per priority order.
// Template-based emails for all notifications.
package service

import (
	"fmt"
	"log"
	"net"
	"net/smtp"
	"os"
	"strings"
	"time"
)

// EmailService handles sending email notifications
type EmailService struct {
	host     string
	port     int
	username string
	password string
	from     string
	fromName string
	tls      string
	enabled  bool
}

// EmailConfig holds SMTP configuration
type EmailConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	TLS      string `yaml:"tls"`
	FromName string `yaml:"from_name"`
	FromEmail string `yaml:"from_email"`
	Enabled  bool   `yaml:"enabled"`
}

// NewEmailService creates a new email service.
// If Host is empty, attempts SMTP auto-detection per PART 18.
// configFilePath is used to save detected SMTP settings per spec step 4.
func NewEmailService(cfg EmailConfig, fqdn, configFilePath string) *EmailService {
	svc := &EmailService{
		host:     cfg.Host,
		port:     cfg.Port,
		username: cfg.Username,
		password: cfg.Password,
		fromName: cfg.FromName,
		from:     cfg.FromEmail,
		tls:      cfg.TLS,
		enabled:  cfg.Enabled,
	}

	// Apply env var overrides (PART 18: SMTP_* vars override config)
	if v := os.Getenv("SMTP_HOST"); v != "" {
		svc.host = v
	}
	if v := os.Getenv("SMTP_PORT"); v != "" {
		fmt.Sscanf(v, "%d", &svc.port)
	}
	if v := os.Getenv("SMTP_USERNAME"); v != "" {
		svc.username = v
	}
	if v := os.Getenv("SMTP_PASSWORD"); v != "" {
		svc.password = v
	}
	if v := os.Getenv("SMTP_TLS"); v != "" {
		svc.tls = v
	}
	if v := os.Getenv("SMTP_FROM_NAME"); v != "" {
		svc.fromName = v
	}
	if v := os.Getenv("SMTP_FROM_EMAIL"); v != "" {
		svc.from = v
	}

	// Default port
	if svc.port == 0 {
		svc.port = 587
	}

	// Default from
	if svc.from == "" && fqdn != "" {
		svc.from = "no-reply@" + fqdn
	}
	if svc.fromName == "" {
		svc.fromName = "ytdl"
	}

	// Auto-detect SMTP if not configured (PART 18)
	if svc.host == "" {
		detected := autoDetectSMTP(fqdn)
		if detected != nil {
			svc.host = detected.host
			svc.port = detected.port
			svc.enabled = true
			log.Printf("SMTP auto-detected: %s:%d", svc.host, svc.port)

			// PART 18 step 4: Save detected host:port to server.yml
			if configFilePath != "" {
				saveSMTPToConfig(configFilePath, svc.host, svc.port)
			}
		} else {
			svc.enabled = false
			log.Println("SMTP not detected - email features disabled")
		}
	} else {
		// Connection test on every startup (PART 18)
		if testSMTPConnection(svc.host, svc.port) {
			svc.enabled = true
			log.Printf("SMTP connection verified: %s:%d", svc.host, svc.port)
		} else {
			svc.enabled = false
			log.Printf("WARNING: SMTP connection failed: %s:%d - email disabled", svc.host, svc.port)
		}
	}

	return svc
}

// saveSMTPToConfig appends detected SMTP settings to config file
// Per PART 18 step 4: "Save detected host:port to server.yml and enable email features"
func saveSMTPToConfig(configFilePath, host string, port int) {
	// Read existing config
	data, err := os.ReadFile(configFilePath)
	if err != nil {
		log.Printf("Warning: could not read config to save SMTP: %v", err)
		return
	}

	content := string(data)

	// Check if SMTP section already exists
	if strings.Contains(content, "smtp:") {
		return
	}

	// Append SMTP config (comments ABOVE settings per AI.md)
	// YAML comments ABOVE settings, NEVER inline
	smtpConfig := fmt.Sprintf(`
# Email notifications
# SMTP auto-detected on first run
server:
  notifications:
    email:
      smtp:
        # SMTP server hostname (auto-detected)
        host: "%s"
        # SMTP server port
        port: %d
        username: ""
        password: ""
        # TLS mode: auto, starttls, tls, none
        tls: auto
      from:
        # Sender display name
        name: "ytdl"
        # Sender email address
        email: ""
`, host, port)

	// Append to config file
	f, err := os.OpenFile(configFilePath, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		log.Printf("Warning: could not save SMTP config: %v", err)
		return
	}
	defer f.Close()
	f.WriteString(smtpConfig)
	log.Printf("SMTP settings saved to %s", configFilePath)
}

// smtpTarget holds a detected SMTP server
type smtpTarget struct {
	host string
	port int
}

// autoDetectSMTP tries SMTP hosts in EXACT priority order per PART 18:
// 1. 127.0.0.1 (loopback)
// 2. 172.17.0.1 (Docker bridge gateway)
// 3. {gateway_ip} (default gateway)
// 4. {fqdn} (detected FQDN)
// 5. {global_ipv4} (global IPv4 if available)
// 6. mail.{fqdn} (common mail subdomain)
// 7. smtp.{fqdn} (common SMTP subdomain)
// Ports per spec: 25, 465, 587
func autoDetectSMTP(fqdn string) *smtpTarget {
	var hosts []string

	// Priority 1: Loopback
	hosts = append(hosts, "127.0.0.1")

	// Priority 2: Docker bridge gateway
	hosts = append(hosts, "172.17.0.1")

	// Priority 3: Default gateway IP
	if gateway := detectGatewayIP(); gateway != "" {
		hosts = append(hosts, gateway)
	}

	// Priority 4: Detected FQDN
	if fqdn != "" {
		hosts = append(hosts, fqdn)
	}

	// Priority 5: Global IPv4
	if globalIP := detectGlobalIPv4(); globalIP != "" {
		hosts = append(hosts, globalIP)
	}

	// Priority 6: mail.{fqdn}
	if fqdn != "" {
		hosts = append(hosts, "mail."+fqdn)
	}

	// Priority 7: smtp.{fqdn}
	if fqdn != "" {
		hosts = append(hosts, "smtp."+fqdn)
	}

	// Ports per PART 18 spec: 25, 465, 587
	ports := []int{25, 465, 587}

	for _, host := range hosts {
		for _, port := range ports {
			if testSMTPConnection(host, port) {
				return &smtpTarget{host: host, port: port}
			}
		}
	}

	return nil
}

// detectGlobalIPv4 returns the global IPv4 address of this machine
func detectGlobalIPv4() string {
	conn, err := net.DialTimeout("udp", "8.8.8.8:53", 1*time.Second)
	if err != nil {
		return ""
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	ip := localAddr.IP.To4()
	if ip == nil {
		return ""
	}

	// Return the IP itself (not the gateway)
	return ip.String()
}

// testSMTPConnection attempts an SMTP handshake (EHLO)
func testSMTPConnection(host string, port int) bool {
	addr := fmt.Sprintf("%s:%d", host, port)

	// Quick TCP connection test with short timeout
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return false
	}

	// Try SMTP handshake
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close()
		return false
	}

	// EHLO handshake
	if err := client.Hello("ytdl"); err != nil {
		client.Close()
		return false
	}

	client.Quit()
	return true
}

// detectGatewayIP returns the default gateway IP
func detectGatewayIP() string {
	// Connect to a public IP to find local interface (doesn't actually send data)
	conn, err := net.DialTimeout("udp", "8.8.8.8:53", 1*time.Second)
	if err != nil {
		return ""
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	// Gateway is typically .1 on the same subnet
	ip := localAddr.IP.To4()
	if ip == nil {
		return ""
	}

	return fmt.Sprintf("%d.%d.%d.1", ip[0], ip[1], ip[2])
}

// IsEnabled returns whether email sending is configured and working
func (e *EmailService) IsEnabled() bool {
	return e.enabled && e.host != "" && e.from != ""
}

// GetHost returns the configured SMTP host
func (e *EmailService) GetHost() string {
	return e.host
}

// GetPort returns the configured SMTP port
func (e *EmailService) GetPort() int {
	return e.port
}

// SendDownloadComplete sends a notification when a download finishes
func (e *EmailService) SendDownloadComplete(to, title, filePath string) error {
	if !e.IsEnabled() {
		return nil
	}

	subject := fmt.Sprintf("Download Complete - %s", title)
	body := fmt.Sprintf(downloadCompleteTemplate, e.fromName, title, filePath, e.fromName)

	return e.send(to, subject, body)
}

// SendDownloadFailed sends a notification when a download fails
func (e *EmailService) SendDownloadFailed(to, title, errorMsg string) error {
	if !e.IsEnabled() {
		return nil
	}

	subject := fmt.Sprintf("Download Failed - %s", title)
	body := fmt.Sprintf(downloadFailedTemplate, e.fromName, title, errorMsg, e.fromName)

	return e.send(to, subject, body)
}

// SendTestEmail sends a test email to verify SMTP is working
func (e *EmailService) SendTestEmail(to string) error {
	if !e.IsEnabled() {
		return fmt.Errorf("email not configured")
	}

	subject := fmt.Sprintf("Test Email - %s", e.fromName)
	body := fmt.Sprintf(testEmailTemplate, e.fromName, e.host, e.port, e.fromName)

	return e.send(to, subject, body)
}

func (e *EmailService) send(to, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", e.host, e.port)

	fromHeader := e.from
	if e.fromName != "" {
		fromHeader = fmt.Sprintf("%s <%s>", e.fromName, e.from)
	}

	// Build message
	msg := strings.Join([]string{
		fmt.Sprintf("From: %s", fromHeader),
		fmt.Sprintf("To: %s", to),
		fmt.Sprintf("Subject: %s", subject),
		"MIME-Version: 1.0",
		"Content-Type: text/html; charset=UTF-8",
		"",
		body,
	}, "\r\n")

	// Auth (optional)
	var auth smtp.Auth
	if e.username != "" {
		auth = smtp.PlainAuth("", e.username, e.password, e.host)
	}

	if err := smtp.SendMail(addr, auth, e.from, []string{to}, []byte(msg)); err != nil {
		log.Printf("Email send failed to %s: %v", to, err)
		return fmt.Errorf("sending email: %w", err)
	}

	log.Printf("Email sent to %s: %s", to, subject)
	return nil
}

// Email templates (HTML)
var downloadCompleteTemplate = `<!DOCTYPE html>
<html>
<head><style>
  body { font-family: -apple-system, sans-serif; background: #1a1a2e; color: #eee; padding: 2rem; }
  .card { background: #16213e; padding: 1.5rem; border-radius: 8px; max-width: 500px; }
  h2 { color: #e94560; margin-top: 0; }
  .file { background: #0f3460; padding: 0.5rem; border-radius: 4px; font-family: monospace; word-break: break-all; }
  .footer { color: #666; font-size: 0.85rem; margin-top: 1rem; }
</style></head>
<body>
  <div class="card">
    <h2>Download Complete</h2>
    <p>This notification was sent by %s.</p>
    <p><strong>%s</strong></p>
    <p>Your download has completed successfully.</p>
    <p class="file">%s</p>
    <p class="footer">--%s</p>
  </div>
</body>
</html>`

var downloadFailedTemplate = `<!DOCTYPE html>
<html>
<head><style>
  body { font-family: -apple-system, sans-serif; background: #1a1a2e; color: #eee; padding: 2rem; }
  .card { background: #16213e; padding: 1.5rem; border-radius: 8px; max-width: 500px; }
  h2 { color: #e94560; margin-top: 0; }
  .error { background: #3e1621; padding: 0.5rem; border-radius: 4px; color: #f44336; }
  .footer { color: #666; font-size: 0.85rem; margin-top: 1rem; }
</style></head>
<body>
  <div class="card">
    <h2>Download Failed</h2>
    <p>This notification was sent by %s.</p>
    <p><strong>%s</strong></p>
    <p>Your download could not be completed.</p>
    <p class="error">%s</p>
    <p class="footer">-- %s</p>
  </div>
</body>
</html>`

var testEmailTemplate = `<!DOCTYPE html>
<html>
<head><style>
  body { font-family: -apple-system, sans-serif; background: #1a1a2e; color: #eee; padding: 2rem; }
  .card { background: #16213e; padding: 1.5rem; border-radius: 8px; max-width: 500px; }
  h2 { color: #e94560; margin-top: 0; }
  .footer { color: #666; font-size: 0.85rem; margin-top: 1rem; }
</style></head>
<body>
  <div class="card">
    <h2>Test Email</h2>
    <p>This is a test email from %s.</p>
    <p>If you received this, SMTP is configured correctly.</p>
    <p>SMTP Server: %s:%d</p>
    <p class="footer">-- %s</p>
  </div>
</body>
</html>`
