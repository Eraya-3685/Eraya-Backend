package aichat_handler

import (
	"encoding/json"
	"eraya/aichat"
	erayamiddleware "eraya/rest/middlewares"
	"eraya/user"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc aichat.Service
}

func NewHandler(svc aichat.Service) *Handler {
	return &Handler{svc: svc}
}

type chatRequest struct {
	Message string              `json:"message"`
	History []aichat.ChatMessage `json:"history"`
}

type chatResponse struct {
	Reply string `json:"reply"`
}

// Simple per-user rate limiter: 20 requests/minute
type rateLimiter struct {
	mu       sync.Mutex
	requests map[int64][]time.Time
}

var limiter = &rateLimiter{requests: make(map[int64][]time.Time)}

func (rl *rateLimiter) allow(userID int64) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	window := now.Add(-1 * time.Minute)

	// Clean old entries
	var recent []time.Time
	for _, t := range rl.requests[userID] {
		if t.After(window) {
			recent = append(recent, t)
		}
	}

	if len(recent) >= 20 {
		rl.requests[userID] = recent
		return false
	}

	rl.requests[userID] = append(recent, now)
	return true
}

func RegisterRoutes(r chi.Router, h *Handler, jwtSecret string, userSvc user.Service) {
	r.Route("/ai", func(r chi.Router) {
		r.Use(erayamiddleware.OptionalAuthMiddleware(jwtSecret, userSvc))
		r.Post("/chat", h.Chat)
		r.Get("/chat", h.GetHistory)
	})
}

func (h *Handler) Chat(w http.ResponseWriter, r *http.Request) {
	var limiterKey int64
	var userIDPtr *int64
	userID, ok := r.Context().Value("user_id").(int64)
	if ok {
		limiterKey = userID
		userIDPtr = &userID
	} else {
		// Use a combination of IP address and a negative offset for guests to separate from user IDs
		// In a real app, you'd use a better IP-based rate limiter
		limiterKey = -1 // Simplified for now
	}

	if !limiter.allow(limiterKey) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]string{"error": "Too many requests. Please wait a moment."})
		return
	}

	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Message == "" {
		http.Error(w, `{"error":"message is required"}`, http.StatusBadRequest)
		return
	}

	// Limit message length
	if len(req.Message) > 1000 {
		http.Error(w, `{"error":"message too long (max 1000 characters)"}`, http.StatusBadRequest)
		return
	}

	// Limit history to last 20 messages
	if len(req.History) > 20 {
		req.History = req.History[len(req.History)-20:]
	}

	reply, err := h.svc.Chat(r.Context(), userIDPtr, req.Message, req.History)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "AI service is temporarily unavailable. Please try again.",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chatResponse{Reply: reply})
}

func (h *Handler) GetHistory(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(int64)
	if !ok {
		// Guest users do not have history
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]aichat.ChatMessage{})
		return
	}

	history, err := h.svc.GetHistory(r.Context(), userID, 50)
	if err != nil {
		http.Error(w, `{"error":"failed to fetch history"}`, http.StatusInternalServerError)
		return
	}

	if history == nil {
		history = []aichat.ChatMessage{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}
