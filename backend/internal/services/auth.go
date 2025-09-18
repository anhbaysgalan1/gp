package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/evanofslack/go-poker/internal/auth"
	"github.com/evanofslack/go-poker/internal/database"
	"github.com/evanofslack/go-poker/internal/formance"
	"github.com/evanofslack/go-poker/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AuthService struct {
	db              *database.DB
	jwtManager      *auth.JWTManager
	emailService    *EmailService
	formanceService *formance.Service
}

func NewAuthService(db *database.DB, jwtManager *auth.JWTManager, emailService *EmailService, formanceService *formance.Service) *AuthService {
	return &AuthService{
		db:              db,
		jwtManager:      jwtManager,
		emailService:    emailService,
		formanceService: formanceService,
	}
}

func (s *AuthService) RegisterUser(req models.CreateUserRequest) (*models.User, error) {
	// Check if user already exists
	var existingUser models.User
	if err := s.db.Where("email = ? OR username = ?", req.Email, req.Username).First(&existingUser).Error; err == nil {
		if existingUser.Email == req.Email {
			return nil, fmt.Errorf("user with email %s already exists", req.Email)
		}
		return nil, fmt.Errorf("user with username %s already exists", req.Username)
	} else if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}

	// Hash password
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := models.User{
		Email:        req.Email,
		Username:     req.Username,
		PasswordHash: hashedPassword,
		Role:         models.UserRolePlayer, // Default role
		IsVerified:   false,                 // Email verification required
	}

	if err := s.db.Create(&user).Error; err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Create wallet in Formance ledger
	ctx := context.Background()
	if err := s.formanceService.CreateUserWallet(ctx, user.ID); err != nil {
		slog.Warn("Failed to create user wallet", "error", err, "user_id", user.ID)
		// Note: We don't fail registration if wallet creation fails
		// The wallet can be created later or on first balance check
	} else {
		slog.Info("User wallet created successfully", "user_id", user.ID)
	}

	// Create and send email verification
	verification, err := s.CreateEmailVerification(user.ID)
	if err != nil {
		slog.Warn("Failed to create email verification", "error", err, "user_id", user.ID)
	} else {
		// Send verification email
		if err := s.emailService.SendVerificationEmail(user.Email, user.Username, verification.Token); err != nil {
			slog.Warn("Failed to send verification email", "error", err, "user_id", user.ID)
		} else {
			slog.Info("Verification email sent successfully", "user_id", user.ID)
		}
	}

	slog.Info("User registered successfully", "user_id", user.ID, "username", user.Username)
	return &user, nil
}

func (s *AuthService) LoginUser(req models.LoginRequest) (*models.LoginResponse, error) {
	var user models.User

	// Find user by email or username
	if err := s.db.Where("email = ? OR username = ?", req.EmailOrUsername, req.EmailOrUsername).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("invalid credentials")
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	// Verify password
	if err := auth.VerifyPassword(req.Password, user.PasswordHash); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Generate JWT token
	token, err := s.jwtManager.GenerateToken(user.ID, user.Username, user.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	slog.Info("User logged in successfully", "user_id", user.ID, "username", user.Username)

	return &models.LoginResponse{
		User:  user,
		Token: token,
	}, nil
}

func (s *AuthService) GetUserByID(userID uuid.UUID) (*models.User, error) {
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

func (s *AuthService) CreateEmailVerification(userID uuid.UUID) (*models.EmailVerification, error) {
	// Generate verification token
	token, err := auth.GenerateToken(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate verification token: %w", err)
	}

	// Create verification record
	verification := models.EmailVerification{
		UserID:    userID,
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour), // 24 hours expiry
	}

	if err := s.db.Create(&verification).Error; err != nil {
		return nil, fmt.Errorf("failed to create email verification: %w", err)
	}

	return &verification, nil
}

func (s *AuthService) VerifyEmail(token string) error {
	var verification models.EmailVerification
	if err := s.db.Where("token = ? AND expires_at > ?", token, time.Now()).First(&verification).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("invalid or expired verification token")
		}
		return fmt.Errorf("failed to find verification token: %w", err)
	}

	// Update user as verified
	if err := s.db.Model(&models.User{}).Where("id = ?", verification.UserID).Update("is_verified", true).Error; err != nil {
		return fmt.Errorf("failed to verify user: %w", err)
	}

	// Delete verification token
	if err := s.db.Delete(&verification).Error; err != nil {
		slog.Warn("Failed to delete verification token", "error", err)
	}

	slog.Info("Email verified successfully", "user_id", verification.UserID)
	return nil
}

func (s *AuthService) UpdateUserProfile(userID uuid.UUID, updates map[string]interface{}) error {
	// Remove sensitive fields
	delete(updates, "password_hash")
	delete(updates, "id")
	delete(updates, "created_at")
	delete(updates, "updated_at")

	if err := s.db.Model(&models.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update user profile: %w", err)
	}

	slog.Info("User profile updated", "user_id", userID)
	return nil
}