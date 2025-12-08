package handlers

import (
	"azlo-goboiler/internal/models"
	"azlo-goboiler/internal/validation"
	"encoding/json"
	"net/http"
	"time"
)

// Register handles user registration via the Service layer
func (h *Handlers) Register(w http.ResponseWriter, r *http.Request) {
	requestID := getRequestID(r.Context())

	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.app.Logger.Warn().
			Str("request_id", requestID).
			Err(err).
			Msg("Invalid JSON in registration request")
		writeError(w, h.app, http.StatusBadRequest, "Invalid request format")
		return
	}

	// Validate input
	if err := validation.ValidateStruct(&req); err != nil {
		h.app.Logger.Warn().
			Str("request_id", requestID).
			Err(err).
			Msg("Registration validation failed")
		writeError(w, h.app, http.StatusBadRequest, err.Error())
		return
	}

	// Call Service Layer
	resp, err := h.service.Register(r.Context(), req)
	if err != nil {
		// Check for specific error messages to return correct status codes
		// In a more advanced setup, you would use custom error types here
		if err.Error() == "user with this email or username already exists" {
			writeError(w, h.app, http.StatusConflict, err.Error())
			return
		}

		h.app.Logger.Error().
			Str("request_id", requestID).
			Err(err).
			Msg("Registration failed")
		writeError(w, h.app, http.StatusInternalServerError, "Registration failed")
		return
	}

	h.app.Logger.Info().
		Str("request_id", requestID).
		Str("user_id", resp.UserID).
		Str("username", resp.Username).
		Msg("User registered successfully")

	writeSuccess(w, h.app, resp, "User registered successfully")
}

// Auth handles user authentication via the Service layer
func (h *Handlers) Auth(w http.ResponseWriter, r *http.Request) {
	requestID := getRequestID(r.Context())

	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.app.Logger.Warn().
			Str("request_id", requestID).
			Err(err).
			Msg("Invalid JSON in login request")
		writeError(w, h.app, http.StatusBadRequest, "Invalid request format")
		return
	}

	// Validate input
	if err := validation.ValidateStruct(&req); err != nil {
		h.app.Logger.Warn().
			Str("request_id", requestID).
			Err(err).
			Msg("Login validation failed")
		writeError(w, h.app, http.StatusBadRequest, err.Error())
		return
	}

	// Call Service Layer
	resp, err := h.service.Login(r.Context(), req)
	if err != nil {
		h.app.Logger.Warn().
			Str("request_id", requestID).
			Str("username", req.Username).
			Err(err).
			Msg("Login failed")
		writeError(w, h.app, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	h.app.Logger.Info().
		Str("request_id", requestID).
		Str("user_id", resp.User.ID).
		Str("username", resp.User.Username).
		Msg("User authenticated successfully")

	// Set the secure, HttpOnly cookie using the token from the service
	http.SetCookie(w, &http.Cookie{
		Name:     "jwt_token",
		Value:    resp.Token,
		Expires:  time.Unix(resp.ExpiresAt, 0),
		HttpOnly: true,                 // Prevents JS access
		Secure:   true,                 // Only send over HTTPS
		Path:     "/",                  // Available to entire site
		SameSite: http.SameSiteLaxMode, // Good security default
	})

	// Return success response without the token (it's in the cookie)
	writeSuccess(w, h.app, map[string]interface{}{
		"expires_at": resp.ExpiresAt,
		"user":       resp.User,
	}, "Authentication successful")
}

// Logout handles user logout by clearing the auth cookie
func (h *Handlers) Logout(w http.ResponseWriter, r *http.Request) {
	// Set the cookie to expire in the past
	http.SetCookie(w, &http.Cookie{
		Name:     "jwt_token",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour), // Expire in the past
		HttpOnly: true,
		Secure:   true,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	})

	writeSuccess(w, h.app, nil, "Logout successful")
}
