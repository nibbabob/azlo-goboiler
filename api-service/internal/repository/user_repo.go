package repository

import (
	"azlo-goboiler/internal/core"
	"azlo-goboiler/internal/models"
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresUserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) core.UserRepository {
	return &PostgresUserRepository{db: db}
}

// --- Auth & Basic ---

func (r *PostgresUserRepository) Create(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO auth.users (id, username, email, password_hash, created_at, updated_at, is_active) 
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.db.Exec(ctx, query,
		user.ID, user.Username, user.Email, user.PasswordHash, user.CreatedAt, user.UpdatedAt, user.IsActive)
	return err
}

func (r *PostgresUserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	var user models.User
	query := `
		SELECT id, username, email, password_hash, is_active, created_at, updated_at, last_login 
		FROM auth.users WHERE id = $1 AND is_active = true`
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.IsActive, &user.CreatedAt, &user.UpdatedAt, &user.LastLogin)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

func (r *PostgresUserRepository) GetByEmailOrUsername(ctx context.Context, email, username string) (*models.User, error) {
	var user models.User
	query := `
		SELECT id, username, email, password_hash, is_active, created_at, updated_at 
		FROM auth.users WHERE (username = $1 OR email = $2) AND is_active = true`
	err := r.db.QueryRow(ctx, query, username, email).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.IsActive, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// --- User Management ---

func (r *PostgresUserRepository) Update(ctx context.Context, user *models.User) error {
	query := `
		UPDATE auth.users 
		SET username = $1, email = $2, updated_at = $3
		WHERE id = $4 AND is_active = true`
	_, err := r.db.Exec(ctx, query, user.Username, user.Email, time.Now(), user.ID)
	return err
}

func (r *PostgresUserRepository) UpdatePassword(ctx context.Context, userID, hash string) error {
	_, err := r.db.Exec(ctx, "UPDATE auth.users SET password_hash = $1, updated_at = $2 WHERE id = $3", hash, time.Now(), userID)
	return err
}

func (r *PostgresUserRepository) UpdateLastLogin(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, "UPDATE auth.users SET last_login = $1 WHERE id = $2", time.Now(), userID)
	return err
}

func (r *PostgresUserRepository) List(ctx context.Context, limit, offset int) ([]models.User, error) {
	query := `
		SELECT id, username, email, created_at, last_login 
		FROM auth.users WHERE is_active = true 
		ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		if err := rows.Scan(&user.ID, &user.Username, &user.Email, &user.CreatedAt, &user.LastLogin); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

func (r *PostgresUserRepository) Count(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM auth.users WHERE is_active = true").Scan(&count)
	return count, err
}

// --- Preferences ---

func (r *PostgresUserRepository) GetPreferences(ctx context.Context, userID string) (*models.UserPreferences, error) {
	var prefs models.UserPreferences
	query := `SELECT email_enabled, frequency FROM auth.user_preferences WHERE user_id = $1`
	err := r.db.QueryRow(ctx, query, userID).Scan(&prefs.EmailEnabled, &prefs.Frequency)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Return nil to indicate no preferences set
		}
		return nil, err
	}
	// Important: Set UserID since it's not retrieved from the DB row directly
	prefs.UserID = userID
	return &prefs, nil
}

func (r *PostgresUserRepository) UpsertPreferences(ctx context.Context, prefs *models.UserPreferences) error {
	query := `
		INSERT INTO auth.user_preferences (user_id, email_enabled, frequency, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			email_enabled = EXCLUDED.email_enabled,
			frequency = EXCLUDED.frequency,
			updated_at = NOW()`
	_, err := r.db.Exec(ctx, query, prefs.UserID, prefs.EmailEnabled, prefs.Frequency)
	return err
}
