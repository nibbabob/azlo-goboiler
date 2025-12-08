package service

import (
	"azlo-goboiler/internal/config"
	"azlo-goboiler/internal/mocks"
	"azlo-goboiler/internal/models"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRegister(t *testing.T) {
	// 1. Setup
	mockRepo := new(mocks.MockUserRepository)
	cfg := &config.Config{App_Secret: "test-secret"}
	service := NewUserService(mockRepo, cfg)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		// Arrange: Expect GetByEmailOrUsername to return nil (user doesn't exist)
		mockRepo.On("GetByEmailOrUsername", ctx, "new@example.com", "newuser").
			Return(nil, nil).
			Once()

		// Arrange: Expect Create to be called with ANY user object, return nil error
		mockRepo.On("Create", ctx, mock.AnythingOfType("*models.User")).
			Return(nil).
			Once()

		// Act
		req := models.RegisterRequest{
			Username: "newuser",
			Email:    "new@example.com",
			Password: "Password123!",
		}
		resp, err := service.Register(ctx, req)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "newuser", resp.Username)
		assert.NotEmpty(t, resp.UserID)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Fail_UserExists", func(t *testing.T) {
		// Arrange: Expect DB to return an existing user
		existingUser := &models.User{ID: "123", Username: "existing"}
		mockRepo.On("GetByEmailOrUsername", ctx, "taken@example.com", "taken").
			Return(existingUser, nil).
			Once()

		// Act
		req := models.RegisterRequest{
			Username: "taken",
			Email:    "taken@example.com",
			Password: "Password123!",
		}
		resp, err := service.Register(ctx, req)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, "user with this email or username already exists", err.Error())

		// Ensure Create was NEVER called
		mockRepo.AssertNotCalled(t, "Create")
	})
}
