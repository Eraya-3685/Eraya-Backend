package product

import (
	middleware "eraya/rest/middlewares"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router, h *Handler, jwtSecret string) {
	r.Route("/products", func(r chi.Router) {
		r.Get("/", h.ListProducts)
		r.Get("/{slug}", h.GetProduct)

		// Admin only
		r.With(
			middleware.AuthMiddleware(jwtSecret),
			middleware.AdminMiddleware(),
		).Post("/", h.CreateProduct)
	})
}
