package core

import (
	"azlo-goboiler/internal/models"
	"context"
)

// UserRepository defines direct database operations.
type UserRepository interface {
	// Auth & Basic
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id string) (*models.User, error)
	GetByEmailOrUsername(ctx context.Context, email, username string) (*models.User, error)

	// User Management
	Update(ctx context.Context, user *models.User) error
	UpdatePassword(ctx context.Context, userID, hash string) error
	UpdateLastLogin(ctx context.Context, userID string) error
	List(ctx context.Context, limit, offset int) ([]models.User, error)
	Count(ctx context.Context) (int, error)
}

// UserService defines the business logic.
type UserService interface {
	// Auth
	Register(ctx context.Context, req models.RegisterRequest) (*models.RegisterResponse, error)
	Login(ctx context.Context, req models.LoginRequest) (*models.LoginResponse, error)

	// User Management
	GetProfile(ctx context.Context, userID string) (*models.User, error)
	UpdateProfile(ctx context.Context, userID string, req models.UpdateUserRequest) error
	ChangePassword(ctx context.Context, userID string, req models.ChangePasswordRequest) error
	GetUsers(ctx context.Context, page, limit int) ([]models.User, *models.PaginationMetadata, error)
}
