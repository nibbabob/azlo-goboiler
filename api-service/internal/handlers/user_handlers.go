package handlers

import (
	"azlo-goboiler/internal/config"
	"azlo-goboiler/internal/models"
	"azlo-goboiler/internal/validation"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/crypto/bcrypt"
)

// Protected is an example of a handler for a protected route
func (h *Handlers) Protected(w http.ResponseWriter, r *http.Request) {
	// Get the global tracer
	tracer := otel.Tracer("handlers") // Or use the tracer provider from h.app

	// Start a new span using the request context
	baseCtx := r.Context()
	ctx, span := tracer.Start(baseCtx, "Handlers.Protected") // Use r.Context()
	defer span.End()                                         // Ensure the span is always ended

	// Extract IDs from the original request context
	requestID := getRequestID(baseCtx)
	userID, ok := baseCtx.Value(config.UserIDKey).(string)
	if !ok {
		h.app.Logger.Error().
			Str("request_id", requestID).
			Msg("Could not extract user ID from context")
		writeError(w, h.app, http.StatusInternalServerError, "Authentication error")
		return
	}
	span.SetAttributes(attribute.String("user.id", userID))

	// Get user details
	// Use the span-context for the database call, but wrap it in a timeout
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var user models.User
	err := h.app.DB.QueryRow(dbCtx, ` // Use dbCtx here
		SELECT id, username, email, created_at, last_login 
		FROM auth.users 
		WHERE id = $1 AND is_active = true`,
		userID).Scan(&user.ID, &user.Username, &user.Email, &user.CreatedAt, &user.LastLogin)

	if err != nil {
		h.app.Logger.Error().
			Str("request_id", requestID).
			Str("user_id", userID).
			Err(err).
			Msg("Failed to fetch user details")
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
	requestID := getRequestID(r.Context())

	// Parse pagination parameters
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 10
	}

	offset := (page - 1) * limit

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Get total count
	var totalCount int
	err := h.app.DB.QueryRow(ctx, "SELECT COUNT(*) FROM auth.users WHERE is_active = true").Scan(&totalCount)
	if err != nil {
		h.app.Logger.Error().
			Str("request_id", requestID).
			Err(err).
			Msg("Failed to count users")
		writeError(w, h.app, http.StatusInternalServerError, "Failed to fetch users")
		return
	}

	// Get users
	rows, err := h.app.DB.Query(ctx, `
		SELECT id, username, email, created_at, last_login 
		FROM auth.users 
		WHERE is_active = true 
		ORDER BY created_at DESC 
		LIMIT $1 OFFSET $2`, limit, offset)

	if err != nil {
		h.app.Logger.Error().
			Str("request_id", requestID).
			Err(err).
			Msg("Failed to query users")
		writeError(w, h.app, http.StatusInternalServerError, "Failed to fetch users")
		return
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(&user.ID, &user.Username, &user.Email, &user.CreatedAt, &user.LastLogin)
		if err != nil {
			h.app.Logger.Error().
				Str("request_id", requestID).
				Err(err).
				Msg("Failed to scan user row")
			continue
		}
		users = append(users, user)
	}

	totalPages := (totalCount + limit - 1) / limit

	data := map[string]interface{}{
		"users": users,
		"pagination": map[string]interface{}{
			"page":        page,
			"limit":       limit,
			"total_count": totalCount,
			"total_pages": totalPages,
			"has_next":    page < totalPages,
			"has_prev":    page > 1,
		},
	}

	writeSuccess(w, h.app, data, "Users retrieved successfully")
}

