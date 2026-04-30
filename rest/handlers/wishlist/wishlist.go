package wishlist

import (
	"encoding/json"
	"eraya/util"
	"eraya/wishlist"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc wishlist.Service
}

func NewHandler(svc wishlist.Service) *Handler {
	return &Handler{svc: svc}
}

// Add godoc
// @Summary Add product to wishlist
// @Description Add a specific product to the authenticated user's wishlist.
// @Tags wishlist
// @Security BearerAuth
// @Param product_id path int true "Product ID"
// @Success 200 {object} map[string]string
// @Router /wishlist/add/{product_id} [post]
func (h *Handler) Add(w http.ResponseWriter, r *http.Request) {
	productID, _ := strconv.ParseInt(chi.URLParam(r, "product_id"), 10, 64)
	userID := r.Context().Value("user_id").(int64)

	err := h.svc.AddToWishlist(r.Context(), userID, productID)
	if err != nil {
		util.SendError(w, http.StatusInternalServerError, "Failed to add to wishlist")
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Added to wishlist"})
}

// Remove godoc
// @Summary Remove product from wishlist
// @Description Remove a specific product from the authenticated user's wishlist.
// @Tags wishlist
// @Security BearerAuth
// @Param product_id path int true "Product ID"
// @Success 200 {object} map[string]string
// @Router /wishlist/remove/{product_id} [delete]
func (h *Handler) Remove(w http.ResponseWriter, r *http.Request) {
	productID, _ := strconv.ParseInt(chi.URLParam(r, "product_id"), 10, 64)
	userID := r.Context().Value("user_id").(int64)

	err := h.svc.RemoveFromWishlist(r.Context(), userID, productID)
	if err != nil {
		util.SendError(w, http.StatusInternalServerError, "Failed to remove from wishlist")
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Removed from wishlist"})
}

// List godoc
// @Summary List wishlist items
// @Description Fetch all products in the authenticated user's wishlist.
// @Tags wishlist
// @Security BearerAuth
// @Success 200 {array} domain.Product
// @Router /wishlist [get]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int64)

	items, err := h.svc.GetUserWishlist(r.Context(), userID)
	if err != nil {
		util.SendError(w, http.StatusInternalServerError, "Failed to fetch wishlist")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

// Clear godoc
// @Summary Clear wishlist
// @Description Remove all items from the authenticated user's wishlist.
// @Tags wishlist
// @Security BearerAuth
// @Success 200 {object} map[string]string
// @Router /wishlist/clear [delete]
func (h *Handler) Clear(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int64)

	err := h.svc.ClearWishlist(r.Context(), userID)
	if err != nil {
		util.SendError(w, http.StatusInternalServerError, "Failed to clear wishlist")
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Wishlist cleared"})
}
