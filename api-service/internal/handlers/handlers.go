// File: internal/handlers/handlers.go
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"azlo-goboiler/internal/config"
	"azlo-goboiler/internal/database"
	"azlo-goboiler/internal/models"
	"azlo-goboiler/internal/validation"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Handlers struct {
	app *config.Application
}

func New(app *config.Application) *Handlers {
	return &Handlers{app: app}
}

var startTime = time.Now()

// Health handles health check requests with enhanced diagnostics
func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	requestID := getRequestID(r.Context())
	healthCtx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	dbStatus := "connected"
	var dbLatency time.Duration
	dbStart := time.Now()
	if err := h.app.DB.Ping(healthCtx); err != nil {
		dbStatus = "disconnected"
		h.app.Logger.Error().
			Str("request_id", requestID).
			Err(err).
			Msg("Database health check failed")
	} else {
		dbLatency = time.Since(dbStart)
	}

	redisStatus := "connected"
	var redisLatency time.Duration
	redisStart := time.Now()
	if _, err := h.app.Redis.Ping(healthCtx).Result(); err != nil {
		redisStatus = "disconnected"
		h.app.Logger.Error().
			Str("request_id", requestID).
			Err(err).
			Msg("Redis health check failed")
	} else {
		redisLatency = time.Since(redisStart)
	}

	health := map[string]interface{}{
		"status":      "healthy",
		"timestamp":   time.Now().UTC(),
		"uptime":      time.Since(startTime).String(),
		"version":     "1.4.0",
		"environment": h.app.Config.App_Env,
		"request_id":  requestID,
		"services": map[string]interface{}{
			"database": map[string]interface{}{
				"status":  dbStatus,
				"latency": dbLatency.String(),
			},
			"redis": map[string]interface{}{
				"status":  redisStatus,
				"latency": redisLatency.String(),
			},
		},
	}

	if dbStatus == "disconnected" || redisStatus == "disconnected" {
		health["status"] = "degraded"
		writeResponse(w, h.app, http.StatusServiceUnavailable, false, health, "Service is degraded")
		return
	}

	writeSuccess(w, h.app, health, "Service is healthy")
}