// GetProfile handles GET /api/v1/profile
func (h *Handlers) GetProfile(w http.ResponseWriter, r *http.Request) {
	requestID := getRequestID(r.Context())
	userID, ok := r.Context().Value(config.UserIDKey).(string)
	if !ok {
		h.app.Logger.Error().
			Str("request_id", requestID).
			Msg("Could not extract user ID from context")
		writeError(w, h.app, http.StatusInternalServerError, "Authentication error")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var user models.User
	err := h.app.DB.QueryRow(ctx, `
		SELECT id, username, email, created_at, updated_at, last_login 
		FROM auth.users 
		WHERE id = $1 AND is_active = true`,
		userID).Scan(&user.ID, &user.Username, &user.Email, &user.CreatedAt, &user.UpdatedAt, &user.LastLogin)

	if err != nil {
		h.app.Logger.Error().
			Str("request_id", requestID).
			Str("user_id", userID).
			Err(err).
			Msg("Failed to fetch user profile")
		writeError(w, h.app, http.StatusInternalServerError, "Failed to fetch profile")
		return
	}

	writeSuccess(w, h.app, user, "Profile retrieved successfully")
}

// UpdateProfile handles PUT /api/v1/profile
func (h *Handlers) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	requestID := getRequestID(r.Context())
	userID, ok := r.Context().Value(config.UserIDKey).(string)
	if !ok {
		writeError(w, h.app, http.StatusInternalServerError, "Authentication error")
		return
	}

	var req models.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, h.app, http.StatusBadRequest, "Invalid request format")
		return
	}

	if err := validation.ValidateStruct(&req); err != nil {
		writeError(w, h.app, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Build dynamic update query
	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.Username != nil {
		setParts = append(setParts, fmt.Sprintf("username = $%d", argIndex))
		args = append(args, *req.Username)
		argIndex++
	}

	if req.Email != nil {
		setParts = append(setParts, fmt.Sprintf("email = $%d", argIndex))
		args = append(args, *req.Email)
		argIndex++
	}

	if len(setParts) == 0 {
		writeError(w, h.app, http.StatusBadRequest, "No fields to update")
		return
	}

	setParts = append(setParts, fmt.Sprintf("updated_at = $%d", argIndex))
	args = append(args, time.Now())
	argIndex++

	args = append(args, userID)
	query := fmt.Sprintf("UPDATE auth.users SET %s WHERE id = $%d AND is_active = true",
		strings.Join(setParts, ", "), argIndex)

	result, err := h.app.DB.Exec(ctx, query, args...)
	if err != nil {
		h.app.Logger.Error().
			Str("request_id", requestID).
			Str("user_id", userID).
			Err(err).
			Msg("Failed to update user profile")
		writeError(w, h.app, http.StatusInternalServerError, "Failed to update profile")
		return
	}

	if result.RowsAffected() == 0 {
		writeError(w, h.app, http.StatusNotFound, "User not found")
		return
	}

	writeSuccess(w, h.app, map[string]string{"user_id": userID}, "Profile updated successfully")
}

// ChangePassword handles PUT /api/v1/password
func (h *Handlers) ChangePassword(w http.ResponseWriter, r *http.Request) {
	requestID := getRequestID(r.Context())
	userID, ok := r.Context().Value(config.UserIDKey).(string)
	if !ok {
		writeError(w, h.app, http.StatusInternalServerError, "Authentication error")
		return
	}

	var req models.ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, h.app, http.StatusBadRequest, "Invalid request format")
		return
	}

	if err := validation.ValidateStruct(&req); err != nil {
		writeError(w, h.app, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Get current password hash
	var currentHash string
	err := h.app.DB.QueryRow(ctx, "SELECT password_hash FROM auth.users WHERE id = $1 AND is_active = true", userID).Scan(&currentHash)
	if err != nil {
		writeError(w, h.app, http.StatusInternalServerError, "Failed to verify current password")
		return
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(req.CurrentPassword)); err != nil {
		writeError(w, h.app, http.StatusUnauthorized, "Current password is incorrect")
		return
	}

	// Hash new password
	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		h.app.Logger.Error().
			Str("request_id", requestID).
			Str("user_id", userID).
			Err(err).
			Msg("Failed to hash new password")
		writeError(w, h.app, http.StatusInternalServerError, "Failed to update password")
		return
	}

	// Update password
	_, err = h.app.DB.Exec(ctx, "UPDATE auth.users SET password_hash = $1, updated_at = $2 WHERE id = $3",
		string(newHash), time.Now(), userID)
	if err != nil {
		h.app.Logger.Error().
			Str("request_id", requestID).
			Str("user_id", userID).
			Err(err).
			Msg("Failed to update password")
		writeError(w, h.app, http.StatusInternalServerError, "Failed to update password")
		return
	}

	h.app.Logger.Info().
		Str("request_id", requestID).
		Str("user_id", userID).
		Msg("Password changed successfully")

	writeSuccess(w, h.app, nil, "Password updated successfully")
}

func (h *Handlers) GetPreferences(w http.ResponseWriter, r *http.Request) {

	userID := r.Context().Value(config.UserIDKey).(string)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var prefs models.UserPreferences
	// Use COALESCE to return defaults if no row exists yet
	query := `
		SELECT email_enabled, frequency
		FROM auth.user_preferences
		WHERE user_id = $1`

	err := h.app.DB.QueryRow(ctx, query, userID).Scan(
		&prefs.EmailEnabled, &prefs.Frequency,
	)

	if err != nil {
		// If no rows, return defaults instead of error
		h.app.Logger.Debug().Str("user_id", userID).Msg("No preferences found, returning defaults")
		prefs = models.UserPreferences{
			EmailEnabled: false,
			Frequency:    "immediate",
		}
	}

	writeSuccess(w, h.app, prefs, "Preferences retrieved successfully")
}

// UpdatePreferences handles PUT /api/v1/preferences
func (h *Handlers) UpdatePreferences(w http.ResponseWriter, r *http.Request) {
	requestID := getRequestID(r.Context())
	userID := r.Context().Value(config.UserIDKey).(string)

	var req models.UserPreferences
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, h.app, http.StatusBadRequest, "Invalid request format")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Upsert (Insert or Update)
	query := `
        INSERT INTO auth.user_preferences (user_id, email_enabled, frequency, updated_at)
        VALUES ($1, $2, $3, $4, $5, NOW())
        ON CONFLICT (user_id) DO UPDATE SET
            email_enabled = EXCLUDED.email_enabled,
            frequency = EXCLUDED.frequency,
            updated_at = NOW()`

	_, err := h.app.DB.Exec(ctx, query,
		userID, req.EmailEnabled, req.Frequency,
	)

	if err != nil {
		h.app.Logger.Error().Err(err).Str("request_id", requestID).Msg("Failed to update preferences")
		writeError(w, h.app, http.StatusInternalServerError, "Failed to update preferences")
		return
	}

	writeSuccess(w, h.app, req, "Preferences updated successfully")
}
