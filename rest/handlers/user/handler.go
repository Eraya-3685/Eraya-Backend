package user

import (
	"encoding/json"
	"eraya/domain"
	"eraya/infra/storage"
	"eraya/user"
	"eraya/util"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc     user.Service
	storage *storage.StorageService
}

func NewHandler(svc user.Service, storageService *storage.StorageService) *Handler {
	return &Handler{
		svc:     svc,
		storage: storageService,
	}
}

type signupReq struct {
	FullName string  `json:"full_name"`
	Email    string  `json:"email"`
	Password string  `json:"password"`
	Phone    *string `json:"phone"`
	Address  *string `json:"address"`
}

// Signup godoc
// @Summary Register a new user
// @Description Create a new user account with full name, email, password, phone, and address.
// @Tags users
// @Accept json
// @Produce json
// @Param signup body signupReq true "Signup Details"
// @Success 201 {object} domain.User
// @Failure 400 {string} string "Bad Request"
// @Router /users/signup [post]
func (h *Handler) Signup(w http.ResponseWriter, r *http.Request) {
	var req signupReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	u := &domain.User{
		FullName: req.FullName,
		Email:    req.Email,
		Phone:    req.Phone,
		Address:  req.Address,
		Role:     "buyer",
	}

	createdUser, err := h.svc.Signup(r.Context(), u, req.Password)
	if err != nil {
		slog.Error("Signup failed", "email", req.Email, "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createdUser)
}

type loginReq struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

// Login godoc
// @Summary Login a user
// @Description Authenticate a user using email/phone and password. Returns a JWT token.
// @Tags users
// @Accept json
// @Produce json
// @Param login body loginReq true "Login Details"
// @Success 200 {object} map[string]string
// @Failure 401 {string} string "Unauthorized"
// @Router /users/login [post]
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Returns token + full user object in one shot — no second /profile call needed
	token, user, err := h.svc.Login(r.Context(), req.Identifier, req.Password)
	if err != nil {
		slog.Warn("Login failed", "identifier", req.Identifier, "error", err)
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"token": token,
		"user":  user,
	})
}

// GetProfile godoc
// @Summary Get user profile
// @Description Retrieve the profile details of the currently logged-in user.
// @Tags users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} domain.User
// @Failure 401 {string} string "Unauthorized"
// @Router /users/profile [get]
func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userIDVal := r.Context().Value("user_id")
	if userIDVal == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	userID := userIDVal.(int64)
	user, err := h.svc.GetProfile(r.Context(), userID)
	if err != nil || user == nil {
		slog.Error("Failed to get profile", "id", userID, "error", err)
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

type updateProfileReq struct {
	FullName string  `json:"full_name"`
	Email    *string `json:"email"`
	Phone    *string `json:"phone"`
	Address  *string `json:"address"`
}

// UpdateProfile godoc
// @Summary Update user profile
// @Description Update the logged-in user's name, phone, and address. Email cannot be changed.
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body updateProfileReq true "Profile fields to update"
// @Success 200 {object} domain.User
// @Failure 400 {string} string "Bad Request"
// @Router /users/profile [patch]
func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userIDVal := r.Context().Value("user_id")
	if userIDVal == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	userID := userIDVal.(int64)

	var req updateProfileReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.svc.UpdateProfile(r.Context(), userID, req.FullName, req.Email, req.Phone, req.Address); err != nil {
		slog.Error("Failed to update profile", "id", userID, "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Return updated profile
	user, _ := h.svc.GetProfile(r.Context(), userID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

type updateRoleReq struct {
	Role string `json:"role"`
}

// UpdateUserRole godoc
// @Summary Update user role
// @Description Update the role of a user (admin only).
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Param body body updateRoleReq true "New Role"
// @Success 200 {string} string "OK"
// @Failure 403 {string} string "Forbidden"
// @Router /users/{id}/role [patch]
func (h *Handler) UpdateUserRole(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	userID, _ := strconv.ParseInt(idStr, 10, 64)

	var req updateRoleReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Role != "admin" && req.Role != "buyer" {
		http.Error(w, "invalid role", http.StatusBadRequest)
		return
	}

	err := h.svc.UpdateRole(r.Context(), userID, req.Role)
	if err != nil {
		slog.Error("Admin failed to update role", "target_id", userID, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// UploadAvatar godoc
// @Summary Upload or update user avatar
// @Description Upload a profile photo (max 5MB). Stores to Supabase and saves URL to DB.
// @Tags users
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param avatar formData file true "Avatar image file"
// @Success 200 {object} map[string]string
// @Router /users/avatar [patch]
func (h *Handler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	userIDVal := r.Context().Value("user_id")
	if userIDVal == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	userID := userIDVal.(int64)

	// Max 2MB per file check, but we keep ParseMultipartForm slightly higher
	if err := r.ParseMultipartForm(5 << 20); err != nil {
		http.Error(w, "invalid multipart form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("avatar")
	if err != nil {
		http.Error(w, "avatar file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Validate file type and size
	contentType := header.Header.Get("Content-Type")
	if err := util.ValidateImage(contentType); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := util.ValidateImageSize(header.Size, 2); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	url, err := h.svc.UploadAvatar(r.Context(), userID, header.Filename, file, contentType)
	if err != nil {
		slog.Error("Failed to upload avatar", "user_id", userID, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"avatar_url": url})
}
