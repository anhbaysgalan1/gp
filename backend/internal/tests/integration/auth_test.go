package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/anhbaysgalan1/gp/internal/auth"
	"github.com/anhbaysgalan1/gp/internal/config"
	"github.com/anhbaysgalan1/gp/internal/database"
	"github.com/anhbaysgalan1/gp/internal/formance"
	"github.com/anhbaysgalan1/gp/internal/handlers"
	"github.com/anhbaysgalan1/gp/internal/models"
	"github.com/anhbaysgalan1/gp/internal/services"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type AuthIntegrationTestSuite struct {
	suite.Suite
	db              *database.DB
	router          chi.Router
	authService     *services.AuthService
	jwtManager      *auth.JWTManager
	authMiddleware  *auth.AuthMiddleware
	formanceService *formance.Service
}

func (suite *AuthIntegrationTestSuite) SetupSuite() {
	// Set test environment
	os.Setenv("ENVIRONMENT", "test")
	os.Setenv("DATABASE_URL", "postgres://poker_user:poker_password@localhost:5432/poker_platform_test?sslmode=disable")
	os.Setenv("JWT_SECRET", "test-secret-key-for-jwt-tokens")

	// Load test configuration
	cfg := config.Load()

	// Connect to test database
	db, err := database.NewConnection(cfg)
	require.NoError(suite.T(), err)
	suite.db = db

	// Run migrations
	err = db.AutoMigrate()
	require.NoError(suite.T(), err)

	// Setup services
	suite.formanceService = formance.NewService(cfg)
	suite.jwtManager = auth.NewJWTManager(cfg.JWTSecret, "poker-platform-test")
	suite.authMiddleware = auth.NewAuthMiddleware(suite.jwtManager)
	emailService := services.NewEmailService(cfg)
	suite.authService = services.NewAuthService(db, suite.jwtManager, emailService)

	// Setup router
	suite.setupRouter()
}

func (suite *AuthIntegrationTestSuite) SetupTest() {
	// Clean database before each test
	suite.db.Exec("TRUNCATE TABLE users, email_verifications RESTART IDENTITY CASCADE")
}

func (suite *AuthIntegrationTestSuite) TearDownSuite() {
	// Clean up test database
	suite.db.Exec("DROP SCHEMA public CASCADE; CREATE SCHEMA public;")
	suite.db.Close()
}

func (suite *AuthIntegrationTestSuite) setupRouter() {
	r := chi.NewRouter()

	// Auth handler
	authHandler := handlers.NewAuthHandler(suite.authService)

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Public auth routes
		r.Mount("/auth", authHandler.Routes())

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(suite.authMiddleware.RequireAuth)
			r.Mount("/user", authHandler.ProtectedRoutes())
		})
	})

	suite.router = r
}

