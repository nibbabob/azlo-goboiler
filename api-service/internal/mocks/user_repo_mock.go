package mocks

import (
	"azlo-goboiler/internal/models"
	"context"

	"github.com/stretchr/testify/mock"
)

// MockUserRepository is a mock implementation of core.UserRepository
type MockUserRepository struct {
	mock.Mock
}

// Create mocks the Create method
func (m *MockUserRepository) Create(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

// GetByEmailOrUsername mocks the query method
func (m *MockUserRepository) GetByEmailOrUsername(ctx context.Context, email, username string) (*models.User, error) {
	args := m.Called(ctx, email, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

// GetByID mocks the GetByID method
func (m *MockUserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

// Implement other methods to satisfy the interface (stubs)
func (m *MockUserRepository) Update(ctx context.Context, user *models.User) error {
	return m.Called(ctx, user).Error(0)
}

func (m *MockUserRepository) UpdatePassword(ctx context.Context, userID, hash string) error {
	return m.Called(ctx, userID, hash).Error(0)
}

func (m *MockUserRepository) UpdateLastLogin(ctx context.Context, userID string) error {
	return m.Called(ctx, userID).Error(0)
}

func (m *MockUserRepository) List(ctx context.Context, limit, offset int) ([]models.User, error) {
	args := m.Called(ctx, limit, offset)
	return args.Get(0).([]models.User), args.Error(1)
}

func (m *MockUserRepository) Count(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *MockUserRepository) GetPreferences(ctx context.Context, userID string) (*models.UserPreferences, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserPreferences), args.Error(1)
}

func (m *MockUserRepository) UpsertPreferences(ctx context.Context, prefs *models.UserPreferences) error {
	return m.Called(ctx, prefs).Error(0)
}
