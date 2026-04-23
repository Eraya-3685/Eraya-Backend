package review

import (
	"encoding/json"
	"eraya/review"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc review.Service
}

func NewHandler(svc review.Service) *Handler {
	return &Handler{
		svc: svc,
	}
}

type createReviewReq struct {
	ProductID int64  `json:"product_id"`
	Rating    int    `json:"rating"`
	Comment   string `json:"comment"`
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

	rev, err := h.svc.CreateReview(userID, req.ProductID, req.Rating, req.Comment)
	if err != nil {
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

	reviews, err := h.svc.GetProductReviews(productID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reviews)
}
