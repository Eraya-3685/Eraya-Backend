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
