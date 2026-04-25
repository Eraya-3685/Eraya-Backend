package chat

import (
	middleware "eraya/rest/middlewares"
	"eraya/user"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router, h *WebSocketHandler, jwtSecret string, userSvc user.Service) {
	r.With(middleware.AuthMiddleware(jwtSecret, userSvc)).Get("/ws/chat", h.HandleConnections)
}
