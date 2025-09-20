package auth

import (
	"context"
	"fmt"
	"net/http"

	"github.com/anhbaysgalan1/gp/internal/database"
	"github.com/anhbaysgalan1/gp/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RoleMiddleware provides role-based authorization middleware
type RoleMiddleware struct {
	db *database.DB
}

// NewRoleMiddleware creates a new role middleware instance
func NewRoleMiddleware(db *database.DB) *RoleMiddleware {
	return &RoleMiddleware{
		db: db,
	}
}

// RequireRole returns a middleware that requires specific roles
func (rm *RoleMiddleware) RequireRole(roles ...models.UserRole) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get user ID from context (should be set by auth middleware)
			userID, ok := GetUserIDFromContext(r.Context())
			if !ok {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Fetch user from database to get their role
			var user models.User
			if err := rm.db.First(&user, "id = ?", userID).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					http.Error(w, "User not found", http.StatusUnauthorized)
					return
				}
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			// Check if user has any of the required roles
			hasRequiredRole := false
			for _, requiredRole := range roles {
				if user.Role == requiredRole {
					hasRequiredRole = true
					break
				}
			}

			if !hasRequiredRole {
				http.Error(w, "Insufficient privileges", http.StatusForbidden)
				return
			}

			// Add user role to context for handlers to use
			ctx := context.WithValue(r.Context(), "user_role", user.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAdmin is a convenience method for admin-only endpoints
func (rm *RoleMiddleware) RequireAdmin(next http.Handler) http.Handler {
	return rm.RequireRole(models.UserRoleAdmin)(next)
}

// RequireModerator is a convenience method for moderator and admin endpoints
func (rm *RoleMiddleware) RequireModerator(next http.Handler) http.Handler {
	return rm.RequireRole(models.UserRoleMod, models.UserRoleAdmin)(next)
}

// GetUserRoleFromContext retrieves the user role from the request context
func GetUserRoleFromContext(ctx context.Context) (models.UserRole, bool) {
	role, ok := ctx.Value("user_role").(models.UserRole)
	return role, ok
}

// HasRole checks if a user has a specific role
func (rm *RoleMiddleware) HasRole(userID uuid.UUID, role models.UserRole) (bool, error) {
	var user models.User
	if err := rm.db.First(&user, "id = ?", userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, fmt.Errorf("user not found")
		}
		return false, fmt.Errorf("database error: %w", err)
	}
	return user.Role == role, nil
}

// IsAdmin checks if a user has admin role
func (rm *RoleMiddleware) IsAdmin(userID uuid.UUID) (bool, error) {
	return rm.HasRole(userID, models.UserRoleAdmin)
}

// IsModerator checks if a user has moderator or admin role
func (rm *RoleMiddleware) IsModerator(userID uuid.UUID) (bool, error) {
	var user models.User
	if err := rm.db.First(&user, "id = ?", userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, fmt.Errorf("user not found")
		}
		return false, fmt.Errorf("database error: %w", err)
	}
	return user.Role == models.UserRoleMod || user.Role == models.UserRoleAdmin, nil
}
