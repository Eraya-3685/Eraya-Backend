package user

import (
	middleware "eraya/rest/middlewares"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router, h *Handler, jwtSecret string) {
	r.Route("/users", func(r chi.Router) {
		r.Post("/signup", h.Signup)
		r.Post("/login", h.Login)
		r.Post("/social-login", h.SocialLogin)
		r.Post("/forgot-password", h.ForgotPassword)
		r.Post("/reset-password", h.ResetPassword)

		// Protected routes
		r.With(middleware.AuthMiddleware(jwtSecret, h.svc)).Get("/profile", h.GetProfile)
		r.With(middleware.AuthMiddleware(jwtSecret, h.svc)).Patch("/profile", h.UpdateProfile)
		r.With(middleware.AuthMiddleware(jwtSecret, h.svc)).Patch("/avatar", h.UploadAvatar)
		r.With(middleware.AuthMiddleware(jwtSecret, h.svc)).Post("/otp/request", h.RequestOTP)
		r.With(middleware.AuthMiddleware(jwtSecret, h.svc)).Patch("/secure-update", h.SecureUpdate)

		// Admin only
		r.With(
			middleware.AuthMiddleware(jwtSecret, h.svc),
			middleware.AdminMiddleware(),
		).Group(func(r chi.Router) {
			r.Patch("/{id}/role", h.UpdateUserRole)
			r.Delete("/{id}", h.DeleteUser)
		})
	})
}
