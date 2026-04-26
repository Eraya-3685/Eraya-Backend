package order

import (
	"context"
	"encoding/json"
	"eraya/order"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc order.Service
}

func NewHandler(svc order.Service) *Handler {
	return &Handler{
		svc: svc,
	}
}

type addToCartReq struct {
	ProductID int64 `json:"product_id"`
	Quantity  int   `json:"quantity"`
}

// AddToCart godoc
// @Summary Add a product to cart
// @Description Add a product to the user's shopping cart (buyers only).
// @Tags cart
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body addToCartReq true "Product Details"
// @Success 201 {string} string "Created"
// @Failure 403 {string} string "Forbidden"
// @Router /cart [post]
func (h *Handler) AddToCart(w http.ResponseWriter, r *http.Request) {
	role := r.Context().Value("role").(string)
	if role == "admin" || role == "moderator" {
		http.Error(w, "staff cannot manage carts", http.StatusForbidden)
		return
	}

	userID := r.Context().Value("user_id").(int64)

	var req addToCartReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := h.svc.AddToCart(r.Context(), userID, req.ProductID, req.Quantity)
	if err != nil {
		slog.Error("Failed to add to cart", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// GetCart godoc
// @Summary View shopping cart
// @Description Retrieve all items in the logged-in user's shopping cart.
// @Tags cart
// @Produce json
// @Security BearerAuth
// @Success 200 {array} domain.CartItem
// @Failure 403 {string} string "Forbidden"
// @Router /cart [get]
func (h *Handler) GetCart(w http.ResponseWriter, r *http.Request) {
	role := r.Context().Value("role").(string)
	if role == "admin" || role == "moderator" {
		http.Error(w, "staff cannot manage carts", http.StatusForbidden)
		return
	}

	userID := r.Context().Value("user_id").(int64)

	items, err := h.svc.GetCart(r.Context(), userID)
	if err != nil {
		slog.Error("Failed to get cart", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

type checkoutReq struct {
	PaymentMethod   string `json:"payment_method"`
	ShippingAddress string `json:"shipping_address"`
}

// Checkout godoc
// @Summary Place an order
// @Description Create an order from the items in the cart (buyers only).
// @Tags orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body checkoutReq true "Checkout Details"
// @Success 201 {object} domain.Order
// @Failure 400 {string} string "Bad Request"
// @Router /orders/checkout [post]
func (h *Handler) Checkout(w http.ResponseWriter, r *http.Request) {
	role := r.Context().Value("role").(string)
	if role == "admin" || role == "moderator" {
		http.Error(w, "staff cannot place orders", http.StatusForbidden)
		return
	}

	userID := r.Context().Value("user_id").(int64)

	var req checkoutReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	order, err := h.svc.Checkout(r.Context(), userID, req.PaymentMethod, req.ShippingAddress)
	if err != nil {
		slog.Error("Checkout failed", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(order)
}

// GetMyOrders godoc
// @Summary Get order history
// @Description Retrieve a list of all orders placed by the logged-in user.
// @Tags orders
// @Produce json
// @Security BearerAuth
// @Success 200 {array} domain.Order
// @Failure 403 {string} string "Forbidden"
// @Router /orders [get]
func (h *Handler) GetMyOrders(w http.ResponseWriter, r *http.Request) {
	role := r.Context().Value("role").(string)
	if role == "admin" || role == "moderator" {
		http.Error(w, "staff do not have order history", http.StatusForbidden)
		return
	}

	userID := r.Context().Value("user_id").(int64)

	orders, err := h.svc.GetOrders(r.Context(), userID)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			slog.Error("Failed to get orders", "error", err)
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}

// AdminGetOrders godoc
// @Summary List all orders (Admin)
// @Description Retrieve a list of all orders in the system (admin only).
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Success 200 {array} domain.Order
// @Failure 403 {string} string "Forbidden"
// @Router /admin/orders [get]
func (h *Handler) AdminGetOrders(w http.ResponseWriter, r *http.Request) {
	orders, err := h.svc.AdminGetAllOrders(r.Context())
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			slog.Error("Admin failed to get orders", "error", err)
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}

// AdminConfirmOrder godoc
// @Summary Confirm an order (Admin)
// @Description Mark an order as confirmed and paid (admin only).
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Param id path int true "Order ID"
// @Success 200 {string} string "OK"
// @Failure 403 {string} string "Forbidden"
// @Router /admin/orders/{id}/confirm [post]
func (h *Handler) AdminConfirmOrder(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	orderID, _ := strconv.ParseInt(idStr, 10, 64)

	err := h.svc.AdminConfirmOrder(r.Context(), orderID)
	if err != nil {
		slog.Error("Admin failed to confirm order", "id", orderID, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// AdminDeleteOrder godoc
// @Summary Delete an order (Admin)
// @Description Remove an order from the system (admin only).
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Param id path int true "Order ID"
// @Success 200 {string} string "OK"
// @Failure 403 {string} string "Forbidden"
// @Router /admin/orders/{id} [delete]
func (h *Handler) AdminDeleteOrder(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	orderID, _ := strconv.ParseInt(idStr, 10, 64)

	err := h.svc.AdminDeleteOrder(r.Context(), orderID)
	if err != nil {
		slog.Error("Admin failed to delete order", "id", orderID, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Order deleted successfully"))
}

