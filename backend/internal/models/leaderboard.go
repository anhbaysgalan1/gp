package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type LeaderboardEntry struct {
	ID               uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID           uuid.UUID      `json:"user_id" gorm:"type:uuid;not null;index"`
	User             User           `json:"user,omitempty" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	LeaderboardType  string         `json:"leaderboard_type" gorm:"not null;size:20;index"` // 'daily', 'weekly', 'monthly', 'alltime'
	PeriodStart      time.Time      `json:"period_start" gorm:"not null;type:date"`
	PeriodEnd        time.Time      `json:"period_end" gorm:"not null;type:date"`
	TotalWinnings    int64          `json:"total_winnings" gorm:"default:0"`       // MNT
	HandsPlayed      int            `json:"hands_played" gorm:"default:0"`
	TournamentsWon   int            `json:"tournaments_won" gorm:"default:0"`
	GamesPlayed      int            `json:"games_played" gorm:"default:0"`
	CalculatedAt     time.Time      `json:"calculated_at" gorm:"autoCreateTime"`
	CreatedAt        time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt        time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt        gorm.DeletedAt `json:"-" gorm:"index"`

	// Joined fields from user table (populated when querying)
	Username  string  `json:"username,omitempty" gorm:"-"`
	AvatarURL *string `json:"avatar_url,omitempty" gorm:"-"`
}

type LeaderboardResponse struct {
	Type    string             `json:"type"`
	Period  string             `json:"period"`
	Entries []LeaderboardEntry `json:"entries"`
	UserRank *int              `json:"user_rank,omitempty"`
}

type UserBalance struct {
	MainBalance int64 `json:"main_balance"` // MNT
	GameBalance int64 `json:"game_balance"` // MNT
	TotalBalance int64 `json:"total_balance"` // MNT
}