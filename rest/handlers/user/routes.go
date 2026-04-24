package user

import (
	middleware "eraya/rest/middlewares"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router, h *Handler, jwtSecret string) {
	r.Route("/users", func(r chi.Router) {
		r.Post("/signup", h.Signup)
		r.Post("/login", h.Login)

		// Protected routes
		r.With(middleware.AuthMiddleware(jwtSecret)).Get("/profile", h.GetProfile)
		r.With(middleware.AuthMiddleware(jwtSecret)).Patch("/profile", h.UpdateProfile)
		r.With(middleware.AuthMiddleware(jwtSecret)).Patch("/avatar", h.UploadAvatar)

		// Admin only
		r.With(
			middleware.AuthMiddleware(jwtSecret),
			middleware.AdminMiddleware(),
		).Patch("/{id}/role", h.UpdateUserRole)
	})
}
