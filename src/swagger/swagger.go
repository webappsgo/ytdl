// Package swagger provides OpenAPI/Swagger spec generation and UI handler.
// See AI.md PART 14 for API documentation requirements.
// Swagger UI available at /openapi (root-level, not under /api/v1/).
package swagger

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Handler serves OpenAPI spec and Swagger UI
type Handler struct {
	version      string
	officialSite string
}

// NewHandler creates a new Swagger handler
func NewHandler(version, officialSite string) *Handler {
	return &Handler{
		version:      version,
		officialSite: officialSite,
	}
}

// HandleSpec serves the OpenAPI JSON spec at /openapi.json
func (h *Handler) HandleSpec(w http.ResponseWriter, r *http.Request) {
	spec := h.generateSpec()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(spec)
}

// HandleUI serves Swagger UI at /openapi
func (h *Handler) HandleUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, swaggerUITemplate, "/openapi.json")
}

func (h *Handler) generateSpec() map[string]interface{} {
	serverURL := "/"
	if h.officialSite != "" {
		serverURL = h.officialSite
	}

	return map[string]interface{}{
		"openapi": "3.0.3",
		"info": map[string]interface{}{
			"title":       "ytdl API",
			"description": "Self-hosted media downloader powered by yt-dlp",
			"version":     h.version,
			"license": map[string]string{
				"name": "MIT",
				"url":  "https://opensource.org/licenses/MIT",
			},
		},
		"servers": []map[string]string{
			{"url": serverURL, "description": "Current server"},
		},
		"paths": map[string]interface{}{
			"/healthz": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Health check",
					"operationId": "healthCheck",
					"tags":        []string{"Health"},
					"responses": map[string]interface{}{
						"200": map[string]string{"description": "Server is healthy"},
					},
				},
			},
			"/api/v1/version": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Get version info",
					"operationId": "getVersion",
					"tags":        []string{"Health"},
					"responses": map[string]interface{}{
						"200": map[string]string{"description": "Version information"},
					},
				},
			},
			"/api/v1/downloads": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "List downloads",
					"operationId": "listDownloads",
					"tags":        []string{"Downloads"},
					"parameters": []map[string]interface{}{
						{"name": "status", "in": "query", "schema": map[string]string{"type": "string"}, "description": "Filter by status"},
						{"name": "page", "in": "query", "schema": map[string]string{"type": "integer"}, "description": "Page number"},
						{"name": "per_page", "in": "query", "schema": map[string]string{"type": "integer"}, "description": "Items per page"},
					},
					"responses": map[string]interface{}{
						"200": map[string]string{"description": "List of downloads"},
					},
				},
				"post": map[string]interface{}{
					"summary":     "Submit download",
					"operationId": "submitDownload",
					"tags":        []string{"Downloads"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"required": []string{"url"},
									"properties": map[string]interface{}{
										"url":      map[string]string{"type": "string", "description": "URL to download"},
										"format":   map[string]string{"type": "string", "description": "Output format (mp4, mkv, mp3, etc.)"},
										"quality":  map[string]string{"type": "string", "description": "Quality (best, 1080, 720, etc.)"},
										"priority": map[string]string{"type": "string", "description": "Priority (high, normal, low)"},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"201": map[string]string{"description": "Download queued"},
						"400": map[string]string{"description": "Invalid request"},
					},
				},
			},
			"/api/v1/search": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Search sites",
					"operationId": "searchSites",
					"tags":        []string{"Search"},
					"parameters": []map[string]interface{}{
						{"name": "q", "in": "query", "required": true, "schema": map[string]string{"type": "string"}, "description": "Search query"},
						{"name": "site", "in": "query", "schema": map[string]string{"type": "string"}, "description": "Site to search (default: youtube)"},
						{"name": "limit", "in": "query", "schema": map[string]string{"type": "integer"}, "description": "Max results"},
					},
					"responses": map[string]interface{}{
						"200": map[string]string{"description": "Search results"},
					},
				},
			},
			"/api/v1/library": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Browse media library",
					"operationId": "browseLibrary",
					"tags":        []string{"Library"},
					"responses": map[string]interface{}{
						"200": map[string]string{"description": "Library items"},
					},
				},
			},
			"/api/v1/collections": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "List collections", "operationId": "listCollections", "tags": []string{"Collections"},
					"responses": map[string]interface{}{"200": map[string]string{"description": "Collections list"}},
				},
				"post": map[string]interface{}{
					"summary": "Create collection", "operationId": "createCollection", "tags": []string{"Collections"},
					"responses": map[string]interface{}{"201": map[string]string{"description": "Collection created"}},
				},
			},
			"/api/v1/presets": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "List presets", "operationId": "listPresets", "tags": []string{"Presets"},
					"responses": map[string]interface{}{"200": map[string]string{"description": "Presets list"}},
				},
			},
			"/api/v1/watch-rules": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "List watch rules", "operationId": "listWatchRules", "tags": []string{"Watch Rules"},
					"responses": map[string]interface{}{"200": map[string]string{"description": "Watch rules list"}},
				},
			},
			"/api/v1/analytics": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Get analytics", "operationId": "getAnalytics", "tags": []string{"Analytics"},
					"responses": map[string]interface{}{"200": map[string]string{"description": "Analytics data"}},
				},
			},
			"/api/v1/feed/rss": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "RSS/podcast feed", "operationId": "rssFeed", "tags": []string{"Feed"},
					"responses": map[string]interface{}{"200": map[string]string{"description": "RSS XML feed"}},
				},
			},
		},
	}
}

var swaggerUITemplate = `<!DOCTYPE html>
<html lang="en" data-theme="auto">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>ytdl - API Documentation</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
  <style>
    body { margin: 0; background: #1a1a2e; }
    .swagger-ui .topbar { display: none; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    SwaggerUIBundle({
      url: "%s",
      dom_id: '#swagger-ui',
      presets: [SwaggerUIBundle.presets.apis, SwaggerUIBundle.SwaggerUIStandalonePreset],
      layout: "BaseLayout",
      deepLinking: true,
      defaultModelsExpandDepth: -1
    });
  </script>
</body>
</html>`
