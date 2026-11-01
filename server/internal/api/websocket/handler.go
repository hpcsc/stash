package websocket

import (
	"log/slog"
	"net/http"

	"github.com/coder/websocket"
	"github.com/hpcsc/stash/internal/api/route"
	"github.com/hpcsc/stash/internal/api/session"
	"github.com/hpcsc/stash/internal/domain"
)

var _ route.Routable = (*Handler)(nil)

type Handler struct {
	hub    *Hub
	issuer *session.Issuer
	logger *slog.Logger
}

func NewHandler(hub *Hub, issuer *session.Issuer, logger *slog.Logger) *Handler {
	return &Handler{
		hub:    hub,
		issuer: issuer,
		logger: logger,
	}
}

func (h *Handler) Routes() []*route.Route {
	return []*route.Route{
		route.Public("GET", "/ws", h.serve),
	}
}

func (h *Handler) serve(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	subject, err := h.issuer.Verify(token)
	user := domain.UserID(subject)
	if err != nil || !user.Valid() {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		h.logger.Warn("websocket upgrade failed", "error", err)
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	h.hub.Register(user, conn)
	defer h.hub.Unregister(user, conn)

	ctx := r.Context()
	for {
		_, _, err := conn.Read(ctx)
		if err != nil {
			break
		}
	}
}
