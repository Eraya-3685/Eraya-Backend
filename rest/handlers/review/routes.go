package review

import (
	middleware "eraya/rest/middlewares"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router, h *Handler, jwtSecret string) {
	r.Route("/reviews", func(r chi.Router) {
		r.Get("/{productId}", h.GetProductReviews)
		r.With(middleware.AuthMiddleware(jwtSecret)).Post("/", h.CreateReview)
	})
}
