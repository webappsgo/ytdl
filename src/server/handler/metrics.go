// Package handler - Prometheus metrics endpoint.
// See AI.md PART 21 for metrics specifications.
// Exposes metrics at /metrics in Prometheus text format.
package handler

import (
	"fmt"
	"net/http"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/casapps/ytdl/src/server/store"
)

// MetricsCollector tracks application metrics
type MetricsCollector struct {
	store     *store.Store
	startTime time.Time

	// Counters (atomic for thread safety)
	httpRequestsTotal  atomic.Int64
	httpRequests2xx    atomic.Int64
	httpRequests4xx    atomic.Int64
	httpRequests5xx    atomic.Int64
	downloadsQueued    atomic.Int64
	downloadsCompleted atomic.Int64
	downloadsFailed    atomic.Int64
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(st *store.Store) *MetricsCollector {
	return &MetricsCollector{
		store:     st,
		startTime: time.Now(),
	}
}

// IncrementHTTPRequests increments the total HTTP request counter
func (m *MetricsCollector) IncrementHTTPRequests(statusCode int) {
	m.httpRequestsTotal.Add(1)
	switch {
	case statusCode >= 200 && statusCode < 300:
		m.httpRequests2xx.Add(1)
	case statusCode >= 400 && statusCode < 500:
		m.httpRequests4xx.Add(1)
	case statusCode >= 500:
		m.httpRequests5xx.Add(1)
	}
}

// IncrementDownloadQueued increments queued download counter
func (m *MetricsCollector) IncrementDownloadQueued() {
	m.downloadsQueued.Add(1)
}

// IncrementDownloadCompleted increments completed download counter
func (m *MetricsCollector) IncrementDownloadCompleted() {
	m.downloadsCompleted.Add(1)
}

// IncrementDownloadFailed increments failed download counter
func (m *MetricsCollector) IncrementDownloadFailed() {
	m.downloadsFailed.Add(1)
}

// MetricsMiddleware records HTTP request metrics
func (m *MetricsCollector) MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := &responseWriter{ResponseWriter: w, statusCode: 200}
		next.ServeHTTP(rw, r)
		m.IncrementHTTPRequests(rw.statusCode)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// HandleMetrics serves Prometheus metrics at /metrics
func (m *MetricsCollector) HandleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Get database stats
	var totalDownloads, completedDownloads, failedDownloads, queuedDownloads int
	var totalSizeBytes int64
	if m.store != nil {
		m.store.DB().QueryRow(`SELECT COUNT(*) FROM downloads`).Scan(&totalDownloads)
		m.store.DB().QueryRow(`SELECT COUNT(*) FROM downloads WHERE status = 'completed'`).Scan(&completedDownloads)
		m.store.DB().QueryRow(`SELECT COUNT(*) FROM downloads WHERE status = 'failed'`).Scan(&failedDownloads)
		m.store.DB().QueryRow(`SELECT COUNT(*) FROM downloads WHERE status = 'queued'`).Scan(&queuedDownloads)
		m.store.DB().QueryRow(`SELECT COALESCE(SUM(file_size), 0) FROM downloads WHERE status = 'completed'`).Scan(&totalSizeBytes)
	}

	uptime := time.Since(m.startTime).Seconds()

	// Application info
	writeMetric(w, "ytdl_info", "gauge", "Application information",
		fmt.Sprintf(`{version="dev",go_version="%s"} 1`, runtime.Version()))

	// Uptime
	writeMetric(w, "ytdl_uptime_seconds", "gauge", "Server uptime in seconds",
		fmt.Sprintf("%f", uptime))

	// HTTP requests
	writeMetric(w, "ytdl_http_requests_total", "counter", "Total HTTP requests",
		fmt.Sprintf("%d", m.httpRequestsTotal.Load()))
	writeMetric(w, "ytdl_http_requests_2xx_total", "counter", "HTTP 2xx responses",
		fmt.Sprintf("%d", m.httpRequests2xx.Load()))
	writeMetric(w, "ytdl_http_requests_4xx_total", "counter", "HTTP 4xx responses",
		fmt.Sprintf("%d", m.httpRequests4xx.Load()))
	writeMetric(w, "ytdl_http_requests_5xx_total", "counter", "HTTP 5xx responses",
		fmt.Sprintf("%d", m.httpRequests5xx.Load()))

	// Downloads
	writeMetric(w, "ytdl_downloads_total", "gauge", "Total downloads in database",
		fmt.Sprintf("%d", totalDownloads))
	writeMetric(w, "ytdl_downloads_completed_total", "gauge", "Completed downloads",
		fmt.Sprintf("%d", completedDownloads))
	writeMetric(w, "ytdl_downloads_failed_total", "gauge", "Failed downloads",
		fmt.Sprintf("%d", failedDownloads))
	writeMetric(w, "ytdl_downloads_queued", "gauge", "Currently queued downloads",
		fmt.Sprintf("%d", queuedDownloads))
	writeMetric(w, "ytdl_downloads_size_bytes_total", "gauge", "Total download size in bytes",
		fmt.Sprintf("%d", totalSizeBytes))

	// Go runtime
	writeMetric(w, "ytdl_go_goroutines", "gauge", "Number of goroutines",
		fmt.Sprintf("%d", runtime.NumGoroutine()))
	writeMetric(w, "ytdl_go_memstats_alloc_bytes", "gauge", "Bytes allocated and in use",
		fmt.Sprintf("%d", memStats.Alloc))
	writeMetric(w, "ytdl_go_memstats_sys_bytes", "gauge", "Bytes obtained from system",
		fmt.Sprintf("%d", memStats.Sys))
	writeMetric(w, "ytdl_go_memstats_gc_total", "counter", "Number of completed GC cycles",
		fmt.Sprintf("%d", memStats.NumGC))
}

func writeMetric(w http.ResponseWriter, name, metricType, help, value string) {
	fmt.Fprintf(w, "# HELP %s %s\n", name, help)
	fmt.Fprintf(w, "# TYPE %s %s\n", name, metricType)
	fmt.Fprintf(w, "%s %s\n", name, value)
}
