package order

import (
	"context"
	"encoding/json"
	"eraya/config"
	"eraya/domain"
	"eraya/infra/bkash"
	"eraya/order"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
)

type Handler struct {
	svc         order.Service
	bkashClient *bkash.Client
}

func NewHandler(svc order.Service, bkashClient *bkash.Client) *Handler {
	return &Handler{
		svc:         svc,
		bkashClient: bkashClient,
	}
}

var adminUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for admin panel
	},
}

var (
	adminConns = make(map[*websocket.Conn]bool)
	adminMu    sync.Mutex
)

func broadcastToAdmins(msg any) {
	adminMu.Lock()
	defer adminMu.Unlock()
	for conn := range adminConns {
		if err := conn.WriteJSON(msg); err != nil {
			slog.Error("Failed to write to admin websocket, closing connection", "error", err)
			conn.Close()
			delete(adminConns, conn)
		}
	}
}

func (h *Handler) AdminWebSocket(w http.ResponseWriter, r *http.Request) {
	ws, err := adminUpgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("Failed to upgrade admin connection", "error", err)
		return
	}
	
	adminMu.Lock()
	adminConns[ws] = true
	adminMu.Unlock()

	defer func() {
		ws.Close()
		adminMu.Lock()
		delete(adminConns, ws)
		adminMu.Unlock()
	}()

	// Keep connection alive with simple read loop
	for {
		var msg map[string]string
		if err := ws.ReadJSON(&msg); err != nil {
			break
		}
		if msg["type"] == "ping" {
			_ = ws.WriteJSON(map[string]string{"type": "pong"})
		}
	}
}

