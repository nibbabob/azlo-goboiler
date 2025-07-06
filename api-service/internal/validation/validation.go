// File: internal/validation/validation.go
package validation

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New()

	// Register custom validators
	validate.RegisterValidation("password", validatePassword)
	validate.RegisterValidation("alphanum", validateAlphaNum)
}

// ValidateStruct validates a struct and returns a user-friendly error message
func ValidateStruct(s interface{}) error {
	err := validate.Struct(s)
	if err == nil {
		return nil
	}

	var errorMessages []string

	for _, err := range err.(validator.ValidationErrors) {
		errorMessages = append(errorMessages, getErrorMessage(err))
	}

	return fmt.Errorf("validation failed: %s", strings.Join(errorMessages, "; "))
}

// getErrorMessage returns a user-friendly error message for validation errors
func getErrorMessage(fe validator.FieldError) string {
	field := strings.ToLower(fe.Field())

	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "email":
		return fmt.Sprintf("%s must be a valid email address", field)
	case "min":
		return fmt.Sprintf("%s must be at least %s characters long", field, fe.Param())
	case "max":
		return fmt.Sprintf("%s must not exceed %s characters", field, fe.Param())
	case "alphanum":
		return fmt.Sprintf("%s must contain only letters and numbers", field)
	case "password":
		return fmt.Sprintf("%s must contain at least one uppercase letter, one lowercase letter, one number, and one special character", field)
	default:
		return fmt.Sprintf("%s is invalid", field)
	}
}

// validatePassword checks if password meets security requirements
func validatePassword(fl validator.FieldLevel) bool {
	password := fl.Field().String()

	if len(password) < 8 {
		return false
	}

	var (
		hasUpper   = false
		hasLower   = false
		hasNumber  = false
		hasSpecial = false
	)

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	return hasUpper && hasLower && hasNumber && hasSpecial
}

// validateAlphaNum checks if string contains only letters and numbers
func validateAlphaNum(fl validator.FieldLevel) bool {
	str := fl.Field().String()
	alphaNumRegex := regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	return alphaNumRegex.MatchString(str)
}

// ValidateEmail validates email format with additional checks
func ValidateEmail(email string) bool {
	if len(email) > 254 {
		return false
	}

	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// SanitizeString removes potentially dangerous characters from user input
func SanitizeString(input string) string {
	// Remove null bytes and control characters
	cleaned := strings.ReplaceAll(input, "\x00", "")

	// Remove or replace other potentially dangerous characters
	dangerous := []string{
		"<script", "</script>", "javascript:", "onload=", "onerror=",
		"<iframe", "</iframe>", "<object", "</object>", "<embed", "</embed>",
	}

	for _, danger := range dangerous {
		cleaned = strings.ReplaceAll(strings.ToLower(cleaned), danger, "")
	}

	return strings.TrimSpace(cleaned)
}
