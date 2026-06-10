package product

import (
	"context"
	"encoding/json"
	"eraya/domain"
	"eraya/infra/storage"
	"eraya/product"
	"eraya/util"
	"errors"
	"fmt"
	"log/slog"
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
		slog.Error("Failed to parse multipart form", "error", err)
		http.Error(w, "failed to parse multipart form", http.StatusBadRequest)
		return
	}

	// Extract product info from form
	name := r.FormValue("name")
	desc := r.FormValue("description")
	basePrice, _ := strconv.ParseFloat(r.FormValue("base_price"), 64)
	discountPrice, _ := strconv.ParseFloat(r.FormValue("discount_price"), 64)
	stockCount, _ := strconv.Atoi(r.FormValue("stock_count"))
	
	// Multiple category support
	catIDsStr := r.MultipartForm.Value["category_id"]
	var catIDs []int
	for _, idStr := range catIDsStr {
		if id, err := strconv.Atoi(idStr); err == nil {
			catIDs = append(catIDs, id)
		}
	}

	isActiveStr := r.FormValue("is_active")
	isActive := isActiveStr != "false" // default true unless explicitly "false"

	colorsStr := r.FormValue("colors")
	var colors []string
	if colorsStr != "" {
		for _, c := range strings.Split(colorsStr, ",") {
			cTrimmed := strings.TrimSpace(c)
			if cTrimmed != "" {
				colors = append(colors, cTrimmed)
			}
		}
	}

	sizesStr := r.FormValue("sizes")
	var sizes []string
	if sizesStr != "" {
		for _, s := range strings.Split(sizesStr, ",") {
			sTrimmed := strings.TrimSpace(s)
			if sTrimmed != "" {
				sizes = append(sizes, sTrimmed)
			}
		}
	}

	variationStockStr := r.FormValue("variation_stock")
	var variationStock domain.VariationStockList
	if variationStockStr != "" {
		_ = json.Unmarshal([]byte(variationStockStr), &variationStock)
	}

	p := &domain.Product{
		Name:           name,
		BasePrice:      basePrice,
		StockCount:     stockCount,
		IsActive:       isActive,
		Colors:         colors,
		Sizes:          sizes,
		VariationStock: variationStock,
	}
	if desc != "" {
		p.Description = &desc
	}
	if discountPrice > 0 {
		p.DiscountPrice = &discountPrice
	}
	p.CategoryIDs = catIDs

	// Slug generation
	p.Slug = strings.ToLower(strings.ReplaceAll(p.Name, " ", "-"))

	// Handle Image Uploads
	primaryIndexStr := r.FormValue("primary_image_index")
	primaryIndex, _ := strconv.Atoi(primaryIndexStr)

	files := r.MultipartForm.File["images"]
	for i, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			slog.Error("Failed to open file", "filename", fileHeader.Filename, "error", err)
			continue
		}
		defer file.Close()

		// Validate file type and size
		contentType := fileHeader.Header.Get("Content-Type")
		if err := util.ValidateImage(contentType); err != nil {
			slog.Error("Invalid image format", "filename", fileHeader.Filename, "error", err)
			file.Close()
			continue
		}

		if err := util.ValidateImageSize(fileHeader.Size, 2); err != nil {
			slog.Error("Image too large", "filename", fileHeader.Filename, "size", fileHeader.Size)
			http.Error(w, fmt.Sprintf("File %s is too large. Max 2MB allowed.", fileHeader.Filename), http.StatusBadRequest)
			return
		}

		// Upload to Supabase in 'products' folder
		url, err := h.storage.UploadFile("products", fileHeader.Filename, file, fileHeader.Header.Get("Content-Type"))
		if err != nil {
			slog.Error("Upload failed", "filename", fileHeader.Filename, "error", err)
			continue
		}

		p.Images = append(p.Images, domain.ProductImage{
			ImageURL:  url,
			IsPrimary: i == primaryIndex,
		})
	}

	created, err := h.svc.CreateProduct(r.Context(), p)
	if err != nil {
		// Cleanup uploaded images if DB fails
		for _, img := range p.Images {
			go h.storage.DeleteFile(img.ImageURL)
		}
		slog.Error("Failed to create product", "error", err)
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
// @Param search query string false "Search products"
// @Success 200 {object} util.PaginatedData
// @Router /products [get]
func (h *Handler) ListProducts(w http.ResponseWriter, r *http.Request) {
	pageAsStr := r.URL.Query().Get("page")
	limitAsStr := r.URL.Query().Get("limit")
	search := r.URL.Query().Get("search")
	sort := r.URL.Query().Get("sort")
	if sort == "" {
		sort = "newest"
	}
	
	catIDStrs := r.URL.Query()["category_id"]
	var categoryIDs []int
	for _, idStr := range catIDStrs {
		if id, err := strconv.Atoi(idStr); err == nil {
			categoryIDs = append(categoryIDs, id)
		}
	}

	minPrice, _ := strconv.ParseFloat(r.URL.Query().Get("min_price"), 64)
	maxPrice, _ := strconv.ParseFloat(r.URL.Query().Get("max_price"), 64)

	page, _ := strconv.ParseInt(pageAsStr, 10, 64)
	if page <= 0 {
		page = 1
	}

	limit, _ := strconv.ParseInt(limitAsStr, 10, 64)
	if limit <= 0 {
		limit = 10
	}

	// adminMode=true skips the is_active filter so all products (published+unpublished) are returned
	adminMode := r.URL.Query().Get("admin") == "true"

	products, count, err := h.svc.GetProducts(r.Context(), page, limit, search, categoryIDs, sort, minPrice, maxPrice, adminMode)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			slog.Error("Failed to list products", "error", err)
		}
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
	product, err := h.svc.GetProductBySlug(r.Context(), slug)
	if err != nil {
		slog.Warn("Product not found", "slug", slug)
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
	// Regenerate slug from name if not provided by client
	if p.Slug == "" && p.Name != "" {
		p.Slug = strings.ToLower(strings.ReplaceAll(p.Name, " ", "-"))
	}

	orphanedURLs, err := h.svc.UpdateProduct(r.Context(), &p)
	if err != nil {
		slog.Error("Failed to update product", "id", id, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Async cleanup of removed images
	for _, url := range orphanedURLs {
		go h.storage.DeleteFile(url)
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
	p, err := h.svc.GetProductByID(r.Context(), id)
	if err != nil {
		slog.Warn("Attempted to delete non-existent product", "id", id)
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	}

	if err := h.svc.DeleteProduct(r.Context(), id); err != nil {
		slog.Error("Failed to delete product", "id", id, "error", err)
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

// BulkDeleteProducts godoc
// @Summary Bulk delete products
// @Description Remove multiple products and their images by IDs (admin only).
// @Tags products
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param ids body object true "Product IDs to delete"
// @Success 200 {string} string "OK"
// @Router /products/bulk [delete]
func (h *Handler) BulkDeleteProducts(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IDs []int64 `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Fetch products first to get image URLs for cleanup
	for _, id := range req.IDs {
		p, err := h.svc.GetProductByID(r.Context(), id)
		if err == nil {
			for _, img := range p.Images {
				go h.storage.DeleteFile(img.ImageURL)
			}
		}
	}

	if err := h.svc.BulkDeleteProducts(r.Context(), req.IDs); err != nil {
		slog.Error("Failed to bulk delete products", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Products deleted successfully"))
}

// CreateCategory godoc
// @Summary Create a new category
// @Description Add a new category (admin only).
// @Tags categories
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param category body domain.Category true "Category Details"
// @Success 201 {object} domain.Category
// @Router /categories [post]
func (h *Handler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var c domain.Category
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	created, err := h.svc.CreateCategory(r.Context(), &c)
	if err != nil {
		util.SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

// ListCategories godoc
// @Summary List all categories
// @Description Retrieve a list of all product categories.
// @Tags categories
// @Produce json
// @Success 200 {array} domain.Category
// @Router /categories [get]
func (h *Handler) ListCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := h.svc.ListCategories(r.Context())
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			slog.Error("Failed to list categories", "error", err)
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	search := r.URL.Query().Get("search")
	var filtered []*domain.Category
	if search != "" {
		searchLower := strings.ToLower(search)
		for _, cat := range categories {
			if strings.Contains(strings.ToLower(cat.Name), searchLower) {
				filtered = append(filtered, cat)
			}
		}
	} else {
		filtered = categories
	}

	pageAsStr := r.URL.Query().Get("page")
	if pageAsStr != "" {
		page, _ := strconv.ParseInt(pageAsStr, 10, 64)
		if page <= 0 {
			page = 1
		}
		limitAsStr := r.URL.Query().Get("limit")
		limit, _ := strconv.ParseInt(limitAsStr, 10, 64)
		if limit <= 0 {
			limit = 10
		}

		total := int64(len(filtered))
		offset := (page - 1) * limit
		if offset > total {
			offset = total
		}
		end := offset + limit
		if end > total {
			end = total
		}

		paginatedCategories := filtered[offset:end]
		util.SendPage(w, paginatedCategories, page, limit, total)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(filtered)
}

// UpdateCategory godoc
// @Summary Update a category
// @Description Rename an existing product category.
// @Tags categories
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Category ID"
// @Param category body domain.Category true "Updated Category Info"
// @Success 200 {object} domain.Category
// @Router /categories/{id} [put]
func (h *Handler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	var c domain.Category
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	c.ID = id

	updated, err := h.svc.UpdateCategory(r.Context(), &c)
	if err != nil {
		slog.Error("Failed to update category", "id", id, "error", err)
		util.SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updated)
}

// DeleteCategory godoc
// @Summary Delete a category
// @Description Remove a category by ID (admin only).
// @Tags categories
// @Security BearerAuth
// @Param id path int true "Category ID"
// @Success 200 {string} string "OK"
// @Router /categories/{id} [delete]
func (h *Handler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))

	if err := h.svc.DeleteCategory(r.Context(), id); err != nil {
		slog.Error("Failed to delete category", "id", id, "error", err)
		// Usually fails if it is in use by products without ON DELETE CASCADE
		http.Error(w, "Failed to delete category. Ensure no products are using it.", http.StatusConflict)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Category deleted successfully"))
}

// BulkDeleteCategories godoc
// @Summary Bulk delete categories
// @Description Remove multiple categories by IDs (admin only).
// @Tags categories
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param ids body object true "Category IDs to delete"
// @Success 200 {string} string "OK"
// @Router /categories/bulk [delete]
func (h *Handler) BulkDeleteCategories(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IDs []int `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.svc.BulkDeleteCategories(r.Context(), req.IDs); err != nil {
		slog.Error("Failed to bulk delete categories", "error", err)
		http.Error(w, "Failed to delete some categories. Ensure they are not in use.", http.StatusConflict)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Categories deleted successfully"))
}
// UploadFile godoc
// @Summary Upload a general image
// @Description Upload an image to the 'categories' folder (admin only).
// @Tags admin
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param image formData file true "Image file"
// @Success 200 {object} map[string]string
// @Router /admin/upload [post]
func (h *Handler) UploadFile(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(5 << 20) // 5MB
	if err != nil {
		util.SendError(w, http.StatusBadRequest, "failed to parse multipart form")
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		util.SendError(w, http.StatusBadRequest, "image is required")
		return
	}
	defer file.Close()

	// Upload to 'categories' or 'general' folder
	url, err := h.storage.UploadFile("categories", header.Filename, file, header.Header.Get("Content-Type"))
	if err != nil {
		util.SendError(w, http.StatusInternalServerError, "upload failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"url": url})
}
