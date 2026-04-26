// Package handler contains HTTP request handlers.
// See AI.md PART 13 for health and versioning specifications.
package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"
)

// HealthHandler handles health check and version endpoints
type HealthHandler struct {
	Version   string
	CommitID  string
	BuildDate string
	DB        *sql.DB
	StartTime time.Time
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(version, commitID, buildDate string) *HealthHandler {
	return &HealthHandler{
		Version:   version,
		CommitID:  commitID,
		BuildDate: buildDate,
		StartTime: time.Now(),
	}
}

// SetDB sets the database reference for health checks
func (h *HealthHandler) SetDB(db *sql.DB) {
	h.DB = db
}

// ExtendedHealthResponse is the full health check response per PART 13
type ExtendedHealthResponse struct {
	Status    string                 `json:"status"`
	Version   string                 `json:"version"`
	GoVersion string                 `json:"go_version"`
	Build     BuildInfo              `json:"build"`
	Features  FeatureFlags           `json:"features"`
	Checks    HealthChecks           `json:"checks"`
	Uptime    string                 `json:"uptime"`
}

// BuildInfo holds build metadata
type BuildInfo struct {
	CommitID  string `json:"commit_id"`
	BuildDate string `json:"build_date"`
}

// FeatureFlags indicates which features are enabled
type FeatureFlags struct {
	MultiUser     bool `json:"multi_user"`
	Organizations bool `json:"organizations"`
	Tor           bool `json:"tor"`
	GeoIP         bool `json:"geoip"`
	Metrics       bool `json:"metrics"`
}

// HealthChecks holds individual component health statuses
type HealthChecks struct {
	Database  string `json:"database"`
	Scheduler string `json:"scheduler"`
}

// VersionResponse is the JSON response for version endpoint
type VersionResponse struct {
	Version   string `json:"version"`
	CommitID  string `json:"commit_id"`
	BuildDate string `json:"build_date"`
}

// HandleHealthCheck handles GET /healthz and GET /api/v1/healthz
// Returns status only in basic mode, extended info with ?detail=true
// NEVER exposes sensitive data (tokens, credentials, paths, internal IPs)
func (h *HealthHandler) HandleHealthCheck(w http.ResponseWriter, r *http.Request) {
	accept := r.Header.Get("Accept")
	detail := r.URL.Query().Get("detail")

	// Extended health check
	if detail == "true" || detail == "1" {
		h.handleExtendedHealth(w, r)
		return
	}

	// JSON response for API clients
	if isJSONRequest(accept) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		return
	}

	// Plain text for browsers/curl
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "ok")
}

func (h *HealthHandler) handleExtendedHealth(w http.ResponseWriter, r *http.Request) {
	dbStatus := "ok"
	if h.DB != nil {
		if err := h.DB.Ping(); err != nil {
			dbStatus = "error"
		}
	}

	uptime := time.Since(h.StartTime).Round(time.Second).String()

	resp := ExtendedHealthResponse{
		Status:    "ok",
		Version:   h.Version,
		GoVersion: runtime.Version(),
		Build: BuildInfo{
			CommitID:  h.CommitID,
			BuildDate: h.BuildDate,
		},
		Features: FeatureFlags{
			MultiUser:     false,
			Organizations: false,
			Tor:           false,
			GeoIP:         false,
			Metrics:       true,
		},
		Checks: HealthChecks{
			Database:  dbStatus,
			Scheduler: "ok",
		},
		Uptime: uptime,
	}

	if dbStatus == "error" {
		resp.Status = "degraded"
	}

	w.Header().Set("Content-Type", "application/json")
	statusCode := http.StatusOK
	if resp.Status != "ok" {
		statusCode = http.StatusServiceUnavailable
	}
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(resp)
}

// HandleVersion handles GET /api/v1/version
func (h *HealthHandler) HandleVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(VersionResponse{
		Version:   h.Version,
		CommitID:  h.CommitID,
		BuildDate: h.BuildDate,
	})
}

func isJSONRequest(accept string) bool {
	return strings.Contains(accept, "application/json") ||
		strings.Contains(accept, "text/json")
}
