package unit

import (
	"testing"

	"github.com/anhbaysgalan1/gp/internal/models"
	"github.com/anhbaysgalan1/gp/internal/validation"
	"github.com/stretchr/testify/assert"
)

func TestValidateCreateUserRequest(t *testing.T) {
	tests := []struct {
		name      string
		request   models.CreateUserRequest
		wantError bool
		errorMsg  string
	}{
		{
			name: "Valid request",
			request: models.CreateUserRequest{
				Email:    "test@example.com",
				Username: "test_user",
				Password: "Password123!",
			},
			wantError: false,
		},
		{
			name: "Missing email",
			request: models.CreateUserRequest{
				Username: "test_user",
				Password: "Password123!",
			},
			wantError: true,
			errorMsg:  "email is required",
		},
		{
			name: "Invalid email format",
			request: models.CreateUserRequest{
				Email:    "invalid-email",
				Username: "test_user",
				Password: "Password123!",
			},
			wantError: true,
			errorMsg:  "email must be a valid email address",
		},
		{
			name: "Missing username",
			request: models.CreateUserRequest{
				Email:    "test@example.com",
				Password: "Password123!",
			},
			wantError: true,
			errorMsg:  "username is required",
		},
		{
			name: "Username too short",
			request: models.CreateUserRequest{
				Email:    "test@example.com",
				Username: "ab",
				Password: "Password123!",
			},
			wantError: true,
			errorMsg:  "username must be at least 3 characters long",
		},
		{
			name: "Username too long",
			request: models.CreateUserRequest{
				Email:    "test@example.com",
				Username: "this_username_is_way_too_long_and_exceeds_the_maximum_allowed_length",
				Password: "Password123!",
			},
			wantError: true,
			errorMsg:  "username must be at most 50 characters long",
		},
		{
			name: "Username with invalid characters",
			request: models.CreateUserRequest{
				Email:    "test@example.com",
				Username: "test-user!",
				Password: "Password123!",
			},
			wantError: true,
			errorMsg:  "username must contain only letters, numbers, and underscores",
		},
		{
			name: "Missing password",
			request: models.CreateUserRequest{
				Email:    "test@example.com",
				Username: "test_user",
			},
			wantError: true,
			errorMsg:  "password is required",
		},
		{
			name: "Password too short",
			request: models.CreateUserRequest{
				Email:    "test@example.com",
				Username: "test_user",
				Password: "Pass1!",
			},
			wantError: true,
			errorMsg:  "password must be at least 8 characters long",
		},
		{
			name: "Password without uppercase",
			request: models.CreateUserRequest{
				Email:    "test@example.com",
				Username: "test_user",
				Password: "password123!",
			},
			wantError: true,
			errorMsg:  "password must contain at least one uppercase letter, one lowercase letter, one number, and one special character",
		},
		{
			name: "Password without lowercase",
			request: models.CreateUserRequest{
				Email:    "test@example.com",
				Username: "test_user",
				Password: "PASSWORD123!",
			},
			wantError: true,
			errorMsg:  "password must contain at least one uppercase letter, one lowercase letter, one number, and one special character",
		},
		{
			name: "Password without number",
			request: models.CreateUserRequest{
				Email:    "test@example.com",
				Username: "test_user",
				Password: "Password!",
			},
			wantError: true,
			errorMsg:  "password must contain at least one uppercase letter, one lowercase letter, one number, and one special character",
		},
		{
			name: "Password without special character",
			request: models.CreateUserRequest{
				Email:    "test@example.com",
				Username: "test_user",
				Password: "Password123",
			},
			wantError: true,
			errorMsg:  "password must contain at least one uppercase letter, one lowercase letter, one number, and one special character",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.Validate(&tt.request)

			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateLoginRequest(t *testing.T) {
	tests := []struct {
		name      string
		request   models.LoginRequest
		wantError bool
		errorMsg  string
	}{
		{
			name: "Valid email login",
			request: models.LoginRequest{
				EmailOrUsername: "test@example.com",
				Password:        "Password123!",
			},
			wantError: false,
		},
		{
			name: "Valid username login",
			request: models.LoginRequest{
				EmailOrUsername: "test_user",
				Password:        "Password123!",
			},
			wantError: false,
		},
		{
			name: "Missing email or username",
			request: models.LoginRequest{
				Password: "Password123!",
			},
			wantError: true,
			errorMsg:  "email_or_username is required",
		},
		{
			name: "Missing password",
			request: models.LoginRequest{
				EmailOrUsername: "test@example.com",
			},
			wantError: true,
			errorMsg:  "password is required",
		},
		{
			name: "Empty email or username",
			request: models.LoginRequest{
				EmailOrUsername: "",
				Password:        "Password123!",
			},
			wantError: true,
			errorMsg:  "email_or_username is required",
		},
		{
			name: "Empty password",
			request: models.LoginRequest{
				EmailOrUsername: "test@example.com",
				Password:        "",
			},
			wantError: true,
			errorMsg:  "password is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.Validate(&tt.request)

			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateUUID(t *testing.T) {
	tests := []struct {
		name      string
		uuid      string
		wantError bool
	}{
		{
			name:      "Valid UUID",
			uuid:      "550e8400-e29b-41d4-a716-446655440000",
			wantError: false,
		},
		{
			name:      "Invalid UUID format",
			uuid:      "invalid-uuid",
			wantError: true,
		},
		{
			name:      "Empty UUID",
			uuid:      "",
			wantError: true,
		},
		{
			name:      "UUID with wrong length",
			uuid:      "550e8400-e29b-41d4-a716",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateUUID(tt.uuid)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name      string
		email     string
		wantError bool
	}{
		{
			name:      "Valid email",
			email:     "test@example.com",
			wantError: false,
		},
		{
			name:      "Valid email with subdomain",
			email:     "user@mail.example.com",
			wantError: false,
		},
		{
			name:      "Invalid email without @",
			email:     "testexample.com",
			wantError: true,
		},
		{
			name:      "Invalid email without domain",
			email:     "test@",
			wantError: true,
		},
		{
			name:      "Empty email",
			email:     "",
			wantError: true,
		},
		{
			name:      "Invalid email format",
			email:     "test.example.com",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateEmail(tt.email)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidatePositiveInt(t *testing.T) {
	tests := []struct {
		name      string
		value     int64
		fieldName string
		wantError bool
	}{
		{
			name:      "Positive value",
			value:     100,
			fieldName: "amount",
			wantError: false,
		},
		{
			name:      "Zero value",
			value:     0,
			fieldName: "amount",
			wantError: true,
		},
		{
			name:      "Negative value",
			value:     -50,
			fieldName: "amount",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidatePositiveInt(tt.value, tt.fieldName)

			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.fieldName)
				assert.Contains(t, err.Error(), "must be greater than 0")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateRange(t *testing.T) {
	tests := []struct {
		name      string
		value     int64
		min       int64
		max       int64
		fieldName string
		wantError bool
	}{
		{
			name:      "Value in range",
			value:     50,
			min:       1,
			max:       100,
			fieldName: "amount",
			wantError: false,
		},
		{
			name:      "Value at minimum",
			value:     1,
			min:       1,
			max:       100,
			fieldName: "amount",
			wantError: false,
		},
		{
			name:      "Value at maximum",
			value:     100,
			min:       1,
			max:       100,
			fieldName: "amount",
			wantError: false,
		},
		{
			name:      "Value below minimum",
			value:     0,
			min:       1,
			max:       100,
			fieldName: "amount",
			wantError: true,
		},
		{
			name:      "Value above maximum",
			value:     101,
			min:       1,
			max:       100,
			fieldName: "amount",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateRange(tt.value, tt.min, tt.max, tt.fieldName)

			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.fieldName)
				assert.Contains(t, err.Error(), "must be between")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
