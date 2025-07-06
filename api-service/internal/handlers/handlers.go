// File: internal/handlers/handlers.go
package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"azlo-goboiler/internal/config"

	"github.com/golang-jwt/jwt/v5"
)

type Handlers struct {
	app *config.Application
}

func New(app *config.Application) *Handlers {
	return &Handlers{app: app}
}

var startTime = time.Now()

// Health handles health check requests.
func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	healthCtx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	dbStatus := "connected"
	if err := h.app.DB.Ping(healthCtx); err != nil {
		dbStatus = "disconnected"
		h.app.Logger.Error().Err(err).Msg("Database health check failed")
	}

	redisStatus := "connected"
	if _, err := h.app.Redis.Ping(healthCtx).Result(); err != nil {
		redisStatus = "disconnected"
		h.app.Logger.Error().Err(err).Msg("Redis health check failed")
	}

	health := map[string]interface{}{
		"status":      "healthy",
		"timestamp":   time.Now().UTC(),
		"uptime":      time.Since(startTime).String(),
		"version":     "1.3.0",
		"environment": h.app.Config.App_Env,
		"services": map[string]string{
			"database": dbStatus,
			"redis":    redisStatus,
		},
	}
	writeSuccess(w, h.app, health, "Service is healthy")
}

// Auth handles user authentication and token generation.
func (h *Handlers) Auth(w http.ResponseWriter, r *http.Request) {
	var creds struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		writeError(w, h.app, http.StatusBadRequest, "Invalid request body")
		return
	}

	// For testing purposes. In a real app, replace with DB lookup and bcrypt comparison.
	if creds.Username != "user" || creds.Password != "password" {
		writeError(w, h.app, http.StatusUnauthorized, "Invalid credentials")
		return
	}
	userID := "user123"

	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &jwt.RegisteredClaims{
		Subject:   userID,
		ExpiresAt: jwt.NewNumericDate(expirationTime),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(h.app.Config.App_Secret))
	if err != nil {
		h.app.Logger.Error().Err(err).Msg("Failed to generate JWT token")
		writeError(w, h.app, http.StatusInternalServerError, "Could not generate token")
		return
	}

	writeSuccess(w, h.app, map[string]string{"token": tokenString}, "Authentication successful")
}

// Protected is an example of a handler for a protected route.
func (h *Handlers) Protected(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(config.UserIDKey).(string)
	if !ok {
		writeError(w, h.app, http.StatusInternalServerError, "Could not process user information")
		return
	}
	data := map[string]interface{}{
		"message": "This is a protected endpoint",
		"userID":  userID,
	}
	writeSuccess(w, h.app, data, "Access granted")
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, app *config.Application, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		app.Logger.Error().Err(err).Msg("Failed to write JSON response")
	}
}

func writeSuccess(w http.ResponseWriter, app *config.Application, data interface{}, message string) {
	writeJSON(w, app, http.StatusOK, map[string]interface{}{"success": true, "data": data, "message": message})
}

func writeError(w http.ResponseWriter, app *config.Application, status int, message string) {
	writeJSON(w, app, status, map[string]interface{}{"success": false, "error": message})
}
