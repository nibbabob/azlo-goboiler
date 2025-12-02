// File: internal/database/seeder.go
package database

import (
	"context"
	"time"

	"azlo-goboiler/internal/config"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// SeedDefaultUser creates a default user for development environments.
func SeedDefaultUser(app *config.Application) {
	// Only seed in development environment
	if !app.Config.IsDevelopment() {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Check if the default user already exists
	var exists bool
	err := app.DB.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM auth.users WHERE username = $1)", app.Config.DefaultUserUsername).Scan(&exists)
	if err != nil {
		app.Logger.Error().Err(err).Msg("Failed to check for default user")
		return
	}

	if exists {
		app.Logger.Info().Str("username", app.Config.DefaultUserUsername).Msg("Default user already exists")
		return
	}

	// Hash the default password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(app.Config.DefaultUserPassword), bcrypt.DefaultCost)
	if err != nil {
		app.Logger.Error().Err(err).Msg("Failed to hash default user password")
		return
	}

	// Create the default user
	userID := uuid.New().String()
	now := time.Now()

	_, err = app.DB.Exec(ctx, `
		INSERT INTO auth.users (id, username, email, password_hash, created_at, updated_at, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		userID, app.Config.DefaultUserUsername, "defaultuser@example.com", string(hashedPassword), now, now, true)

	if err != nil {
		app.Logger.Error().Err(err).Msg("Failed to create default user")
		return
	}

	app.Logger.Info().Str("username", app.Config.DefaultUserUsername).Msg("Default user created successfully")
}
