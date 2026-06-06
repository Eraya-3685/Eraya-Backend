package coupon

import (
	"encoding/json"
	"eraya/domain"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc domain.CouponService
}

func NewHandler(svc domain.CouponService) *Handler {
	return &Handler{svc: svc}
}

type createCouponReq struct {
	Code          string    `json:"code"`
	DiscountType  string    `json:"discount_type"`
	DiscountValue float64   `json:"discount_value"`
	MinCartValue  float64   `json:"min_cart_value"`
	ExpiresAt     time.Time `json:"expires_at"`
}

func (h *Handler) CreateCoupon(w http.ResponseWriter, r *http.Request) {
	var req createCouponReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	c, err := h.svc.CreateCoupon(r.Context(), req.Code, req.DiscountType, req.DiscountValue, req.MinCartValue, req.ExpiresAt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(c)
}

func (h *Handler) ListCoupons(w http.ResponseWriter, r *http.Request) {
	coupons, err := h.svc.ListCoupons(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(coupons)
}

func (h *Handler) DeleteCoupon(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := h.svc.DeleteCoupon(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("coupon deleted"))
}

type applyCouponReq struct {
	Code      string  `json:"code"`
	CartTotal float64 `json:"cart_total"`
}

type applyCouponResp struct {
	Coupon         *domain.Coupon `json:"coupon"`
	DiscountAmount float64        `json:"discount_amount"`
}

func (h *Handler) ApplyCoupon(w http.ResponseWriter, r *http.Request) {
	var req applyCouponReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	c, discount, err := h.svc.ValidateAndApplyCoupon(r.Context(), req.Code, req.CartTotal)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(applyCouponResp{
		Coupon:         c,
		DiscountAmount: discount,
	})
}
