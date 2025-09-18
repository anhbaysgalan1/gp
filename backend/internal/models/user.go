package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserRole represents the role of a user in the system
type UserRole string

const (
	UserRolePlayer UserRole = "player"
	UserRoleMod    UserRole = "moderator"
	UserRoleAdmin  UserRole = "admin"
)

type User struct {
	ID                  uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Email               string         `json:"email" gorm:"uniqueIndex;not null;size:255"`
	Username            string         `json:"username" gorm:"uniqueIndex;not null;size:50"`
	PasswordHash        string         `json:"-" gorm:"not null;size:255"`
	Role                UserRole       `json:"role" gorm:"type:varchar(20);default:'player'"`
	IsVerified          bool           `json:"is_verified" gorm:"default:false"`
	FormanceAccountID   *string        `json:"formance_account_id,omitempty" gorm:"uniqueIndex;size:255"`
	AvatarURL           *string        `json:"avatar_url,omitempty" gorm:"size:500"`
	TotalHandsPlayed    int            `json:"total_hands_played" gorm:"default:0"`
	TotalWinnings       int64          `json:"total_winnings" gorm:"default:0"` // MNT
	CreatedAt           time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt           time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt           gorm.DeletedAt `json:"-" gorm:"index"`
}

type CreateUserRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Username string `json:"username" validate:"required,min=3,max=50,username"`
	Password string `json:"password" validate:"required,min=8,strong_password"`
}

type LoginRequest struct {
	EmailOrUsername string `json:"email_or_username" validate:"required"`
	Password        string `json:"password" validate:"required"`
}

type LoginResponse struct {
	User  User   `json:"user"`
	Token string `json:"token"`
}

type EmailVerification struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID      `json:"user_id" gorm:"type:uuid;not null;index"`
	User      User           `json:"-" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Token     string         `json:"token" gorm:"uniqueIndex;not null;size:255"`
	ExpiresAt time.Time      `json:"expires_at" gorm:"not null"`
	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

type UserStatistics struct {
	ID               uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID           uuid.UUID      `json:"user_id" gorm:"type:uuid;not null;index"`
	User             User           `json:"-" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	StatType         string         `json:"stat_type" gorm:"not null;size:30"` // 'cash_games', 'tournaments', 'sitng'
	TotalSessions    int            `json:"total_sessions" gorm:"default:0"`
	TotalHands       int            `json:"total_hands" gorm:"default:0"`
	TotalWinnings    int64          `json:"total_winnings" gorm:"default:0"` // MNT
	BestSession      int64          `json:"best_session" gorm:"default:0"`   // MNT
	AvgSessionLength *time.Duration `json:"avg_session_length"`
	LastUpdated      time.Time      `json:"last_updated" gorm:"autoUpdateTime"`
	CreatedAt        time.Time      `json:"created_at" gorm:"autoCreateTime"`
	DeletedAt        gorm.DeletedAt `json:"-" gorm:"index"`
}

// Add unique constraint for user_id + stat_type
func (UserStatistics) TableName() string {
	return "user_statistics"
}