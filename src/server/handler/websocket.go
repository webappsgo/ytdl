// Package handler - WebSocket endpoint for real-time download progress.
// Uses gorilla/websocket per spec.
package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/casapps/ytdl/src/server/service"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Allow connections from any origin (CORS handled by middleware)
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// WebSocketHub manages active WebSocket connections
type WebSocketHub struct {
	clients    map[*websocket.Conn]bool
	mu         sync.RWMutex
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	broadcast  chan []byte
}

// NewWebSocketHub creates a new WebSocket hub
func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[*websocket.Conn]bool),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
		broadcast:  make(chan []byte, 256),
	}
}

// Run starts the hub event loop
func (h *WebSocketHub) Run() {
	for {
		select {
		case conn := <-h.register:
			h.mu.Lock()
			h.clients[conn] = true
			h.mu.Unlock()

		case conn := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[conn]; ok {
				delete(h.clients, conn)
				conn.Close()
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for conn := range h.clients {
				if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
					h.mu.RUnlock()
					h.unregister <- conn
					h.mu.RLock()
				}
			}
			h.mu.RUnlock()
		}
	}
}

// BroadcastProgress sends a progress update to all connected clients
func (h *WebSocketHub) BroadcastProgress(update service.ProgressUpdate) {
	data, err := json.Marshal(map[string]interface{}{
		"type": "progress",
		"data": map[string]interface{}{
			"download_id": update.DownloadID,
			"status":      update.Status,
			"percent":     update.Percent,
			"speed":       update.Speed,
			"eta":         update.ETA,
			"file_size":   update.FileSize,
		},
	})
	if err != nil {
		return
	}

	select {
	case h.broadcast <- data:
	default:
		// Drop message if broadcast channel is full
	}
}

// BroadcastEvent sends a generic event to all connected clients
func (h *WebSocketHub) BroadcastEvent(eventType string, data interface{}) {
	msg, err := json.Marshal(map[string]interface{}{
		"type": eventType,
		"data": data,
	})
	if err != nil {
		return
	}

	select {
	case h.broadcast <- msg:
	default:
	}
}

// HandleWebSocket handles WebSocket upgrade requests
func (h *WebSocketHub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	h.register <- conn

	// Read loop (handle client messages and detect disconnect)
	go func() {
		defer func() {
			h.unregister <- conn
		}()

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
			// Client messages can be used for subscriptions or ping
		}
	}()
}

// ClientCount returns the number of connected WebSocket clients
func (h *WebSocketHub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
