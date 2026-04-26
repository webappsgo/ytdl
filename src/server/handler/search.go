// Package handler - Search API endpoint.
// Searches any yt-dlp searchable site directly from the web UI.
package handler

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/casapps/ytdl/src/server/service"
)

// SearchHandler handles search requests
type SearchHandler struct {
	ytdlp *service.YTDLPService
}

// NewSearchHandler creates a new search handler
func NewSearchHandler(ytdlp *service.YTDLPService) *SearchHandler {
	return &SearchHandler{ytdlp: ytdlp}
}

// HandleSearch handles GET /api/v1/search?q=query&site=youtube&limit=10
func (h *SearchHandler) HandleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Error: "Search query is required", Code: "MISSING_QUERY"})
		return
	}

	site := r.URL.Query().Get("site")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 50 {
		limit = 10
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	results, err := h.ytdlp.SearchSite(ctx, query, site, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Error: "Search failed", Code: "SEARCH_FAILED"})
		return
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Data:  results,
		Total: len(results),
	})
}
