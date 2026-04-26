package wishlist

import (
	erayamiddleware "eraya/rest/middlewares"
	"eraya/user"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router, h *Handler, jwtSecret string, userSvc user.Service) {
	r.Group(func(r chi.Router) {
		r.Use(erayamiddleware.AuthMiddleware(jwtSecret, userSvc))

		r.Get("/wishlist", h.List)
		r.Post("/wishlist/{product_id}", h.Add)
		r.Delete("/wishlist/{product_id}", h.Remove)
		r.Delete("/wishlist", h.Clear)
	})
}
