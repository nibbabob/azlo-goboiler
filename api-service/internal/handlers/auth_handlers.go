package handlers

import (
	"azlo-goboiler/internal/models"
	"azlo-goboiler/internal/validation"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

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
	err := h.app.DB.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM auth.users WHERE email = $1 OR username = $2)", req.Email, req.Username).Scan(&exists)
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
		INSERT INTO auth.users (id, username, email, password_hash, created_at, updated_at, is_active) 
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
		FROM auth.users 
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
	_, err = h.app.DB.Exec(ctx, "UPDATE auth.users SET last_login = $1 WHERE id = $2", time.Now(), user.ID)
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

	// Set the secure, HttpOnly cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "jwt_token",
		Value:    tokenString,
		Expires:  expirationTime,
		HttpOnly: true,                 // Most important: Prevents JS access
		Secure:   true,                 // Only send over HTTPS
		Path:     "/",                  // Available to entire site
		SameSite: http.SameSiteLaxMode, // Good security default
	})

	// Return success response without the token
	writeSuccess(w, h.app, map[string]interface{}{
		"expires_at": expirationTime.Unix(),
		"user": map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
		},
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
