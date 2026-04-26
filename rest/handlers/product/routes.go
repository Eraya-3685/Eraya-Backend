package product

import (
	"eraya/user"
	middleware "eraya/rest/middlewares"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router, h *Handler, jwtSecret string, userSvc user.Service) {
	r.Route("/products", func(r chi.Router) {
		r.Get("/", h.ListProducts)
		r.Get("/{slug}", h.GetProduct)

		r.With(
			middleware.AuthMiddleware(jwtSecret, userSvc),
			middleware.PermissionMiddleware("products"),
		).Group(func(r chi.Router) {
			r.Post("/", h.CreateProduct)
			r.Put("/{id}", h.UpdateProduct)
			r.Delete("/{id}", h.DeleteProduct)
			r.Post("/bulk-delete", h.BulkDeleteProducts)
		})
	})

	r.Route("/categories", func(r chi.Router) {
		r.Get("/", h.ListCategories)
		r.With(
			middleware.AuthMiddleware(jwtSecret, userSvc),
			middleware.PermissionMiddleware("categories"),
		).Group(func(r chi.Router) {
			r.Post("/", h.CreateCategory)
			r.Put("/{id}", h.UpdateCategory)
			r.Delete("/{id}", h.DeleteCategory)
			r.Post("/bulk-delete", h.BulkDeleteCategories)
		})
	})

	r.With(
		middleware.AuthMiddleware(jwtSecret, userSvc),
		middleware.PermissionMiddleware("products"),
	).Post("/upload", h.UploadFile)
}
