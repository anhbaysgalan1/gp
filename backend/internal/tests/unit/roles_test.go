package unit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/anhbaysgalan1/gp/internal/auth"
	"github.com/anhbaysgalan1/gp/internal/config"
	"github.com/anhbaysgalan1/gp/internal/database"
	"github.com/anhbaysgalan1/gp/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type RoleTestSuite struct {
	suite.Suite
	db             *database.DB
	roleMiddleware *auth.RoleMiddleware
	jwtManager     *auth.JWTManager
	testUsers      map[models.UserRole]*models.User
}

func (suite *RoleTestSuite) SetupSuite() {
	// Set test environment
	os.Setenv("ENVIRONMENT", "test")
	os.Setenv("DATABASE_URL", "postgres://poker_user:poker_password@localhost:5432/poker_platform_test?sslmode=disable")

	// Load test configuration
	cfg := config.Load()

	// Connect to test database
	db, err := database.NewConnection(cfg)
	require.NoError(suite.T(), err)
	suite.db = db

	// Run migrations
	err = db.AutoMigrate()
	require.NoError(suite.T(), err)

	// Setup role middleware
	suite.roleMiddleware = auth.NewRoleMiddleware(db)
	suite.jwtManager = auth.NewJWTManager(cfg.JWTSecret, "poker-platform-test")
}

func (suite *RoleTestSuite) SetupTest() {
	// Clean database before each test
	suite.db.Exec("DELETE FROM users")

	// Create test users with different roles
	suite.testUsers = make(map[models.UserRole]*models.User)

	roles := []models.UserRole{models.UserRolePlayer, models.UserRoleMod, models.UserRoleAdmin}
	for _, role := range roles {
		user := &models.User{
			Email:        string(role) + "@example.com",
			Username:     string(role) + "_user",
			PasswordHash: "hashed_password",
			Role:         role,
			IsVerified:   true,
		}
		err := suite.db.Create(user).Error
		require.NoError(suite.T(), err)
		suite.testUsers[role] = user
	}
}

func (suite *RoleTestSuite) TearDownSuite() {
	// Clean up test database
	suite.db.Exec("DROP SCHEMA public CASCADE; CREATE SCHEMA public;")
	suite.db.Close()
}

func (suite *RoleTestSuite) TestRequireRole_AdminOnly() {
	// Create a handler that requires admin role
	handler := suite.roleMiddleware.RequireRole(models.UserRoleAdmin)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("admin access granted"))
		}))

	tests := []struct {
		name           string
		userRole       models.UserRole
		expectedStatus int
	}{
		{
			name:           "Admin user should have access",
			userRole:       models.UserRoleAdmin,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Moderator should be denied",
			userRole:       models.UserRoleMod,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Player should be denied",
			userRole:       models.UserRolePlayer,
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			user := suite.testUsers[tt.userRole]
			ctx := context.WithValue(context.Background(), "user_id", user.ID)

			req := httptest.NewRequest(http.MethodGet, "/admin", nil)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			assert.Equal(suite.T(), tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				assert.Equal(suite.T(), "admin access granted", w.Body.String())
				// Check that user role is added to context
				assert.Contains(suite.T(), w.Header(), "user_role")
			}
		})
	}
}

func (suite *RoleTestSuite) TestRequireRole_ModeratorAndAdmin() {
	// Create a handler that requires moderator or admin role
	handler := suite.roleMiddleware.RequireRole(models.UserRoleMod, models.UserRoleAdmin)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("moderator access granted"))
		}))

	tests := []struct {
		name           string
		userRole       models.UserRole
		expectedStatus int
	}{
		{
			name:           "Admin user should have access",
			userRole:       models.UserRoleAdmin,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Moderator should have access",
			userRole:       models.UserRoleMod,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Player should be denied",
			userRole:       models.UserRolePlayer,
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			user := suite.testUsers[tt.userRole]
			ctx := context.WithValue(context.Background(), "user_id", user.ID)

			req := httptest.NewRequest(http.MethodGet, "/moderate", nil)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			assert.Equal(suite.T(), tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				assert.Equal(suite.T(), "moderator access granted", w.Body.String())
			}
		})
	}
}

func (suite *RoleTestSuite) TestRequireAdmin_ConvenienceMethod() {
	handler := suite.roleMiddleware.RequireAdmin(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("admin only"))
		}))

	// Test with admin user
	adminUser := suite.testUsers[models.UserRoleAdmin]
	ctx := context.WithValue(context.Background(), "user_id", adminUser.ID)

	req := httptest.NewRequest(http.MethodGet, "/admin-only", nil)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	assert.Equal(suite.T(), "admin only", w.Body.String())

	// Test with non-admin user
	playerUser := suite.testUsers[models.UserRolePlayer]
	ctx = context.WithValue(context.Background(), "user_id", playerUser.ID)

	req = httptest.NewRequest(http.MethodGet, "/admin-only", nil)
	req = req.WithContext(ctx)

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusForbidden, w.Code)
}

