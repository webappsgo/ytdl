// Package server - Audit logging for security events.
// See AI.md PART 11: structured audit log (who/what/when).
// JSON format to {log_dir}/audit.log.
package server

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// AuditLogger writes structured security events to audit.log
type AuditLogger struct {
	file *os.File
	mu   sync.Mutex
}

// AuditEvent represents a single audit log entry
type AuditEvent struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Event     string `json:"event"`
	User      string `json:"user,omitempty"`
	IP        string `json:"ip,omitempty"`
	Resource  string `json:"resource,omitempty"`
	Action    string `json:"action,omitempty"`
	Result    string `json:"result"`
	Detail    string `json:"detail,omitempty"`
}

// NewAuditLogger creates a new audit logger writing to the specified path
func NewAuditLogger(logPath string) (*AuditLogger, error) {
	if err := os.MkdirAll(logPath, 0700); err != nil {
		return nil, fmt.Errorf("creating audit log directory: %w", err)
	}

	filePath := logPath + "/audit.log"
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("opening audit log: %w", err)
	}

	return &AuditLogger{file: f}, nil
}

// Close closes the audit log file
func (a *AuditLogger) Close() error {
	if a.file != nil {
		return a.file.Close()
	}
	return nil
}

// Log writes an audit event
func (a *AuditLogger) Log(event AuditEvent) {
	if a == nil || a.file == nil {
		return
	}

	event.Timestamp = time.Now().UTC().Format(time.RFC3339)
	if event.Level == "" {
		event.Level = "info"
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	a.file.Write(data)
	a.file.Write([]byte("\n"))
}

// LogLoginAttempt records a login attempt
func (a *AuditLogger) LogLoginAttempt(username, ip, result string) {
	a.Log(AuditEvent{
		Event:    "auth.login",
		User:     username,
		IP:       ip,
		Action:   "login",
		Result:   result,
	})
}

// LogLoginSuccess records a successful login
func (a *AuditLogger) LogLoginSuccess(username, ip string) {
	a.LogLoginAttempt(username, ip, "success")
}

// LogLoginFailure records a failed login
func (a *AuditLogger) LogLoginFailure(username, ip string) {
	a.LogLoginAttempt(username, ip, "failure")
}

// LogAdminAction records an admin panel action
func (a *AuditLogger) LogAdminAction(username, ip, resource, action, detail string) {
	a.Log(AuditEvent{
		Event:    "admin.action",
		User:     username,
		IP:       ip,
		Resource: resource,
		Action:   action,
		Result:   "success",
		Detail:   detail,
	})
}

// LogConfigChange records a configuration change
func (a *AuditLogger) LogConfigChange(username, ip, setting, oldVal, newVal string) {
	a.Log(AuditEvent{
		Event:    "config.change",
		User:     username,
		IP:       ip,
		Resource: setting,
		Action:   "update",
		Result:   "success",
		Detail:   fmt.Sprintf("%s -> %s", oldVal, newVal),
	})
}

// LogSecurityEvent records a security-related event
func (a *AuditLogger) LogSecurityEvent(event, ip, detail string) {
	a.Log(AuditEvent{
		Level:  "warn",
		Event:  "security." + event,
		IP:     ip,
		Result: "blocked",
		Detail: detail,
	})
}

// LogRateLimitHit records a rate limit event
func (a *AuditLogger) LogRateLimitHit(ip, endpoint string) {
	a.Log(AuditEvent{
		Level:    "warn",
		Event:    "security.rate_limit",
		IP:       ip,
		Resource: endpoint,
		Action:   "request",
		Result:   "blocked",
	})
}

// ReopenLogFile reopens the audit log file (for log rotation via SIGUSR1)
func (a *AuditLogger) ReopenLogFile() error {
	if a == nil || a.file == nil {
		return nil
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	name := a.file.Name()
	a.file.Close()

	f, err := os.OpenFile(name, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	a.file = f
	return nil
}
