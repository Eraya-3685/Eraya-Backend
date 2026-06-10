package review

import (
	"encoding/json"
	"eraya/infra/storage"
	"eraya/review"
	"eraya/util"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc     review.Service
	storage *storage.StorageService
}

func NewHandler(svc review.Service, storage *storage.StorageService) *Handler {
	return &Handler{
		svc:     svc,
		storage: storage,
	}
}

type createReviewReq struct {
	ProductID int64   `json:"product_id"`
	Rating    int     `json:"rating"`
	Comment   string  `json:"comment"`
	ImageURL  *string `json:"image_url"`
}

// CreateReview godoc
// @Summary Submit a product review
// @Description Submit a rating and comment for a product (buyers only).
// @Tags reviews
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body createReviewReq true "Review Details"
// @Success 201 {object} domain.Review
// @Failure 403 {string} string "Forbidden"
// @Router /reviews [post]
func (h *Handler) CreateReview(w http.ResponseWriter, r *http.Request) {
	role := r.Context().Value("role").(string)
	if role == "admin" {
		http.Error(w, "admins cannot post reviews", http.StatusForbidden)
		return
	}

	userID := r.Context().Value("user_id").(int64)

	var req createReviewReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	rev, err := h.svc.CreateReview(r.Context(), userID, req.ProductID, req.Rating, req.Comment, req.ImageURL)
	if err != nil {
		slog.Error("Failed to create review", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(rev)
}

// GetProductReviews godoc
// @Summary Get reviews for a product
// @Description Retrieve all reviews for a specific product.
// @Tags reviews
// @Produce json
// @Param productId path int true "Product ID"
// @Success 200 {array} domain.Review
// @Router /reviews/{productId} [get]
func (h *Handler) GetProductReviews(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "productId")
	productID, _ := strconv.ParseInt(idStr, 10, 64)

	reviews, err := h.svc.GetProductReviews(r.Context(), productID)
	if err != nil {
		slog.Error("Failed to get product reviews", "product_id", productID, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reviews)
}

// ListAllReviews godoc
// @Summary List all reviews
// @Description Retrieve all reviews (admin only).
// @Tags reviews
// @Produce json
// @Security BearerAuth
// @Success 200 {array} domain.Review
// @Router /admin/reviews [get]
func (h *Handler) ListAllReviews(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	filter := r.URL.Query().Get("filter")

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

		reviews, count, err := h.svc.ListAllReviews(r.Context(), page, limit, search, filter)
		if err != nil {
			slog.Error("Failed to list all reviews", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		util.SendPage(w, reviews, page, limit, count)
		return
	}

	reviews, _, err := h.svc.ListAllReviews(r.Context(), 0, 0, search, filter)
	if err != nil {
		slog.Error("Failed to list all reviews", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reviews)
}

// ApproveReview godoc
// @Summary Approve a review
// @Description Approve a pending review (admin only).
// @Tags reviews
// @Produce json
// @Security BearerAuth
// @Param id path int true "Review ID"
// @Success 200 {string} string "OK"
// @Router /admin/reviews/{id}/approve [post]
func (h *Handler) ApproveReview(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

	if err := h.svc.ApproveReview(r.Context(), id); err != nil {
		slog.Error("Failed to approve review", "id", id, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Review approved successfully"))
}

// DeleteReview godoc
// @Summary Delete a review
// @Description Remove a review (admin only).
// @Tags reviews
// @Produce json
// @Security BearerAuth
// @Param id path int true "Review ID"
// @Success 200 {string} string "OK"
// @Router /reviews/{id} [delete]
func (h *Handler) DeleteReview(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

	if err := h.svc.DeleteReview(r.Context(), id); err != nil {
		slog.Error("Failed to delete review", "id", id, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Review deleted successfully"))
}

func (h *Handler) UploadReviewImage(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(5 << 20) // 5MB
	if err != nil {
		http.Error(w, "failed to parse multipart form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "image file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	url, err := h.storage.UploadFile("reviews", header.Filename, file, header.Header.Get("Content-Type"))
	if err != nil {
		slog.Error("Review image upload failed", "error", err)
		http.Error(w, "upload failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"url": url})
}