type addToCartReq struct {
	ProductID     int64  `json:"product_id"`
	Quantity      int    `json:"quantity"`
	SelectedColor string `json:"selected_color"`
	SelectedSize  string `json:"selected_size"`
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

	err := h.svc.AddToCart(r.Context(), userID, req.ProductID, req.Quantity, req.SelectedColor, req.SelectedSize)
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

type checkoutItem struct {
	ProductID     int64  `json:"product_id"`
	Quantity      int    `json:"quantity"`
	SelectedColor string `json:"selected_color"`
	SelectedSize  string `json:"selected_size"`
}

type checkoutReq struct {
	Items           []checkoutItem `json:"items"`
	PaymentMethod   string         `json:"payment_method"`
	ShippingAddress string         `json:"shipping_address"`
	TrxID           *string        `json:"trx_id"`
	SenderNumber    *string        `json:"sender_number"`
	PaidAmount      *float64       `json:"paid_amount"`
	CouponCode      *string        `json:"coupon_code"`
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

	items := make([]domain.CartItem, len(req.Items))
	for i, it := range req.Items {
		items[i] = domain.CartItem{
			ProductID:     it.ProductID,
			Quantity:      it.Quantity,
			SelectedColor: it.SelectedColor,
			SelectedSize:  it.SelectedSize,
		}
	}

	order, err := h.svc.Checkout(r.Context(), userID, items, req.PaymentMethod, req.ShippingAddress, req.TrxID, req.SenderNumber, req.PaidAmount, req.CouponCode)
	if err != nil {
		slog.Error("Checkout failed", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.PaymentMethod == "bKash" && req.TrxID != nil && *req.TrxID != "" {
		err = h.svc.ConfirmPayment(r.Context(), order.ID, *req.TrxID, *req.PaidAmount)
		if err == nil {
			order.OrderStatus = "Pending" // Keep it pending for admin review
			order.PaymentStatus = "Paid"
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"order": order,
	})

	// Broadcast real-time order event to connected admins
	broadcastToAdmins(map[string]any{
		"type":     "NEW_ORDER",
		"message":  fmt.Sprintf("A customer placed a new order! Order ID: #%d", order.ID),
		"order_id": order.ID,
	})
}

type bkashInitReq struct {
	Amount float64 `json:"amount"`
}

// InitBKashPayment godoc
// @Summary Initialize bKash payment
// @Description Start a bKash payment process. Returns a redirect URL to bKash gateway.
// @Tags orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body bkashInitReq true "Payment Amount"
// @Success 200 {object} map[string]string
// @Router /orders/bkash/init [post]
func (h *Handler) InitBKashPayment(w http.ResponseWriter, r *http.Request) {
	var req bkashInitReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	invoiceNumber := fmt.Sprintf("TEMP-%d", time.Now().UnixNano())
	callbackURL := config.GetConfig().BaseURL + "/orders/bkash/callback"

	bkashRes, err := h.bkashClient.CreatePayment(req.Amount, invoiceNumber, callbackURL)
	var bkashURL string
	if err != nil {
		slog.Error("Failed to create bKash payment, falling back to mock UI", "error", err)
		bkashURL = fmt.Sprintf("%s/bkash-sandbox?amount=%.2f&invoice=%s&callback=%s", config.GetConfig().FrontendURL, req.Amount, invoiceNumber, callbackURL)
	} else {
		bkashURL = bkashRes.BkashURL
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"bkashURL": bkashURL})
}

// BKashCallback godoc
// @Summary bKash payment callback
// @Description Internal callback URL for bKash to report payment status.
// @Tags orders
// @Param paymentID query string true "bKash Payment ID"
// @Param status query string true "Payment Status"
// @Success 302 {string} string "Redirect"
// @Router /orders/bkash/callback [get]
func (h *Handler) BKashCallback(w http.ResponseWriter, r *http.Request) {
	paymentID := r.URL.Query().Get("paymentID")
	status := r.URL.Query().Get("status")

	if status != "success" || paymentID == "" {
		http.Redirect(w, r, config.GetConfig().FrontendURL+"/checkout?bkash_error=true", http.StatusTemporaryRedirect)
		return
	}

	res, err := h.bkashClient.ExecutePayment(paymentID)
	if err != nil {
		slog.Error("Failed to execute bKash payment", "error", err)
		http.Redirect(w, r, config.GetConfig().FrontendURL+"/checkout?bkash_error=true", http.StatusTemporaryRedirect)
		return
	}

	senderNumber := r.URL.Query().Get("senderNumber")
	redirectURL := fmt.Sprintf("%s/checkout?bkash_success=true&trxID=%s&amount=%s&senderNumber=%s", config.GetConfig().FrontendURL, res.TrxID, res.Amount, senderNumber)
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
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

// GetOrder godoc
// @Summary Get order details
// @Description Retrieve details for a specific order (buyers only).
// @Tags orders
// @Produce json
// @Security BearerAuth
// @Param id path int true "Order ID"
// @Success 200 {object} domain.Order
// @Failure 403 {string} string "Forbidden"
// @Failure 404 {string} string "Not Found"
// @Router /orders/{id} [get]
func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	role := r.Context().Value("role").(string)
	if role == "admin" || role == "moderator" {
		http.Error(w, "staff do not use this endpoint", http.StatusForbidden)
		return
	}

	idStr := chi.URLParam(r, "id")
	orderID, _ := strconv.ParseInt(idStr, 10, 64)
	userID := r.Context().Value("user_id").(int64)

	order, err := h.svc.GetOrderByID(r.Context(), orderID, userID)
	if err != nil {
		slog.Error("Failed to get order", "id", orderID, "error", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
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

	// Broadcast confirmed order event to connected admins
	broadcastToAdmins(map[string]any{
		"type":     "ORDER_CONFIRMED",
		"message":  fmt.Sprintf("Order #%d confirmed!", orderID),
		"order_id": orderID,
	})
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
	adminID := r.Context().Value("user_id").(int64)

	var req struct {
		OTP string `json:"otp"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "OTP is required", http.StatusBadRequest)
		return
	}

	err := h.svc.AdminDeleteOrder(r.Context(), orderID, req.OTP, adminID)
	if err != nil {
		slog.Error("Admin failed to delete order", "id", orderID, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Order deleted successfully"))

	// Broadcast deleted order event to connected admins
	broadcastToAdmins(map[string]any{
		"type":     "ORDER_DELETED",
		"message":  fmt.Sprintf("Order #%d deleted!", orderID),
		"order_id": orderID,
	})
}

// AdminRequestDeleteOTP godoc
// @Summary Request OTP for order deletion (Admin)
// @Description Send an OTP to the admin's email to authorize order deletion.
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Success 200 {string} string "OK"
// @Router /admin/orders/request-delete-otp [post]
func (h *Handler) AdminRequestDeleteOTP(w http.ResponseWriter, r *http.Request) {
	adminID := r.Context().Value("user_id").(int64)

	err := h.svc.AdminRequestDeleteOTP(r.Context(), adminID)
	if err != nil {
		slog.Error("Admin failed to request delete OTP", "admin_id", adminID, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OTP sent to your email"))
}

type updateStatusReq struct {
	Status        string `json:"status"`
	EstimatedDate string `json:"estimated_date"`
}

// AdminUpdateStatus godoc
// @Summary Update order status (Admin)
// @Description Update the status and estimated delivery date of an order (admin/moderator only).
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Order ID"
// @Param body body updateStatusReq true "Status Update Details"
// @Success 200 {string} string "OK"
// @Router /admin/orders/{id}/status [put]
func (h *Handler) AdminUpdateStatus(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	orderID, _ := strconv.ParseInt(idStr, 10, 64)

	var req updateStatusReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := h.svc.AdminUpdateOrderStatus(r.Context(), orderID, req.Status, req.EstimatedDate)
	if err != nil {
		slog.Error("Admin failed to update order status", "id", orderID, "status", req.Status, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	// Broadcast status updated event to connected admins
	broadcastToAdmins(map[string]any{
		"type":     "ORDER_STATUS_UPDATED",
		"message":  fmt.Sprintf("Order #%d status updated to %s", orderID, req.Status),
		"order_id": orderID,
	})
}

// AdminGetStats godoc
// @Summary Get dashboard analytics (Admin)
// @Description Retrieve aggregated metrics for the admin dashboard (admin only).
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Param timeframe query string false "Timeframe (day, week, month, year)"
// @Success 200 {object} domain.DashboardStats
// @Router /admin/orders/stats [get]
func (h *Handler) AdminGetStats(w http.ResponseWriter, r *http.Request) {
	timeframe := r.URL.Query().Get("timeframe")
	adminID := r.Context().Value("user_id").(int64)
	stats, err := h.svc.AdminGetDashboardStats(r.Context(), adminID, timeframe)
	if err != nil {
		slog.Error("Admin failed to get dashboard stats", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
