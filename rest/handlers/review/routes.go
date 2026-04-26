package review

import (
	"eraya/user"
	middleware "eraya/rest/middlewares"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router, h *Handler, jwtSecret string, userSvc user.Service) {
	r.Route("/reviews", func(r chi.Router) {
		r.Get("/{productId}", h.GetProductReviews)
		r.With(middleware.AuthMiddleware(jwtSecret, userSvc)).Post("/", h.CreateReview)
		r.With(
			middleware.AuthMiddleware(jwtSecret, userSvc),
			middleware.AdminMiddleware(),
		).Delete("/{id}", h.DeleteReview)
	})

	r.Route("/admin/reviews", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(jwtSecret, userSvc))
		r.Use(middleware.AdminMiddleware())

		r.Get("/", h.ListAllReviews)
		r.Post("/{id}/approve", h.ApproveReview)
	})
}
