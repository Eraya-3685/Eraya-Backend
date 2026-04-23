package product

import (
	"encoding/json"
	"eraya/domain"
	"eraya/product"
	middleware "eraya/rest/middlewares"
	"eraya/util"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	middlewares *middleware.Middlewares
	svc         product.Service
}

func NewHandler(middlewares *middleware.Middlewares, svc product.Service) *Handler {
	return &Handler{
		middlewares: middlewares,
		svc:         svc,
	}
}

// CreateProduct godoc
// @Summary Create a new product
// @Description Add a new product to the catalog (admin only).
// @Tags products
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param product body domain.Product true "Product Details"
// @Success 201 {object} domain.Product
// @Failure 403 {string} string "Forbidden"
// @Router /products [post]
func (h *Handler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var p domain.Product
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Simple slug generation
	if p.Slug == "" {
		p.Slug = strings.ToLower(strings.ReplaceAll(p.Name, " ", "-"))
	}
	p.IsActive = true

	created, err := h.svc.CreateProduct(&p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

// ListProducts godoc
// @Summary List all products
// @Description Retrieve a paginated list of active products.
// @Tags products
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Success 200 {object} util.PaginatedData
// @Router /products [get]
func (h *Handler) ListProducts(w http.ResponseWriter, r *http.Request) {
	pageAsStr := r.URL.Query().Get("page")
	limitAsStr := r.URL.Query().Get("limit")

	page, _ := strconv.ParseInt(pageAsStr, 10, 64)
	if page <= 0 {
		page = 1
	}

	limit, _ := strconv.ParseInt(limitAsStr, 10, 64)
	if limit <= 0 {
		limit = 10
	}

	products, count, err := h.svc.GetProducts(page, limit)
	if err != nil {
		util.SendError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	util.SendPage(w, products, page, limit, count)
}

// GetProduct godoc
// @Summary Get a single product
// @Description Retrieve details of a product by its slug.
// @Tags products
// @Produce json
// @Param slug path string true "Product Slug"
// @Success 200 {object} domain.Product
// @Failure 404 {string} string "Not Found"
// @Router /products/{slug} [get]
func (h *Handler) GetProduct(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	product, err := h.svc.GetProductBySlug(slug)
	if err != nil {
		http.Error(w, "product not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(product)
}
