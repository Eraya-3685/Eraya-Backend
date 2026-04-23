package chat

import (
	middleware "eraya/rest/middlewares"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router, h *WebSocketHandler, jwtSecret string) {
	r.With(middleware.AuthMiddleware(jwtSecret)).Get("/ws/chat", h.HandleConnections)
}
