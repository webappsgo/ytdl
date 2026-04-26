// Package handler - Download API endpoints.
// See AI.md PART 14 for API structure specifications.
package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/casapps/ytdl/src/server/service"
	"github.com/casapps/ytdl/src/server/store"
	"github.com/go-chi/chi/v5"
)

// DownloadHandler handles download-related HTTP requests
type DownloadHandler struct {
	store *store.Store
	queue *service.DownloadQueue
}

// NewDownloadHandler creates a new download handler
func NewDownloadHandler(st *store.Store, queue *service.DownloadQueue) *DownloadHandler {
	return &DownloadHandler{
		store: st,
		queue: queue,
	}
}

// SubmitRequest is the JSON body for submitting a download
type SubmitRequest struct {
	URL      string `json:"url"`
	Format   string `json:"format"`
	Quality  string `json:"quality"`
	Bitrate  string `json:"bitrate"`
	Priority string `json:"priority"`
	PresetID *int64 `json:"preset_id,omitempty"`
}

// BatchSubmitRequest is the JSON body for batch URL submission
type BatchSubmitRequest struct {
	URLs     []string `json:"urls"`
	Format   string   `json:"format"`
	Quality  string   `json:"quality"`
	Bitrate  string   `json:"bitrate"`
	Priority string   `json:"priority"`
	PresetID *int64   `json:"preset_id,omitempty"`
}

// APIResponse is the standard JSON response wrapper
type APIResponse struct {
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Code    string      `json:"code,omitempty"`
	Total   int         `json:"total,omitempty"`
	Page    int         `json:"page,omitempty"`
	PerPage int         `json:"per_page,omitempty"`
}

// HandleSubmitDownload handles POST /api/v1/downloads
func (h *DownloadHandler) HandleSubmitDownload(w http.ResponseWriter, r *http.Request) {
	var req SubmitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Error: "Invalid request body", Code: "INVALID_REQUEST"})
		return
	}

	// Validate URL
	url := strings.TrimSpace(req.URL)
	if url == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Error: "URL is required", Code: "MISSING_URL"})
		return
	}

	// Apply defaults
	if req.Format == "" {
		req.Format = "mp4"
	}
	if req.Quality == "" {
		req.Quality = "1080"
	}
	if req.Bitrate == "" {
		req.Bitrate = "320k"
	}
	if req.Priority == "" {
		req.Priority = store.PriorityNormal
	}

	// Validate priority
	if req.Priority != store.PriorityHigh && req.Priority != store.PriorityNormal && req.Priority != store.PriorityLow {
		writeJSON(w, http.StatusBadRequest, APIResponse{Error: "Invalid priority: must be high, normal, or low", Code: "INVALID_PRIORITY"})
		return
	}

	id, err := h.queue.Submit(url, req.Format, req.Quality, req.Bitrate, req.Priority, req.PresetID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Error: "Failed to submit download", Code: "SUBMIT_FAILED"})
		return
	}

	download, _ := h.store.GetDownloadByID(id)
	writeJSON(w, http.StatusCreated, APIResponse{Data: download})
}

// HandleBatchSubmit handles POST /api/v1/downloads/batch
func (h *DownloadHandler) HandleBatchSubmit(w http.ResponseWriter, r *http.Request) {
	var req BatchSubmitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Error: "Invalid request body", Code: "INVALID_REQUEST"})
		return
	}

	if len(req.URLs) == 0 {
		writeJSON(w, http.StatusBadRequest, APIResponse{Error: "At least one URL is required", Code: "MISSING_URLS"})
		return
	}

	// Apply defaults
	if req.Format == "" {
		req.Format = "mp4"
	}
	if req.Quality == "" {
		req.Quality = "1080"
	}
	if req.Bitrate == "" {
		req.Bitrate = "320k"
	}
	if req.Priority == "" {
		req.Priority = store.PriorityNormal
	}

	var ids []int64
	var errors []string
	for _, url := range req.URLs {
		url = strings.TrimSpace(url)
		if url == "" {
			continue
		}
		id, err := h.queue.Submit(url, req.Format, req.Quality, req.Bitrate, req.Priority, req.PresetID)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", url, err))
			continue
		}
		ids = append(ids, id)
	}

	result := map[string]interface{}{
		"submitted": ids,
		"count":     len(ids),
	}
	if len(errors) > 0 {
		result["errors"] = errors
	}

	writeJSON(w, http.StatusCreated, APIResponse{Data: result})
}

// HandleListDownloads handles GET /api/v1/downloads
func (h *DownloadHandler) HandleListDownloads(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))

	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	offset := (page - 1) * perPage

	downloads, total, err := h.store.ListDownloads(status, perPage, offset)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Error: "Failed to list downloads", Code: "LIST_FAILED"})
		return
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Data:    downloads,
		Total:   total,
		Page:    page,
		PerPage: perPage,
	})
}

