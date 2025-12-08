package service

import (
	"azlo-goboiler/internal/config"
	"azlo-goboiler/internal/core"
	"azlo-goboiler/internal/models"
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	repo   core.UserRepository
	config *config.Config
}

func NewUserService(repo core.UserRepository, cfg *config.Config) core.UserService {
	return &UserService{repo: repo, config: cfg}
}

// --- Auth Methods (Already Implemented) ---
func (s *UserService) Register(ctx context.Context, req models.RegisterRequest) (*models.RegisterResponse, error) {
	existing, err := s.repo.GetByEmailOrUsername(ctx, req.Email, req.Username)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errors.New("user with this email or username already exists")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	newUser := &models.User{
		ID: uuid.New().String(), Username: req.Username, Email: req.Email,
		PasswordHash: string(hashedPassword), IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}

	if err := s.repo.Create(ctx, newUser); err != nil {
		return nil, err
	}
	return &models.RegisterResponse{UserID: newUser.ID, Username: newUser.Username, Email: newUser.Email}, nil
}

func (s *UserService) Login(ctx context.Context, req models.LoginRequest) (*models.LoginResponse, error) {
	user, err := s.repo.GetByEmailOrUsername(ctx, req.Username, req.Username)
	if err != nil || user == nil {
		return nil, errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	_ = s.repo.UpdateLastLogin(ctx, user.ID)

	expirationTime := time.Now().Add(s.config.GetJWTExpiration())
	claims := &jwt.RegisteredClaims{
		Subject: user.ID, ExpiresAt: jwt.NewNumericDate(expirationTime),
		IssuedAt: jwt.NewNumericDate(time.Now()), Issuer: "go-api-boilerplate",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.config.App_Secret))
	if err != nil {
		return nil, err
	}

	return &models.LoginResponse{
		Token: tokenString, ExpiresAt: expirationTime.Unix(),
		User: models.UserSummary{ID: user.ID, Username: user.Username, Email: user.Email},
	}, nil
}

// --- User Management Methods ---

func (s *UserService) GetProfile(ctx context.Context, userID string) (*models.User, error) {
	return s.repo.GetByID(ctx, userID)
}

func (s *UserService) UpdateProfile(ctx context.Context, userID string, req models.UpdateUserRequest) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	// Apply updates
	if req.Username != nil {
		user.Username = *req.Username
	}
	if req.Email != nil {
		user.Email = *req.Email
	}

	return s.repo.Update(ctx, user)
}

func (s *UserService) ChangePassword(ctx context.Context, userID string, req models.ChangePasswordRequest) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	// Verify old password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		return errors.New("current password is incorrect")
	}

	// Hash new password
	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return s.repo.UpdatePassword(ctx, userID, string(newHash))
}

func (s *UserService) GetUsers(ctx context.Context, page, limit int) ([]models.User, *models.PaginationMetadata, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	offset := (page - 1) * limit

	users, err := s.repo.List(ctx, limit, offset)
	if err != nil {
		return nil, nil, err
	}

	totalCount, err := s.repo.Count(ctx)
	if err != nil {
		return nil, nil, err
	}

	totalPages := (totalCount + limit - 1) / limit

	meta := &models.PaginationMetadata{
		Page:       page,
		Limit:      limit,
		TotalCount: totalCount,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}

	return users, meta, nil
}

// --- Preferences Methods ---

func (s *UserService) GetPreferences(ctx context.Context, userID string) (*models.UserPreferences, error) {
	prefs, err := s.repo.GetPreferences(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Return defaults if none found
	if prefs == nil {
		return &models.UserPreferences{UserID: userID, EmailEnabled: false, Frequency: "immediate"}, nil
	}
	return prefs, nil
}

func (s *UserService) UpdatePreferences(ctx context.Context, userID string, req models.UserPreferences) error {
	req.UserID = userID // Ensure ID is set from context
	return s.repo.UpsertPreferences(ctx, &req)
}