func (suite *RoleTestSuite) TestRequireModerator_ConvenienceMethod() {
	handler := suite.roleMiddleware.RequireModerator(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("moderator or admin"))
		}))

	// Test with admin user
	adminUser := suite.testUsers[models.UserRoleAdmin]
	ctx := context.WithValue(context.Background(), "user_id", adminUser.ID)

	req := httptest.NewRequest(http.MethodGet, "/moderate", nil)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Test with moderator user
	modUser := suite.testUsers[models.UserRoleMod]
	ctx = context.WithValue(context.Background(), "user_id", modUser.ID)

	req = httptest.NewRequest(http.MethodGet, "/moderate", nil)
	req = req.WithContext(ctx)

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Test with player user
	playerUser := suite.testUsers[models.UserRolePlayer]
	ctx = context.WithValue(context.Background(), "user_id", playerUser.ID)

	req = httptest.NewRequest(http.MethodGet, "/moderate", nil)
	req = req.WithContext(ctx)

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusForbidden, w.Code)
}

func (suite *RoleTestSuite) TestRequireRole_NoUserInContext() {
	handler := suite.roleMiddleware.RequireRole(models.UserRolePlayer)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// No user_id in context

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}

func (suite *RoleTestSuite) TestRequireRole_NonExistentUser() {
	handler := suite.roleMiddleware.RequireRole(models.UserRolePlayer)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

	// Use a non-existent user ID
	nonExistentID := uuid.New()
	ctx := context.WithValue(context.Background(), "user_id", nonExistentID)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}

func (suite *RoleTestSuite) TestGetUserRoleFromContext() {
	// Test getting role from context
	ctx := context.WithValue(context.Background(), "user_role", models.UserRoleAdmin)
	role, ok := auth.GetUserRoleFromContext(ctx)

	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), models.UserRoleAdmin, role)

	// Test with no role in context
	emptyCtx := context.Background()
	role, ok = auth.GetUserRoleFromContext(emptyCtx)

	assert.False(suite.T(), ok)
	assert.Equal(suite.T(), models.UserRole(""), role)
}

func (suite *RoleTestSuite) TestHasRole() {
	adminUser := suite.testUsers[models.UserRoleAdmin]
	playerUser := suite.testUsers[models.UserRolePlayer]

	// Test admin user has admin role
	hasRole, err := suite.roleMiddleware.HasRole(adminUser.ID, models.UserRoleAdmin)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), hasRole)

	// Test admin user doesn't have player role
	hasRole, err = suite.roleMiddleware.HasRole(adminUser.ID, models.UserRolePlayer)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), hasRole)

	// Test player user has player role
	hasRole, err = suite.roleMiddleware.HasRole(playerUser.ID, models.UserRolePlayer)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), hasRole)

	// Test non-existent user
	nonExistentID := uuid.New()
	hasRole, err = suite.roleMiddleware.HasRole(nonExistentID, models.UserRolePlayer)
	assert.Error(suite.T(), err)
	assert.False(suite.T(), hasRole)
	assert.Contains(suite.T(), err.Error(), "user not found")
}

func (suite *RoleTestSuite) TestIsAdmin() {
	adminUser := suite.testUsers[models.UserRoleAdmin]
	playerUser := suite.testUsers[models.UserRolePlayer]

	// Test admin user
	isAdmin, err := suite.roleMiddleware.IsAdmin(adminUser.ID)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), isAdmin)

	// Test non-admin user
	isAdmin, err = suite.roleMiddleware.IsAdmin(playerUser.ID)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), isAdmin)

	// Test non-existent user
	nonExistentID := uuid.New()
	isAdmin, err = suite.roleMiddleware.IsAdmin(nonExistentID)
	assert.Error(suite.T(), err)
	assert.False(suite.T(), isAdmin)
}

func (suite *RoleTestSuite) TestIsModerator() {
	adminUser := suite.testUsers[models.UserRoleAdmin]
	modUser := suite.testUsers[models.UserRoleMod]
	playerUser := suite.testUsers[models.UserRolePlayer]

	// Test admin user (should be considered moderator)
	isMod, err := suite.roleMiddleware.IsModerator(adminUser.ID)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), isMod)

	// Test moderator user
	isMod, err = suite.roleMiddleware.IsModerator(modUser.ID)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), isMod)

	// Test player user
	isMod, err = suite.roleMiddleware.IsModerator(playerUser.ID)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), isMod)

	// Test non-existent user
	nonExistentID := uuid.New()
	isMod, err = suite.roleMiddleware.IsModerator(nonExistentID)
	assert.Error(suite.T(), err)
	assert.False(suite.T(), isMod)
}

func TestRoleTestSuite(t *testing.T) {
	suite.Run(t, new(RoleTestSuite))
}
