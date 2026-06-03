package settings

import (
	"encoding/json"
	"eraya/domain"
	"eraya/infra/storage"
	middleware "eraya/rest/middlewares"
	"eraya/settings"
	"eraya/user"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc     settings.Service
	storage *storage.StorageService
}

func NewHandler(svc settings.Service, storage *storage.StorageService) *Handler {
	return &Handler{svc: svc, storage: storage}
}

func RegisterRoutes(r chi.Router, h *Handler, jwtSecret string, userSvc user.Service) {
	r.Route("/settings", func(r chi.Router) {
		r.Get("/", h.GetSettings)

		r.With(
			middleware.AuthMiddleware(jwtSecret, userSvc),
			middleware.RoleMiddleware("admin", "moderator"),
		).Group(func(r chi.Router) {
			r.Put("/", h.UpdateSettings)
			r.Post("/logo", h.UploadLogo)
		})
	})
}

// GetSettings godoc
// @Summary Get store settings
// @Description Retrieve global store configuration such as banners and metadata.
// @Tags admin
// @Produce json
// @Success 200 {object} domain.StoreSettings
// @Router /settings [get]
func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
	s, err := h.svc.GetSettings(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s)
}

// UpdateSettings godoc
// @Summary Update store settings (Admin)
// @Description Update global store configuration (admin/moderator only).
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body domain.StoreSettings true "Updated Settings"
// @Success 200 {object} map[string]string
// @Router /settings [put]
func (h *Handler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var s domain.StoreSettings
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := h.svc.UpdateSettings(r.Context(), &s)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Settings updated successfully"})
}

// UploadLogo godoc
// @Summary Upload store logo (Admin)
// @Tags admin
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param image formData file true "Logo image file"
// @Success 200 {object} map[string]string
// @Router /settings/logo [post]
func (h *Handler) UploadLogo(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(2 << 20); err != nil {
		http.Error(w, "failed to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "image is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	url, err := h.storage.UploadFile("logos", header.Filename, file, header.Header.Get("Content-Type"))
	if err != nil {
		http.Error(w, "upload failed", http.StatusInternalServerError)
		return
	}

	// Delete old logo if provided
	if oldURL := r.FormValue("old_url"); oldURL != "" {
		_ = h.storage.DeleteFile(oldURL) // non-fatal
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"url": url})
}