// HandleGetDownload handles GET /api/v1/downloads/{id}
func (h *DownloadHandler) HandleGetDownload(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Error: "Invalid download ID", Code: "INVALID_ID"})
		return
	}

	download, err := h.store.GetDownloadByID(id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Error: "Failed to get download", Code: "GET_FAILED"})
		return
	}
	if download == nil {
		writeJSON(w, http.StatusNotFound, APIResponse{Error: "Download not found", Code: "NOT_FOUND"})
		return
	}

	writeJSON(w, http.StatusOK, APIResponse{Data: download})
}

// HandleCancelDownload handles POST /api/v1/downloads/{id}/cancel
func (h *DownloadHandler) HandleCancelDownload(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Error: "Invalid download ID", Code: "INVALID_ID"})
		return
	}

	if err := h.queue.CancelDownload(id); err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Error: "Failed to cancel download", Code: "CANCEL_FAILED"})
		return
	}

	writeJSON(w, http.StatusOK, APIResponse{Data: map[string]string{"status": "cancelled"}})
}

// HandlePauseDownload handles POST /api/v1/downloads/{id}/pause
func (h *DownloadHandler) HandlePauseDownload(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Error: "Invalid download ID", Code: "INVALID_ID"})
		return
	}

	if err := h.queue.PauseDownload(id); err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Error: "Failed to pause download", Code: "PAUSE_FAILED"})
		return
	}

	writeJSON(w, http.StatusOK, APIResponse{Data: map[string]string{"status": "paused"}})
}

// HandleResumeDownload handles POST /api/v1/downloads/{id}/resume
func (h *DownloadHandler) HandleResumeDownload(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Error: "Invalid download ID", Code: "INVALID_ID"})
		return
	}

	if err := h.queue.ResumeDownload(id); err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Error: "Failed to resume download", Code: "RESUME_FAILED"})
		return
	}

	writeJSON(w, http.StatusOK, APIResponse{Data: map[string]string{"status": "queued"}})
}

// HandleRetryDownload handles POST /api/v1/downloads/{id}/retry
func (h *DownloadHandler) HandleRetryDownload(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Error: "Invalid download ID", Code: "INVALID_ID"})
		return
	}

	if err := h.queue.RetryDownload(id); err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Error: "Failed to retry download", Code: "RETRY_FAILED"})
		return
	}

	writeJSON(w, http.StatusOK, APIResponse{Data: map[string]string{"status": "queued"}})
}

// HandleDeleteDownload handles DELETE /api/v1/downloads/{id}
func (h *DownloadHandler) HandleDeleteDownload(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Error: "Invalid download ID", Code: "INVALID_ID"})
		return
	}

	// Get download to find file path
	download, err := h.store.GetDownloadByID(id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Error: "Failed to get download", Code: "GET_FAILED"})
		return
	}
	if download == nil {
		writeJSON(w, http.StatusNotFound, APIResponse{Error: "Download not found", Code: "NOT_FOUND"})
		return
	}

	// Delete file if exists
	if download.FilePath != "" {
		os.Remove(download.FilePath)
		// Also remove thumbnail
		if download.ThumbnailPath != "" {
			os.Remove(download.ThumbnailPath)
		}
	}

	// Delete from database
	if err := h.store.DeleteDownload(id); err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Error: "Failed to delete download", Code: "DELETE_FAILED"})
		return
	}

	writeJSON(w, http.StatusOK, APIResponse{Data: map[string]string{"status": "deleted"}})
}

// HandleDownloadFile handles GET /api/v1/downloads/{id}/file
func (h *DownloadHandler) HandleDownloadFile(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Error: "Invalid download ID", Code: "INVALID_ID"})
		return
	}

	download, err := h.store.GetDownloadByID(id)
	if err != nil || download == nil {
		writeJSON(w, http.StatusNotFound, APIResponse{Error: "Download not found", Code: "NOT_FOUND"})
		return
	}

	if download.Status != store.StatusCompleted || download.FilePath == "" {
		writeJSON(w, http.StatusNotFound, APIResponse{Error: "File not available", Code: "FILE_NOT_AVAILABLE"})
		return
	}

	// Verify file exists
	if _, err := os.Stat(download.FilePath); os.IsNotExist(err) {
		writeJSON(w, http.StatusGone, APIResponse{Error: "File has been deleted", Code: "FILE_DELETED"})
		return
	}

	// Set content disposition for download
	filename := filepath.Base(download.FilePath)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))

	http.ServeFile(w, r, download.FilePath)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
