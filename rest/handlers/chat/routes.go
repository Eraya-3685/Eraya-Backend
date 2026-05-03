package chat

import (
	middleware "eraya/rest/middlewares"
	"eraya/user"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router, h *Handler, jwtSecret string, userSvc user.Service) {
	r.Route("/chat", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(jwtSecret, userSvc))
		r.Get("/conversations", h.GetConversations)
		r.Get("/conversation/{withID}", h.GetConversation)
		r.Delete("/conversation/{id}", h.DeleteConversation)
		r.Post("/conversation/{id}/read", h.MarkAsRead)
		r.Post("/messages/bulk-delete", h.BulkDeleteMessages)
		r.Get("/users/search", h.SearchUsers)
		r.Get("/ws", h.HandleConnections)
		r.Get("/unread-count", h.GetUnreadCount)
	})
}
