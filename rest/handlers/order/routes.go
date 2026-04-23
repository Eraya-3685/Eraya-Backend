package order

import (
	middleware "eraya/rest/middlewares"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router, h *Handler, jwtSecret string) {
	r.Route("/cart", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(jwtSecret))
		r.Post("/", h.AddToCart)
		r.Get("/", h.GetCart)
	})

	r.Route("/orders", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(jwtSecret))
		r.Post("/checkout", h.Checkout)
		r.Get("/", h.GetMyOrders)
	})

	r.Route("/admin/orders", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(jwtSecret), middleware.AdminMiddleware())
		r.Get("/", h.AdminGetOrders)
		r.Post("/{id}/confirm", h.AdminConfirmOrder)
		r.Delete("/{id}", h.AdminDeleteOrder)
	})
}
