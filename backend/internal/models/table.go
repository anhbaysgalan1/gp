package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PokerTable struct {
	ID             uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name           string         `json:"name" gorm:"uniqueIndex;not null;size:100"`
	TableType      string         `json:"table_type" gorm:"not null;size:20;index"` // 'cash', 'tournament', 'sitng'
	GameType       string         `json:"game_type" gorm:"not null;size:20;default:texas_holdem"` // 'texas_holdem', 'omaha'
	MaxPlayers     int            `json:"max_players" gorm:"not null;default:9"`
	MinBuyIn       int64          `json:"min_buy_in" gorm:"not null"`     // MNT
	MaxBuyIn       int64          `json:"max_buy_in" gorm:"not null"`     // MNT
	SmallBlind     int64          `json:"small_blind" gorm:"not null"`    // MNT
	BigBlind       int64          `json:"big_blind" gorm:"not null"`      // MNT
	IsPrivate      bool           `json:"is_private" gorm:"default:false"`
	PasswordHash   *string        `json:"-" gorm:"size:255"`
	Status         string         `json:"status" gorm:"not null;size:20;default:waiting;index"` // 'waiting', 'active', 'finished'
	CurrentPlayers int            `json:"current_players" gorm:"default:0"`
	CreatedBy      uuid.UUID      `json:"created_by" gorm:"type:uuid;not null;index"`
	Creator        User           `json:"creator,omitempty" gorm:"foreignKey:CreatedBy"`
	CreatedAt      time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt      gorm.DeletedAt `json:"-" gorm:"index"`
}

type CreateTableRequest struct {
	Name       string `json:"name" validate:"required,min=3,max=100"`
	TableType  string `json:"table_type" validate:"required,oneof=cash tournament sitng"`
	GameType   string `json:"game_type" validate:"required,oneof=texas_holdem omaha"`
	MaxPlayers int    `json:"max_players" validate:"required,min=2,max=9"`
	MinBuyIn   int64  `json:"min_buy_in" validate:"required,min=1"`
	MaxBuyIn   int64  `json:"max_buy_in" validate:"required,gtfield=MinBuyIn"`
	SmallBlind int64  `json:"small_blind" validate:"required,min=1"`
	BigBlind   int64  `json:"big_blind" validate:"required,gtfield=SmallBlind"`
	IsPrivate  bool   `json:"is_private"`
	Password   string `json:"password,omitempty" validate:"omitempty,min=4"`
}

type Tournament struct {
	ID                uuid.UUID       `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name              string          `json:"name" gorm:"not null;size:100"`
	TournamentType    string          `json:"tournament_type" gorm:"not null;size:20;index"` // 'scheduled', 'sitng'
	BuyIn             int64           `json:"buy_in" gorm:"not null"`                         // MNT
	PrizePool         int64           `json:"prize_pool" gorm:"default:0"`                   // MNT
	MaxPlayers        int             `json:"max_players" gorm:"not null"`
	RegisteredPlayers int             `json:"registered_players" gorm:"default:0"`
	Status            string          `json:"status" gorm:"not null;size:20;default:registering;index"` // 'registering', 'running', 'finished'
	StartTime         *time.Time      `json:"start_time" gorm:"index"`
	EndTime           *time.Time      `json:"end_time"`
	BlindStructure    json.RawMessage `json:"blind_structure" gorm:"type:jsonb"`
	PayoutStructure   json.RawMessage `json:"payout_structure" gorm:"type:jsonb"`
	CreatedAt         time.Time       `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt         time.Time       `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt         gorm.DeletedAt  `json:"-" gorm:"index"`
}

type CreateTournamentRequest struct {
	Name            string          `json:"name" validate:"required,min=3,max=100"`
	TournamentType  string          `json:"tournament_type" validate:"required,oneof=scheduled sitng"`
	BuyIn           int64           `json:"buy_in" validate:"required,min=1"`
	MaxPlayers      int             `json:"max_players" validate:"required,min=2,max=1000"`
	StartTime       *time.Time      `json:"start_time,omitempty"`
	BlindStructure  json.RawMessage `json:"blind_structure" validate:"required"`
	PayoutStructure json.RawMessage `json:"payout_structure" validate:"required"`
}

type TournamentRegistration struct {
	ID                   uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TournamentID         uuid.UUID      `json:"tournament_id" gorm:"type:uuid;not null;index"`
	Tournament           Tournament     `json:"tournament,omitempty" gorm:"foreignKey:TournamentID;constraint:OnDelete:CASCADE"`
	UserID               uuid.UUID      `json:"user_id" gorm:"type:uuid;not null;index"`
	User                 User           `json:"user,omitempty" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	BuyInTransactionID   *string        `json:"buy_in_transaction_id" gorm:"size:255"`
	FinalPosition        *int           `json:"final_position"`
	PrizeAmount          int64          `json:"prize_amount" gorm:"default:0"` // MNT
	RegisteredAt         time.Time      `json:"registered_at" gorm:"autoCreateTime"`
	CreatedAt            time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt            time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt            gorm.DeletedAt `json:"-" gorm:"index"`
}

// Add composite unique index for tournament_id + user_id
func (TournamentRegistration) TableName() string {
	return "tournament_registrations"
}

