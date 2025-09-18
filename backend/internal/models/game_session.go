package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GameSessionStatus string

const (
	GameSessionStatusActive   GameSessionStatus = "active"
	GameSessionStatusFinished GameSessionStatus = "finished"
	GameSessionStatusAbandoned GameSessionStatus = "abandoned"
)

// GameSession represents an active game session for a user at a table
type GameSession struct {
	ID           uuid.UUID         `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID       uuid.UUID         `json:"user_id" gorm:"type:uuid;not null;index"`
	TableID      uuid.UUID         `json:"table_id" gorm:"type:uuid;not null;index"`
	BuyInAmount  int64             `json:"buy_in_amount" gorm:"not null"`
	CurrentChips int64             `json:"current_chips" gorm:"not null"`
	Status       GameSessionStatus `json:"status" gorm:"type:varchar(20);not null;default:'active'"`
	SeatNumber   *int              `json:"seat_number,omitempty" gorm:"index"`
	JoinedAt     time.Time         `json:"joined_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	LeftAt       *time.Time        `json:"left_at,omitempty"`
	CreatedAt    time.Time         `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time         `json:"updated_at" gorm:"autoUpdateTime"`

	// Relations
	User  User        `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Table PokerTable  `json:"table,omitempty" gorm:"foreignKey:TableID"`
}

// BeforeCreate sets the ID if not already set
func (gs *GameSession) BeforeCreate(tx *gorm.DB) error {
	if gs.ID == uuid.Nil {
		gs.ID = uuid.New()
	}
	return nil
}

// IsActive returns true if the session is active
func (gs *GameSession) IsActive() bool {
	return gs.Status == GameSessionStatusActive
}

// GetNetResult returns the net result (profit/loss) for this session
func (gs *GameSession) GetNetResult() int64 {
	return gs.CurrentChips - gs.BuyInAmount
}

// Finish marks the session as finished and sets the left_at timestamp
func (gs *GameSession) Finish() {
	gs.Status = GameSessionStatusFinished
	now := time.Now()
	gs.LeftAt = &now
}

// Abandon marks the session as abandoned (for unexpected disconnections)
func (gs *GameSession) Abandon() {
	gs.Status = GameSessionStatusAbandoned
	now := time.Now()
	gs.LeftAt = &now
}