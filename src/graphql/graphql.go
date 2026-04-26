// Package graphql provides a GraphQL API endpoint.
// See AI.md PART 14 for GraphQL requirements.
// Available at /graphql (web) and /api/v1/graphql (API).
package graphql

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/casapps/ytdl/src/server/store"
)

// Handler serves GraphQL queries
type Handler struct {
	store *store.Store
}

// NewHandler creates a new GraphQL handler
func NewHandler(st *store.Store) *Handler {
	return &Handler{store: st}
}

// GraphQLRequest is the incoming query format
type GraphQLRequest struct {
	Query         string                 `json:"query"`
	OperationName string                 `json:"operationName"`
	Variables     map[string]interface{} `json:"variables"`
}

// HandleQuery handles POST /graphql and /api/v1/graphql
func (h *Handler) HandleQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		h.handleGraphiQL(w, r)
		return
	}

	var req GraphQLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeGraphQLError(w, "Invalid request body")
		return
	}

	result := h.executeQuery(req)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *Handler) executeQuery(req GraphQLRequest) map[string]interface{} {
	query := strings.TrimSpace(req.Query)

	// Simple query parser for common operations
	if strings.Contains(query, "downloads") {
		return h.queryDownloads(req.Variables)
	}
	if strings.Contains(query, "collections") {
		return h.queryCollections()
	}
	if strings.Contains(query, "presets") {
		return h.queryPresets()
	}
	if strings.Contains(query, "analytics") {
		return h.queryAnalytics()
	}
	if strings.Contains(query, "__schema") || strings.Contains(query, "IntrospectionQuery") {
		return h.introspection()
	}

	return map[string]interface{}{
		"data":   nil,
		"errors": []map[string]string{{"message": "Unsupported query"}},
	}
}

func (h *Handler) queryDownloads(variables map[string]interface{}) map[string]interface{} {
	status := ""
	limit := 20
	offset := 0

	if v, ok := variables["status"]; ok {
		status = fmt.Sprintf("%v", v)
	}
	if v, ok := variables["limit"]; ok {
		if n, ok := v.(float64); ok {
			limit = int(n)
		}
	}

	downloads, total, err := h.store.ListDownloads(status, limit, offset)
	if err != nil {
		return map[string]interface{}{
			"data":   nil,
			"errors": []map[string]string{{"message": err.Error()}},
		}
	}

	return map[string]interface{}{
		"data": map[string]interface{}{
			"downloads": map[string]interface{}{
				"items": downloads,
				"total": total,
			},
		},
	}
}

func (h *Handler) queryCollections() map[string]interface{} {
	rows, err := h.store.DB().Query(`SELECT id, name, type, created_at FROM collections ORDER BY name`)
	if err != nil {
		return map[string]interface{}{"data": nil, "errors": []map[string]string{{"message": err.Error()}}}
	}
	defer rows.Close()

	var items []map[string]interface{}
	for rows.Next() {
		var id int64
		var name, colType, createdAt string
		if rows.Scan(&id, &name, &colType, &createdAt) == nil {
			items = append(items, map[string]interface{}{
				"id": id, "name": name, "type": colType, "created_at": createdAt,
			})
		}
	}

	return map[string]interface{}{"data": map[string]interface{}{"collections": items}}
}

func (h *Handler) queryPresets() map[string]interface{} {
	rows, err := h.store.DB().Query(`SELECT id, name, format, quality, is_default FROM download_presets ORDER BY name`)
	if err != nil {
		return map[string]interface{}{"data": nil, "errors": []map[string]string{{"message": err.Error()}}}
	}
	defer rows.Close()

	var items []map[string]interface{}
	for rows.Next() {
		var id int64
		var name, format, quality string
		var isDefault bool
		if rows.Scan(&id, &name, &format, &quality, &isDefault) == nil {
			items = append(items, map[string]interface{}{
				"id": id, "name": name, "format": format, "quality": quality, "is_default": isDefault,
			})
		}
	}

	return map[string]interface{}{"data": map[string]interface{}{"presets": items}}
}

func (h *Handler) queryAnalytics() map[string]interface{} {
	var totalDownloads, completedCount int
	var totalSize int64

	h.store.DB().QueryRow(`SELECT COUNT(*) FROM downloads`).Scan(&totalDownloads)
	h.store.DB().QueryRow(`SELECT COUNT(*) FROM downloads WHERE status = 'completed'`).Scan(&completedCount)
	h.store.DB().QueryRow(`SELECT COALESCE(SUM(file_size), 0) FROM downloads WHERE status = 'completed'`).Scan(&totalSize)

	return map[string]interface{}{
		"data": map[string]interface{}{
			"analytics": map[string]interface{}{
				"total_downloads": totalDownloads,
				"completed":       completedCount,
				"total_size":      totalSize,
			},
		},
	}
}

func (h *Handler) introspection() map[string]interface{} {
	return map[string]interface{}{
		"data": map[string]interface{}{
			"__schema": map[string]interface{}{
				"queryType": map[string]string{"name": "Query"},
				"types": []map[string]interface{}{
					{"name": "Query", "kind": "OBJECT"},
					{"name": "Download", "kind": "OBJECT"},
					{"name": "Collection", "kind": "OBJECT"},
					{"name": "Preset", "kind": "OBJECT"},
					{"name": "Analytics", "kind": "OBJECT"},
				},
			},
		},
	}
}

func (h *Handler) handleGraphiQL(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, graphiQLHTML)
}

func writeGraphQLError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":   nil,
		"errors": []map[string]string{{"message": message}},
	})
}

var graphiQLHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>ytdl - GraphQL</title>
  <style>body { height: 100vh; margin: 0; overflow: hidden; }</style>
  <link rel="stylesheet" href="https://unpkg.com/graphiql@3/graphiql.min.css">
</head>
<body>
  <div id="graphiql" style="height:100vh"></div>
  <script src="https://unpkg.com/react@18/umd/react.production.min.js"></script>
  <script src="https://unpkg.com/react-dom@18/umd/react-dom.production.min.js"></script>
  <script src="https://unpkg.com/graphiql@3/graphiql.min.js"></script>
  <script>
    const fetcher = GraphiQL.createFetcher({ url: window.location.href });
    ReactDOM.createRoot(document.getElementById('graphiql'))
      .render(React.createElement(GraphiQL, { fetcher }));
  </script>
</body>
</html>`
