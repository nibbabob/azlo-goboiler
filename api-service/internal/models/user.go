// File: internal/models/user.go
package models

import (
	"time"
)

// User represents a user in the system
type User struct {
	ID           string     `json:"id" db:"id"`
	Username     string     `json:"username" db:"username"`
	Email        string     `json:"email" db:"email"`
	PasswordHash string     `json:"-" db:"password_hash"` // Never serialize to JSON
	IsActive     bool       `json:"is_active" db:"is_active"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
	LastLogin    *time.Time `json:"last_login,omitempty" db:"last_login"`
}

type UserPreferences struct {
	UserID       string `json:"-" db:"user_id"`
	EmailEnabled bool   `json:"email_enabled" db:"email_enabled"`
	Frequency    string `json:"frequency" db:"frequency"` // e.g., "immediate", "daily"
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Password string `json:"password" validate:"required,min=8,max=128"`
}

// RegisterRequest represents a user registration request
type RegisterRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50,alphanum"`
	Email    string `json:"email" validate:"required,email,max=100"`
	Password string `json:"password" validate:"required,min=8,max=128,password"`
}

// UpdateUserRequest represents a user update request
type UpdateUserRequest struct {
	Username *string `json:"username,omitempty" validate:"omitempty,min=3,max=50,alphanum"`
	Email    *string `json:"email,omitempty" validate:"omitempty,email,max=100"`
}

// ChangePasswordRequest represents a password change request
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8,max=128,password"`
}
