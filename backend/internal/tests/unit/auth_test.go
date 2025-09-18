package unit

import (
	"testing"
	"time"

	"github.com/evanofslack/go-poker/internal/auth"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTManager_GenerateToken(t *testing.T) {
	jwtManager := auth.NewJWTManager("test-secret", "test-issuer")

	userID := uuid.New()
	username := "testuser"
	email := "test@example.com"

	token, err := jwtManager.GenerateToken(userID, username, email)

	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// Parse the token to verify its contents
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte("test-secret"), nil
	})

	require.NoError(t, err)
	assert.True(t, parsedToken.Valid)

	claims := parsedToken.Claims.(jwt.MapClaims)
	assert.Equal(t, userID.String(), claims["user_id"])
	assert.Equal(t, username, claims["username"])
	assert.Equal(t, email, claims["email"])
	assert.Equal(t, "test-issuer", claims["iss"])
}

func TestJWTManager_ValidateToken(t *testing.T) {
	jwtManager := auth.NewJWTManager("test-secret", "test-issuer")

	userID := uuid.New()
	username := "testuser"
	email := "test@example.com"

	tests := []struct {
		name        string
		setupToken  func() string
		expectError bool
	}{
		{
			name: "Valid token",
			setupToken: func() string {
				token, _ := jwtManager.GenerateToken(userID, username, email)
				return token
			},
			expectError: false,
		},
		{
			name: "Invalid token",
			setupToken: func() string {
				return "invalid.jwt.token"
			},
			expectError: true,
		},
		{
			name: "Token with wrong secret",
			setupToken: func() string {
				wrongManager := auth.NewJWTManager("wrong-secret", "test-issuer")
				token, _ := wrongManager.GenerateToken(userID, username, email)
				return token
			},
			expectError: true,
		},
		{
			name: "Empty token",
			setupToken: func() string {
				return ""
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := tt.setupToken()
			claims, err := jwtManager.ValidateToken(token)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, claims)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, claims)
				assert.Equal(t, userID, claims.UserID)
				assert.Equal(t, username, claims.Username)
				assert.Equal(t, email, claims.Email)
			}
		})
	}
}

func TestJWTManager_ExtractTokenFromBearer(t *testing.T) {
	jwtManager := auth.NewJWTManager("test-secret", "test-issuer")

	tests := []struct {
		name           string
		bearerToken    string
		expectedToken  string
	}{
		{
			name:          "Valid bearer token",
			bearerToken:   "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test.token",
			expectedToken: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test.token",
		},
		{
			name:          "Bearer with extra spaces",
			bearerToken:   "Bearer  eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test.token",
			expectedToken: " eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test.token",
		},
		{
			name:          "Invalid format - missing Bearer",
			bearerToken:   "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test.token",
			expectedToken: "",
		},
		{
			name:          "Invalid format - wrong prefix",
			bearerToken:   "Token eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test.token",
			expectedToken: "",
		},
		{
			name:          "Empty bearer token",
			bearerToken:   "",
			expectedToken: "",
		},
		{
			name:          "Only Bearer prefix",
			bearerToken:   "Bearer",
			expectedToken: "",
		},
		{
			name:          "Bearer with space but no token",
			bearerToken:   "Bearer ",
			expectedToken: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := jwtManager.ExtractTokenFromBearer(tt.bearerToken)
			assert.Equal(t, tt.expectedToken, token)
		})
	}
}

func TestJWTManager_TokenExpiration(t *testing.T) {
	jwtManager := auth.NewJWTManager("test-secret", "test-issuer")

	userID := uuid.New()
	username := "testuser"
	email := "test@example.com"

	// Generate token
	token, err := jwtManager.GenerateToken(userID, username, email)
	require.NoError(t, err)

	// Validate immediately (should work)
	claims, err := jwtManager.ValidateToken(token)
	assert.NoError(t, err)
	assert.NotNil(t, claims)

	// Check expiration time is set correctly (should be ~24 hours from now)
	expectedExp := time.Now().Add(24 * time.Hour).Unix()
	actualExp := claims.ExpiresAt.Unix()

	// Allow 5 second tolerance for test execution time
	assert.InDelta(t, expectedExp, actualExp, 5)

	// Check other time claims
	now := time.Now().Unix()
	assert.InDelta(t, now, claims.IssuedAt.Unix(), 5)
	assert.InDelta(t, now, claims.NotBefore.Unix(), 5)
}

func TestJWTManager_DifferentUsers(t *testing.T) {
	jwtManager := auth.NewJWTManager("test-secret", "test-issuer")

	user1ID := uuid.New()
	user2ID := uuid.New()

	token1, err := jwtManager.GenerateToken(user1ID, "user1", "user1@example.com")
	require.NoError(t, err)

	token2, err := jwtManager.GenerateToken(user2ID, "user2", "user2@example.com")
	require.NoError(t, err)

	// Tokens should be different
	assert.NotEqual(t, token1, token2)

	// Each token should validate to the correct user
	claims1, err := jwtManager.ValidateToken(token1)
	require.NoError(t, err)
	assert.Equal(t, user1ID, claims1.UserID)
	assert.Equal(t, "user1", claims1.Username)
	assert.Equal(t, "user1@example.com", claims1.Email)

	claims2, err := jwtManager.ValidateToken(token2)
	require.NoError(t, err)
	assert.Equal(t, user2ID, claims2.UserID)
	assert.Equal(t, "user2", claims2.Username)
	assert.Equal(t, "user2@example.com", claims2.Email)
}

func TestJWTManager_SigningMethod(t *testing.T) {
	jwtManager := auth.NewJWTManager("test-secret", "test-issuer")

	userID := uuid.New()
	token, err := jwtManager.GenerateToken(userID, "testuser", "test@example.com")
	require.NoError(t, err)

	// Parse token without verification to check signing method
	parsedToken, _, err := new(jwt.Parser).ParseUnverified(token, &auth.Claims{})
	require.NoError(t, err)

	// Should use HS256 signing method
	assert.Equal(t, "HS256", parsedToken.Header["alg"])
	assert.Equal(t, "JWT", parsedToken.Header["typ"])
}