package websocket

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/coder/websocket"
	"github.com/hpcsc/stash/internal/domain"
)

type Hub struct {
	mu      sync.RWMutex
	clients map[domain.UserID]map[*websocket.Conn]struct{}
	logger  *slog.Logger
}

func NewHub(logger *slog.Logger) *Hub {
	return &Hub{
		clients: make(map[domain.UserID]map[*websocket.Conn]struct{}),
		logger:  logger,
	}
}

func (h *Hub) Register(user domain.UserID, conn *websocket.Conn) {
	h.mu.Lock()
	if h.clients[user] == nil {
		h.clients[user] = make(map[*websocket.Conn]struct{})
	}
	h.clients[user][conn] = struct{}{}
	h.mu.Unlock()
	h.logger.Info("websocket client connected", "user", user, "total", len(h.clients[user]))
}

func (h *Hub) Unregister(user domain.UserID, conn *websocket.Conn) {
	h.mu.Lock()
	delete(h.clients[user], conn)
	count := len(h.clients[user])
	if count == 0 {
		delete(h.clients, user)
	}
	h.mu.Unlock()
	h.logger.Info("websocket client disconnected", "user", user, "total", count)
}

func (h *Hub) Broadcast(ctx context.Context, user domain.UserID, msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		h.logger.Error("failed to marshal websocket message", "error", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for conn := range h.clients[user] {
		if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
			h.logger.Warn("failed to write to websocket client", "error", err)
		}
	}
}

func (h *Hub) Shutdown(ctx context.Context) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, conns := range h.clients {
		for conn := range conns {
			conn.Close(websocket.StatusGoingAway, "server shutting down")
		}
	}
	h.clients = make(map[domain.UserID]map[*websocket.Conn]struct{})
	h.logger.Info("websocket hub shut down")
}

type Message struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}