func (suite *AuthIntegrationTestSuite) TestRegisterUser() {
	tests := []struct {
		name           string
		payload        models.CreateUserRequest
		expectedStatus int
		expectedError  string
	}{
		{
			name: "Valid registration",
			payload: models.CreateUserRequest{
				Email:    "test@example.com",
				Username: "test_user",
				Password: "Password123!",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Missing email",
			payload: models.CreateUserRequest{
				Username: "test_user",
				Password: "Password123!",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "email is required",
		},
		{
			name: "Invalid email",
			payload: models.CreateUserRequest{
				Email:    "invalid-email",
				Username: "test_user",
				Password: "Password123!",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "email must be a valid email address",
		},
		{
			name: "Missing username",
			payload: models.CreateUserRequest{
				Email:    "test@example.com",
				Password: "Password123!",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "username is required",
		},
		{
			name: "Invalid username with special chars",
			payload: models.CreateUserRequest{
				Email:    "test@example.com",
				Username: "test-user!",
				Password: "Password123!",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "username must contain only letters, numbers, and underscores",
		},
		{
			name: "Missing password",
			payload: models.CreateUserRequest{
				Email:    "test@example.com",
				Username: "test_user",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "password is required",
		},
		{
			name: "Weak password",
			payload: models.CreateUserRequest{
				Email:    "test@example.com",
				Username: "test_user",
				Password: "password123",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "password must contain at least one uppercase letter, one lowercase letter, one number, and one special character",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			// Prepare request
			body, err := json.Marshal(tt.payload)
			require.NoError(suite.T(), err)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			// Execute request
			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)

			// Verify response
			assert.Equal(suite.T(), tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(suite.T(), err)

			if tt.expectedError != "" {
				assert.Equal(suite.T(), tt.expectedError, response["error"])
			} else {
				assert.Equal(suite.T(), "User registered successfully", response["message"])
				assert.NotEmpty(suite.T(), response["user"])
				assert.NotEmpty(suite.T(), response["verification_token"])

				// Verify user in database
				user := response["user"].(map[string]interface{})
				assert.Equal(suite.T(), tt.payload.Email, user["email"])
				assert.Equal(suite.T(), tt.payload.Username, user["username"])
				assert.Equal(suite.T(), false, user["is_verified"])
			}
		})
	}
}

func (suite *AuthIntegrationTestSuite) TestRegisterDuplicateUser() {
	// First registration
	payload := models.CreateUserRequest{
		Email:    "test@example.com",
		Username: "test_user",
		Password: "Password123!",
	}

	body, err := json.Marshal(payload)
	require.NoError(suite.T(), err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	// Second registration with same email
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusConflict, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	assert.Contains(suite.T(), response["error"], "already exists")
}

func (suite *AuthIntegrationTestSuite) TestLoginUser() {
	// First register a user
	registerPayload := models.CreateUserRequest{
		Email:    "test@example.com",
		Username: "test_user",
		Password: "Password123!",
	}

	body, err := json.Marshal(registerPayload)
	require.NoError(suite.T(), err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	require.Equal(suite.T(), http.StatusCreated, w.Code)

	tests := []struct {
		name           string
		payload        models.LoginRequest
		expectedStatus int
		expectedError  string
	}{
		{
			name: "Valid login with email",
			payload: models.LoginRequest{
				EmailOrUsername: "test@example.com",
				Password:        "Password123!",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Valid login with username",
			payload: models.LoginRequest{
				EmailOrUsername: "test_user",
				Password:        "Password123!",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Invalid password",
			payload: models.LoginRequest{
				EmailOrUsername: "test@example.com",
				Password:        "WrongPassword123!",
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Invalid credentials",
		},
		{
			name: "Non-existent user",
			payload: models.LoginRequest{
				EmailOrUsername: "nonexistent@example.com",
				Password:        "Password123!",
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Invalid credentials",
		},
		{
			name: "Missing credentials",
			payload: models.LoginRequest{
				EmailOrUsername: "",
				Password:        "Password123!",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "email_or_username is required",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			// Prepare request
			body, err := json.Marshal(tt.payload)
			require.NoError(suite.T(), err)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			// Execute request
			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)

			// Verify response
			assert.Equal(suite.T(), tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(suite.T(), err)

			if tt.expectedError != "" {
				assert.Equal(suite.T(), tt.expectedError, response["error"])
			} else {
				assert.NotEmpty(suite.T(), response["user"])
				assert.NotEmpty(suite.T(), response["token"])

				// Verify JWT token
				token := response["token"].(string)
				claims, err := suite.jwtManager.ValidateToken(token)
				require.NoError(suite.T(), err)
				assert.Equal(suite.T(), registerPayload.Username, claims.Username)
				assert.Equal(suite.T(), registerPayload.Email, claims.Email)
			}
		})
	}
}

func (suite *AuthIntegrationTestSuite) TestGetCurrentUser() {
	// First register and login a user
	registerPayload := models.CreateUserRequest{
		Email:    "test@example.com",
		Username: "test_user",
		Password: "Password123!",
	}

	body, err := json.Marshal(registerPayload)
	require.NoError(suite.T(), err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	require.Equal(suite.T(), http.StatusCreated, w.Code)

	// Login to get token
	loginPayload := models.LoginRequest{
		EmailOrUsername: "test@example.com",
		Password:        "Password123!",
	}

	body, err = json.Marshal(loginPayload)
	require.NoError(suite.T(), err)

	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	require.Equal(suite.T(), http.StatusOK, w.Code)

	var loginResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &loginResponse)
	require.NoError(suite.T(), err)

	token := loginResponse["token"].(string)

	tests := []struct {
		name           string
		token          string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Valid token",
			token:          token,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Missing token",
			token:          "",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "User not authenticated",
		},
		{
			name:           "Invalid token",
			token:          "invalid.jwt.token",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "User not authenticated",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/user/me", nil)
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}

			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)

			assert.Equal(suite.T(), tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(suite.T(), err)

			if tt.expectedError != "" {
				assert.Equal(suite.T(), tt.expectedError, response["error"])
			} else {
				assert.Equal(suite.T(), registerPayload.Email, response["email"])
				assert.Equal(suite.T(), registerPayload.Username, response["username"])
				assert.Equal(suite.T(), false, response["is_verified"])
			}
		})
	}
}

func (suite *AuthIntegrationTestSuite) TestEmailVerification() {
	// Register a user
	registerPayload := models.CreateUserRequest{
		Email:    "test@example.com",
		Username: "test_user",
		Password: "Password123!",
	}

	body, err := json.Marshal(registerPayload)
	require.NoError(suite.T(), err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	require.Equal(suite.T(), http.StatusCreated, w.Code)

	var registerResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &registerResponse)
	require.NoError(suite.T(), err)

	verificationToken := registerResponse["verification_token"].(string)

	tests := []struct {
		name           string
		token          string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Valid verification token",
			token:          verificationToken,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid verification token",
			token:          "invalid_token",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Missing token",
			token:          "",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Token is required",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			payload := map[string]string{
				"token": tt.token,
			}

			body, err := json.Marshal(payload)
			require.NoError(suite.T(), err)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/verify-email", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)

			assert.Equal(suite.T(), tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(suite.T(), err)

			if tt.expectedError != "" {
				assert.Equal(suite.T(), tt.expectedError, response["error"])
			} else if tt.expectedStatus == http.StatusOK {
				assert.Equal(suite.T(), "Email verified successfully", response["message"])
			}
		})
	}
}

func (suite *AuthIntegrationTestSuite) TestUpdateProfile() {
	// Register, login, and get token
	registerPayload := models.CreateUserRequest{
		Email:    "test@example.com",
		Username: "test_user",
		Password: "Password123!",
	}

	body, err := json.Marshal(registerPayload)
	require.NoError(suite.T(), err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	require.Equal(suite.T(), http.StatusCreated, w.Code)

	// Login
	loginPayload := models.LoginRequest{
		EmailOrUsername: "test@example.com",
		Password:        "Password123!",
	}

	body, err = json.Marshal(loginPayload)
	require.NoError(suite.T(), err)

	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	require.Equal(suite.T(), http.StatusOK, w.Code)

	var loginResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &loginResponse)
	require.NoError(suite.T(), err)

	token := loginResponse["token"].(string)

	tests := []struct {
		name           string
		payload        map[string]interface{}
		token          string
		expectedStatus int
		expectedError  string
	}{
		{
			name: "Valid profile update",
			payload: map[string]interface{}{
				"avatar_url": "https://example.com/avatar.png",
			},
			token:          token,
			expectedStatus: http.StatusOK,
		},
		{
			name: "Update without token",
			payload: map[string]interface{}{
				"avatar_url": "https://example.com/avatar.png",
			},
			token:          "",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "User not authenticated",
		},
		{
			name: "Update with invalid token",
			payload: map[string]interface{}{
				"avatar_url": "https://example.com/avatar.png",
			},
			token:          "invalid.jwt.token",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "User not authenticated",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			body, err := json.Marshal(tt.payload)
			require.NoError(suite.T(), err)

			req := httptest.NewRequest(http.MethodPut, "/api/v1/user/profile", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}

			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)

			assert.Equal(suite.T(), tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(suite.T(), err)

			if tt.expectedError != "" {
				assert.Equal(suite.T(), tt.expectedError, response["error"])
			} else if tt.expectedStatus == http.StatusOK {
				assert.Equal(suite.T(), "Profile updated successfully", response["message"])
			}
		})
	}
}

// Helper functions

func generateTestToken(jwtManager *auth.JWTManager, userID uuid.UUID, username, email string) (string, error) {
	return jwtManager.GenerateToken(userID, username, email)
}

func TestAuthIntegrationSuite(t *testing.T) {
	suite.Run(t, new(AuthIntegrationTestSuite))
}
