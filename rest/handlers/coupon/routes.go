package coupon

import (
	"eraya/user"
	middleware "eraya/rest/middlewares"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router, h *Handler, jwtSecret string, userSvc user.Service) {
	r.Route("/coupons", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(jwtSecret, userSvc))
		r.Post("/apply", h.ApplyCoupon)
	})

	r.Route("/admin/coupons", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(jwtSecret, userSvc))
		r.Use(middleware.AdminMiddleware())

		r.Post("/", h.CreateCoupon)
		r.Get("/", h.ListCoupons)
		r.Delete("/{id}", h.DeleteCoupon)
	})
}
