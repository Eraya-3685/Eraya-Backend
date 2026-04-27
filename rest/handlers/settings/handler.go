package settings

import (
	"encoding/json"
	"eraya/domain"
	middleware "eraya/rest/middlewares"
	"eraya/settings"
	"eraya/user"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc settings.Service
}

func NewHandler(svc settings.Service) *Handler {
	return &Handler{svc: svc}
}

func RegisterRoutes(r chi.Router, h *Handler, jwtSecret string, userSvc user.Service) {
	r.Route("/settings", func(r chi.Router) {
		r.Get("/", h.GetSettings)

		r.With(
			middleware.AuthMiddleware(jwtSecret, userSvc),
			middleware.RoleMiddleware("admin", "moderator"),
		).Put("/", h.UpdateSettings)
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
