package user

import (
	middleware "eraya/rest/middlewares"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router, h *Handler, jwtSecret string) {
	r.Route("/users", func(r chi.Router) {
		r.Post("/signup", h.Signup)
		r.Post("/verify-signup", h.VerifySignup)
		r.Post("/resend-activation", h.ResendActivationOTP)
		r.Post("/login", h.Login)
		r.Post("/social-login", h.SocialLogin)
		r.Post("/forgot-password", h.ForgotPassword)
		r.Post("/reset-password", h.ResetPassword)

		// Protected routes
		r.With(middleware.AuthMiddleware(jwtSecret, h.svc)).Get("/profile", h.GetProfile)
		r.With(middleware.AuthMiddleware(jwtSecret, h.svc)).Patch("/profile", h.UpdateProfile)
		r.With(middleware.AuthMiddleware(jwtSecret, h.svc)).Patch("/avatar", h.UploadAvatar)
		r.With(middleware.AuthMiddleware(jwtSecret, h.svc)).Post("/otp/request", h.RequestOTP)
		r.With(middleware.AuthMiddleware(jwtSecret, h.svc)).Post("/otp/verify", h.VerifyOTPOnly)
		r.With(middleware.AuthMiddleware(jwtSecret, h.svc)).Patch("/secure-update", h.SecureUpdate)

		// Admin / Moderators with 'users' permission
		r.With(
			middleware.AuthMiddleware(jwtSecret, h.svc),
			middleware.PermissionMiddleware("users"),
		).Group(func(r chi.Router) {
			r.Get("/", h.ListUsers)
			r.Get("/{id}", h.GetUserByID)
		})

		// Admin only
		r.With(
			middleware.AuthMiddleware(jwtSecret, h.svc),
			middleware.AdminMiddleware(),
		).Group(func(r chi.Router) {
			r.Patch("/{id}/role", h.UpdateUserRole)
			r.Post("/bulk-role", h.BulkUpdateUserRole)
			r.Delete("/{id}", h.DeleteUser)
		})
	})
}
