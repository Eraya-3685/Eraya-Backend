package order

import (
	"eraya/user"
	middleware "eraya/rest/middlewares"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router, h *Handler, jwtSecret string, userSvc user.Service) {
	r.Route("/cart", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(jwtSecret, userSvc))
		r.Post("/", h.AddToCart)
		r.Get("/", h.GetCart)
	})

	r.Route("/orders", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(jwtSecret, userSvc))
		r.Post("/checkout", h.Checkout)
		r.Post("/bkash/init", h.InitBKashPayment)
		r.Get("/", h.GetMyOrders)
		r.Get("/{id}", h.GetOrder)
	})

	r.Get("/orders/bkash/callback", h.BKashCallback)
	r.Route("/admin/orders", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(jwtSecret, userSvc), middleware.PermissionMiddleware("orders"))
		r.Get("/ws", h.AdminWebSocket)
		r.Get("/", h.AdminGetOrders)
		r.Get("/stats", h.AdminGetStats)
		r.Post("/{id}/confirm", h.AdminConfirmOrder)
		r.Put("/{id}/status", h.AdminUpdateStatus)
		r.Post("/request-delete-otp", h.AdminRequestDeleteOTP)
		r.Delete("/{id}", h.AdminDeleteOrder)
	})
}
