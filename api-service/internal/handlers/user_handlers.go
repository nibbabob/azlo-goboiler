package handlers

import (
	"azlo-goboiler/internal/config"
	"azlo-goboiler/internal/models"
	"azlo-goboiler/internal/validation"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

// Protected verifies the user can access the route and returns their profile info
func (h *Handlers) Protected(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("handlers")
	ctx, span := tracer.Start(r.Context(), "Handlers.Protected")
	defer span.End()

	requestID := getRequestID(ctx)
	userID, ok := ctx.Value(config.UserIDKey).(string)
	if !ok {
		writeError(w, h.app, http.StatusInternalServerError, "Authentication error")
		return
	}
	span.SetAttributes(attribute.String("user.id", userID))

	user, err := h.service.GetProfile(ctx, userID)
	if err != nil {
		h.app.Logger.Error().Str("request_id", requestID).Err(err).Msg("Failed to fetch user")
		writeError(w, h.app, http.StatusInternalServerError, "Failed to fetch user information")
		return
	}

	data := map[string]interface{}{
		"message":     "This is a protected endpoint",
		"user":        user,
		"access_time": time.Now().UTC(),
	}
	writeSuccess(w, h.app, data, "Access granted")
}

// GetUsers handles GET /api/v1/users with pagination
func (h *Handlers) GetUsers(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	users, meta, err := h.service.GetUsers(r.Context(), page, limit)
	if err != nil {
		h.app.Logger.Error().Err(err).Msg("Failed to fetch users")
		writeError(w, h.app, http.StatusInternalServerError, "Failed to fetch users")
		return
	}

	writeSuccess(w, h.app, map[string]interface{}{
		"users":      users,
		"pagination": meta,
	}, "Users retrieved successfully")
}

// GetProfile handles GET /api/v1/profile
func (h *Handlers) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(config.UserIDKey).(string)

	user, err := h.service.GetProfile(r.Context(), userID)
	if err != nil {
		writeError(w, h.app, http.StatusNotFound, "User not found")
		return
	}

	writeSuccess(w, h.app, user, "Profile retrieved successfully")
}

// UpdateProfile handles PUT /api/v1/profile
func (h *Handlers) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(config.UserIDKey).(string)

	var req models.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, h.app, http.StatusBadRequest, "Invalid request format")
		return
	}

	if err := validation.ValidateStruct(&req); err != nil {
		writeError(w, h.app, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.service.UpdateProfile(r.Context(), userID, req); err != nil {
		h.app.Logger.Error().Err(err).Msg("Failed to update profile")
		writeError(w, h.app, http.StatusInternalServerError, "Failed to update profile")
		return
	}

	writeSuccess(w, h.app, map[string]string{"user_id": userID}, "Profile updated successfully")
}

// ChangePassword handles PUT /api/v1/password
func (h *Handlers) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(config.UserIDKey).(string)

	var req models.ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, h.app, http.StatusBadRequest, "Invalid request format")
		return
	}

	if err := validation.ValidateStruct(&req); err != nil {
		writeError(w, h.app, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.service.ChangePassword(r.Context(), userID, req); err != nil {
		if err.Error() == "current password is incorrect" {
			writeError(w, h.app, http.StatusUnauthorized, err.Error())
			return
		}
		h.app.Logger.Error().Err(err).Msg("Failed to change password")
		writeError(w, h.app, http.StatusInternalServerError, "Failed to update password")
		return
	}

	writeSuccess(w, h.app, nil, "Password updated successfully")
}

// GetPreferences handles GET /api/v1/preferences
func (h *Handlers) GetPreferences(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(config.UserIDKey).(string)

	prefs, err := h.service.GetPreferences(r.Context(), userID)
	if err != nil {
		h.app.Logger.Error().Err(err).Msg("Failed to fetch preferences")
		writeError(w, h.app, http.StatusInternalServerError, "Failed to fetch preferences")
		return
	}

	writeSuccess(w, h.app, prefs, "Preferences retrieved successfully")
}

// UpdatePreferences handles PUT /api/v1/preferences
func (h *Handlers) UpdatePreferences(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(config.UserIDKey).(string)

	var req models.UserPreferences
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, h.app, http.StatusBadRequest, "Invalid request format")
		return
	}

	if err := h.service.UpdatePreferences(r.Context(), userID, req); err != nil {
		h.app.Logger.Error().Err(err).Msg("Failed to update preferences")
		writeError(w, h.app, http.StatusInternalServerError, "Failed to update preferences")
		return
	}

	writeSuccess(w, h.app, req, "Preferences updated successfully")
}
