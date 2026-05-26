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
	svc       user.Service
	storage   *storage.StorageService
	jwtSecret string
}

func NewHandler(svc user.Service, storageService *storage.StorageService, jwtSecret string) *Handler {
	return &Handler{
		svc:       svc,
		storage:   storageService,
		jwtSecret: jwtSecret,
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
// @Description Create a new buyer account. Requires a valid OTP previously sent via /users/signup-otp.
// @Tags users
// @Accept multipart/form-data
// @Produce json
// @Param full_name formData string true "Full Name"
// @Param email formData string true "Email"
// @Param password formData string true "Password"
// @Param phone formData string false "Phone"
// @Param address formData string false "Address"
// @Param otp formData string true "OTP"
// @Param avatar formData file false "Avatar"
// @Success 201 {object} map[string]any
// @Failure 400 {string} string "Bad Request"
// @Router /users/signup [post]
func (h *Handler) Signup(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form
	if err := r.ParseMultipartForm(5 << 20); err != nil {
		util.SendError(w, http.StatusBadRequest, "failed to parse form")
		return
	}

	fullName := r.FormValue("full_name")
	email := r.FormValue("email")
	password := r.FormValue("password")
	phone := r.FormValue("phone")
	address := r.FormValue("address")

	u := &domain.User{
		FullName: fullName,
		Email:    email,
		Role:     "buyer",
		IsActive: false, // Inactive until verified
	}
	if phone != "" {
		u.Phone = &phone
	}
	if address != "" {
		u.Address = &address
	}

	createdUser, err := h.svc.Signup(r.Context(), u, password)
	if err != nil {
		util.SendError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Handle Avatar Upload if present
	file, header, err := r.FormFile("avatar")
	if err == nil {
		defer file.Close()
		contentType := header.Header.Get("Content-Type")
		if util.ValidateImage(contentType) == nil && util.ValidateImageSize(header.Size, 2) == nil {
			url, uploadErr := h.svc.UploadAvatar(r.Context(), createdUser.ID, header.Filename, file, contentType)
			if uploadErr == nil {
				createdUser.AvatarURL = &url
			}
		}
	}

	util.SendData(w, http.StatusCreated, map[string]any{
		"message": "Registration successful. Please verify your email.",
		"user":    createdUser,
	})
}

// VerifySignup godoc
// @Summary Verify signup OTP
// @Description Verify the OTP sent to email during signup and activate the account.
// @Tags users
// @Accept json
// @Produce json
// @Param body body map[string]any true "User ID and OTP"
// @Success 200 {object} map[string]any
// @Router /users/verify-signup [post]
func (h *Handler) VerifySignup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID int64  `json:"user_id"`
		OTP    string `json:"otp"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.SendError(w, http.StatusBadRequest, "invalid request")
		return
	}

	token, user, err := h.svc.VerifySignup(r.Context(), req.UserID, req.OTP)
	if err != nil {
		util.SendError(w, http.StatusUnauthorized, err.Error())
		return
	}

	util.SendData(w, http.StatusOK, map[string]any{
		"token": token,
		"user":  user,
	})
}

// ResendActivationOTP godoc
// @Summary Resend activation OTP
// @Description Resend the account activation code to the user's email.
// @Tags users
// @Accept json
// @Produce json
// @Param body body map[string]int64 true "User ID"
// @Success 200 {string} string "OK"
// @Router /users/resend-activation [post]
func (h *Handler) ResendActivationOTP(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID int64 `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.SendError(w, http.StatusBadRequest, "invalid request")
		return
	}

	// Verify user is actually inactive before sending
	u, err := h.svc.GetProfile(r.Context(), req.UserID)
	if err != nil || u == nil {
		util.SendError(w, http.StatusNotFound, "user not found")
		return
	}
	if u.IsActive {
		util.SendError(w, http.StatusBadRequest, "account is already active")
		return
	}

	err = h.svc.RequestOTP(r.Context(), req.UserID, "verify_signup")
	if err != nil {
		util.SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	util.SendOK(w, "Verification code resent successfully")
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
		if err.Error() == "user not found" {
			util.SendError(w, http.StatusNotFound, "No account found with this email or phone")
			return
		}
		if err.Error() == "incorrect password" {
			util.SendError(w, http.StatusUnauthorized, "Incorrect password. Please try again.")
			return
		}
		util.SendError(w, http.StatusUnauthorized, err.Error())
		return
	}

	util.SendData(w, http.StatusOK, map[string]any{
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

	err := h.svc.UpdateRole(r.Context(), userID, req.Role)
	if err != nil {
		slog.Error("Admin failed to update role", "target_id", userID, "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

type bulkUpdateRoleReq struct {
	IDs         []int64  `json:"ids"`
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
	OTP         string   `json:"otp"`
	Password    string   `json:"password"`
}

// BulkUpdateUserRole godoc
// @Summary Bulk update user roles
// @Description Update the role of multiple users at once (admin only).
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body bulkUpdateRoleReq true "Bulk Update Details"
// @Success 200 {string} string "OK"
// @Router /users/bulk-role [post]
func (h *Handler) BulkUpdateUserRole(w http.ResponseWriter, r *http.Request) {
	userIDVal := r.Context().Value("user_id")
	if userIDVal == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	adminID := userIDVal.(int64)

	var req bulkUpdateRoleReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := h.svc.BulkUpdateRole(r.Context(), adminID, req.IDs, req.Role, req.Permissions, req.OTP, req.Password)
	if err != nil {
		slog.Error("Admin failed to bulk update roles", "admin_id", adminID, "count", len(req.IDs), "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// ListUsers godoc
// @Summary List all users
// @Description Retrieve a list of all registered users (admin only).
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Success 200 {array} domain.User
// @Router /users [get]
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.svc.ListUsers(r.Context())
	if err != nil {
		slog.Error("Failed to list users", "error", err)
		http.Error(w, "failed to list users", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

// GetUserByID godoc
// @Summary Get any user profile by ID
// @Description Retrieve profile details of any user (admin/moderator only).
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Success 200 {object} domain.User
// @Router /users/{id} [get]
func (h *Handler) GetUserByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	userID, _ := strconv.ParseInt(idStr, 10, 64)

	user, err := h.svc.GetProfile(r.Context(), userID)
	if err != nil || user == nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
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

type socialLoginReq struct {
	FullName  string  `json:"full_name"`
	Email     string  `json:"email"`
	SocialID  string  `json:"social_id"`
	AvatarURL *string `json:"avatar_url"`
}

// SocialLogin godoc
// @Summary Login with social provider
// @Description Authenticate or register a user using social provider data (e.g. Google).
// @Tags users
// @Accept json
// @Produce json
// @Param body body socialLoginReq true "Social Login Details"
// @Success 200 {object} map[string]any
// @Router /users/social-login [post]
func (h *Handler) SocialLogin(w http.ResponseWriter, r *http.Request) {
	var req socialLoginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	u := &domain.User{
		FullName:  req.FullName,
		Email:     req.Email,
		SocialID:  &req.SocialID,
		AvatarURL: req.AvatarURL,
	}

	token, user, err := h.svc.SocialLogin(r.Context(), u)
	if err != nil {
		slog.Error("Social login failed", "email", req.Email, "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"token": token,
		"user":  user,
	})
}

type otpRequest struct {
	Purpose string `json:"purpose"` // e.g., "password", "email", "phone"
}

// RequestOTP godoc
// @Summary Request an OTP
// @Description Send a 6-digit OTP to user's email for a specific purpose (password, email, phone).
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body otpRequest true "OTP Purpose"
// @Success 200 {string} string "OK"
// @Router /users/otp/request [post]
func (h *Handler) RequestOTP(w http.ResponseWriter, r *http.Request) {
	var req otpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	userIDVal := r.Context().Value("user_id")
	if userIDVal == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	userID := userIDVal.(int64)

	err := h.svc.RequestOTP(r.Context(), userID, req.Purpose)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

type otpVerifyReq struct {
	Purpose string `json:"purpose"`
	OTP     string `json:"otp"`
}

func (h *Handler) VerifyOTPOnly(w http.ResponseWriter, r *http.Request) {
	var req otpVerifyReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	userIDVal := r.Context().Value("user_id")
	if userIDVal == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	userID := userIDVal.(int64)

	valid, err := h.svc.CheckOTP(r.Context(), userID, req.Purpose, req.OTP)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !valid {
		http.Error(w, "Invalid or expired OTP", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

type secureUpdateReq struct {
	OTP      string  `json:"otp"`
	Purpose  string  `json:"purpose"`
	Password *string `json:"password"`
	Email    *string `json:"email"`
	Phone    *string `json:"phone"`
}

// SecureUpdate godoc
// @Summary Securely update sensitive info
// @Description Update password, email, or phone after OTP verification.
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body secureUpdateReq true "Update details and OTP"
// @Success 204 {string} string "No Content"
// @Router /users/secure-update [patch]
func (h *Handler) SecureUpdate(w http.ResponseWriter, r *http.Request) {
	var req secureUpdateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	userIDVal := r.Context().Value("user_id")
	if userIDVal == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	userID := userIDVal.(int64)

	// 1. Verify OTP
	valid, err := h.svc.VerifyOTP(r.Context(), userID, req.Purpose, req.OTP)
	if err != nil {
		http.Error(w, "Server error during verification", http.StatusInternalServerError)
		return
	}
	if !valid {
		http.Error(w, "Invalid or expired OTP", http.StatusUnauthorized)
		return
	}

	// 2. Execute update based on purpose
	switch req.Purpose {
	case "password":
		if req.Password == nil {
			http.Error(w, "password is required", http.StatusBadRequest)
			return
		}
		err = h.svc.UpdatePassword(r.Context(), userID, *req.Password)
	case "email":
		err = h.svc.UpdateProfile(r.Context(), userID, "", req.Email, nil, nil)
	case "phone":
		err = h.svc.UpdateProfile(r.Context(), userID, "", nil, req.Phone, nil)
	default:
		http.Error(w, "invalid purpose", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type forgotPasswordReq struct {
	Email string `json:"email"`
}

// ForgotPassword godoc
// @Summary Request password reset
// @Description Send a reset code to the user's email if they forgot their password.
// @Tags users
// @Accept json
// @Produce json
// @Param body body forgotPasswordReq true "User Email"
// @Success 200 {string} string "OK"
// @Router /users/forgot-password [post]
func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req forgotPasswordReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if err := h.svc.ForgotPassword(r.Context(), req.Email); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

type resetPasswordReq struct {
	Email    string `json:"email"`
	OTP      string `json:"otp"`
	Password string `json:"password"`
}

// ResetPassword godoc
// @Summary Reset password with OTP
// @Description Reset password using the code sent via ForgotPassword. Returns a new JWT token.
// @Tags users
// @Accept json
// @Produce json
// @Param body body resetPasswordReq true "Reset Details"
// @Success 200 {object} map[string]any
// @Router /users/reset-password [post]
func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req resetPasswordReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	token, user, err := h.svc.ResetPassword(r.Context(), req.Email, req.OTP, req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"token": token,
		"user":  user,
	})
}

// DeleteUser godoc
// @Summary Delete a user account
// @Description Remove a user account and all associated data (admin only).
// @Tags admin
// @Security BearerAuth
// @Param id path int true "User ID"
// @Success 200 {string} string "OK"
// @Router /users/{id} [delete]
func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	userID, _ := strconv.ParseInt(idStr, 10, 64)

	if err := h.svc.DeleteUser(r.Context(), userID); err != nil {
		slog.Error("Failed to delete user", "id", userID, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("User deleted successfully"))
}