// Register handles user registration
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

	// Check if user already exists
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var exists bool
	err := h.app.DB.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE email = $1 OR username = $2)", req.Email, req.Username).Scan(&exists)
	if err != nil {
		h.app.Logger.Error().
			Str("request_id", requestID).
			Err(err).
			Msg("Database error checking user existence")
		writeError(w, h.app, http.StatusInternalServerError, "Registration failed")
		return
	}

	if exists {
		writeError(w, h.app, http.StatusConflict, "User with this email or username already exists")
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		h.app.Logger.Error().
			Str("request_id", requestID).
			Err(err).
			Msg("Password hashing failed")
		writeError(w, h.app, http.StatusInternalServerError, "Registration failed")
		return
	}

	// Create user
	userID := uuid.New().String()
	now := time.Now()

	_, err = h.app.DB.Exec(ctx, `
		INSERT INTO users (id, username, email, password_hash, created_at, updated_at, is_active) 
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		userID, req.Username, req.Email, string(hashedPassword), now, now, true)

	if err != nil {
		h.app.Logger.Error().
			Str("request_id", requestID).
			Err(err).
			Msg("Database error creating user")
		writeError(w, h.app, http.StatusInternalServerError, "Registration failed")
		return
	}

	h.app.Logger.Info().
		Str("request_id", requestID).
		Str("user_id", userID).
		Str("username", req.Username).
		Msg("User registered successfully")

	writeSuccess(w, h.app, map[string]string{
		"user_id":  userID,
		"username": req.Username,
		"email":    req.Email,
	}, "User registered successfully")
}

// Auth handles user authentication with proper password verification
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

	// Get user from database
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var user models.User
	err := h.app.DB.QueryRow(ctx, `
		SELECT id, username, email, password_hash, is_active, created_at, updated_at 
		FROM users 
		WHERE (username = $1 OR email = $1) AND is_active = true`,
		req.Username).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.IsActive, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		h.app.Logger.Warn().
			Str("request_id", requestID).
			Str("username", req.Username).
			Msg("Login attempt with invalid credentials")
		writeError(w, h.app, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		h.app.Logger.Warn().
			Str("request_id", requestID).
			Str("user_id", user.ID).
			Msg("Login attempt with wrong password")
		writeError(w, h.app, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Update last login time
	_, err = h.app.DB.Exec(ctx, "UPDATE users SET last_login = $1 WHERE id = $2", time.Now(), user.ID)
	if err != nil {
		h.app.Logger.Warn().
			Str("request_id", requestID).
			Str("user_id", user.ID).
			Err(err).
			Msg("Failed to update last login time")
	}

	// Generate JWT token
	expirationTime := time.Now().Add(h.app.Config.GetJWTExpiration())
	claims := &jwt.RegisteredClaims{
		Subject:   user.ID,
		ExpiresAt: jwt.NewNumericDate(expirationTime),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Issuer:    "go-api-boilerplate",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(h.app.Config.App_Secret))
	if err != nil {
		h.app.Logger.Error().
			Str("request_id", requestID).
			Str("user_id", user.ID).
			Err(err).
			Msg("Failed to generate JWT token")
		writeError(w, h.app, http.StatusInternalServerError, "Authentication failed")
		return
	}

	h.app.Logger.Info().
		Str("request_id", requestID).
		Str("user_id", user.ID).
		Str("username", user.Username).
		Msg("User authenticated successfully")

	writeSuccess(w, h.app, map[string]interface{}{
		"token":      tokenString,
		"expires_at": expirationTime.Unix(),
		"user": map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
		},
	}, "Authentication successful")
}

// Protected is an example of a handler for a protected route
func (h *Handlers) Protected(w http.ResponseWriter, r *http.Request) {
	requestID := getRequestID(r.Context())
	userID, ok := r.Context().Value(config.UserIDKey).(string)
	if !ok {
		h.app.Logger.Error().
			Str("request_id", requestID).
			Msg("Could not extract user ID from context")
		writeError(w, h.app, http.StatusInternalServerError, "Authentication error")
		return
	}

	// Get user details
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var user models.User
	err := h.app.DB.QueryRow(ctx, `
		SELECT id, username, email, created_at, last_login 
		FROM users 
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
	err := h.app.DB.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE is_active = true").Scan(&totalCount)
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
		FROM users 
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
		FROM users 
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
	query := fmt.Sprintf("UPDATE users SET %s WHERE id = $%d AND is_active = true",
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
	err := h.app.DB.QueryRow(ctx, "SELECT password_hash FROM users WHERE id = $1 AND is_active = true", userID).Scan(&currentHash)
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
	_, err = h.app.DB.Exec(ctx, "UPDATE users SET password_hash = $1, updated_at = $2 WHERE id = $3",
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

// HealthDetailed provides detailed health information including database stats
func (h *Handlers) HealthDetailed(w http.ResponseWriter, r *http.Request) {
	requestID := getRequestID(r.Context())
	healthCtx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	health := map[string]interface{}{
		"status":      "healthy",
		"timestamp":   time.Now().UTC(),
		"uptime":      time.Since(startTime).String(),
		"version":     "1.4.0",
		"environment": h.app.Config.App_Env,
		"request_id":  requestID,
	}

	// Database health
	dbHealth := make(map[string]interface{})
	dbStart := time.Now()
	if err := database.HealthCheck(h.app.DB); err != nil {
		dbHealth["status"] = "unhealthy"
		dbHealth["error"] = err.Error()
		health["status"] = "degraded"
	} else {
		dbHealth["status"] = "healthy"
		dbHealth["latency"] = time.Since(dbStart).String()
		dbHealth["stats"] = database.GetConnectionStats(h.app.DB)
	}
	health["database"] = dbHealth

	// Redis health
	redisHealth := make(map[string]interface{})
	redisStart := time.Now()
	if _, err := h.app.Redis.Ping(healthCtx).Result(); err != nil {
		redisHealth["status"] = "unhealthy"
		redisHealth["error"] = err.Error()
		health["status"] = "degraded"
	} else {
		redisHealth["status"] = "healthy"
		redisHealth["latency"] = time.Since(redisStart).String()
	}
	health["redis"] = redisHealth

	statusCode := http.StatusOK
	if health["status"] == "degraded" {
		statusCode = http.StatusServiceUnavailable
	}

	writeResponse(w, h.app, statusCode, health["status"] == "healthy", health, "Detailed health check complete")
}

// GetDatabaseStats provides database statistics (admin only)
func (h *Handlers) GetDatabaseStats(w http.ResponseWriter, r *http.Request) {
	// In production, you might want to add admin role checking here
	stats := database.GetConnectionStats(h.app.DB)
	writeSuccess(w, h.app, stats, "Database statistics retrieved")
}

// --- Helper Functions ---

func getRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value("request_id").(string); ok {
		return requestID
	}
	return "unknown"
}

func writeJSON(w http.ResponseWriter, app *config.Application, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		app.Logger.Error().Err(err).Msg("Failed to write JSON response")
	}
}

func writeResponse(w http.ResponseWriter, app *config.Application, status int, success bool, data interface{}, message string) {
	response := map[string]interface{}{
		"success": success,
		"message": message,
	}

	if data != nil {
		response["data"] = data
	}

	if !success {
		response["error"] = message
	}

	writeJSON(w, app, status, response)
}

func writeSuccess(w http.ResponseWriter, app *config.Application, data interface{}, message string) {
	writeResponse(w, app, http.StatusOK, true, data, message)
}

func writeError(w http.ResponseWriter, app *config.Application, status int, message string) {
	writeResponse(w, app, status, false, nil, message)
}
