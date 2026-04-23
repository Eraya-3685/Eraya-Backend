package product

import (
	"encoding/json"
	"eraya/domain"
	"eraya/infra/storage"
	"eraya/product"
	"eraya/util"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc     product.Service
	storage *storage.StorageService
}

func NewHandler(svc product.Service, storage *storage.StorageService) *Handler {
	return &Handler{
		svc:     svc,
		storage: storage,
	}
}

// CreateProduct godoc
// @Summary Create a new product with images
// @Description Add a new product and upload images to Supabase (admin only).
// @Tags products
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param name formData string true "Product Name"
// @Param description formData string false "Product Description"
// @Param base_price formData number true "Base Price"
// @Param discount_price formData number false "Discount Price"
// @Param stock_count formData int true "Stock Count"
// @Param category_id formData int false "Category ID"
// @Param images formData file true "Product Images"
// @Success 201 {object} domain.Product
// @Router /products [post]
func (h *Handler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	// 10MB max memory
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "failed to parse multipart form", http.StatusBadRequest)
		return
	}

	// Extract product info from form
	name := r.FormValue("name")
	desc := r.FormValue("description")
	basePrice, _ := strconv.ParseFloat(r.FormValue("base_price"), 64)
	discountPrice, _ := strconv.ParseFloat(r.FormValue("discount_price"), 64)
	stockCount, _ := strconv.Atoi(r.FormValue("stock_count"))
	catID, _ := strconv.Atoi(r.FormValue("category_id"))

	p := &domain.Product{
		Name:       name,
		BasePrice:  basePrice,
		StockCount: stockCount,
		IsActive:   true,
	}
	if desc != "" {
		p.Description = &desc
	}
	if discountPrice > 0 {
		p.DiscountPrice = &discountPrice
	}
	if catID > 0 {
		p.CategoryID = &catID
	}

	// Slug generation
	p.Slug = strings.ToLower(strings.ReplaceAll(p.Name, " ", "-"))

	// Handle Image Uploads
	files := r.MultipartForm.File["images"]
	for i, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			continue
		}
		defer file.Close()

		// Upload to Supabase in 'products' folder
		url, err := h.storage.UploadFile("products", fileHeader.Filename, file, fileHeader.Header.Get("Content-Type"))
		if err != nil {
			fmt.Printf("Upload failed: %v\n", err)
			continue
		}

		p.Images = append(p.Images, domain.ProductImage{
			ImageURL:  url,
			IsPrimary: i == 0, // First image is primary
		})
	}

	created, err := h.svc.CreateProduct(p)
	if err != nil {
		// Cleanup uploaded images if DB fails
		for _, img := range p.Images {
			go h.storage.DeleteFile(img.ImageURL)
		}
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

// UpdateProduct godoc
// @Summary Update an existing product
// @Description Update product details (admin only).
// @Tags products
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Product ID"
// @Param product body domain.Product true "Updated Product Details"
// @Success 200 {string} string "OK"
// @Router /products/{id} [put]
func (h *Handler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var p domain.Product
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	p.ID = id

	if err := h.svc.UpdateProduct(&p); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Product updated successfully"))
}

// DeleteProduct godoc
// @Summary Delete a product
// @Description Remove a product and its images (admin only).
// @Tags products
// @Produce json
// @Security BearerAuth
// @Param id path int true "Product ID"
// @Success 200 {string} string "OK"
// @Router /products/{id} [delete]
func (h *Handler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

	// Fetch product first to get image URLs for cleanup
	p, err := h.svc.GetProductByID(id)
	if err != nil {
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	}

	if err := h.svc.DeleteProduct(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Async cleanup of images from Supabase
	for _, img := range p.Images {
		go h.storage.DeleteFile(img.ImageURL)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Product deleted successfully"))
}

