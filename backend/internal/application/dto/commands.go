package dto

import (
	"github.com/anhbaysgalan1/gp/internal/engine/domain/table"
	"github.com/google/uuid"
)

// Commands for CQRS pattern
type CreateTableCommand struct {
	Name       string            `json:"name"`
	Type       table.TableType   `json:"type"`
	MaxPlayers int               `json:"max_players"`
	SmallBlind int64             `json:"small_blind"`
	BigBlind   int64             `json:"big_blind"`
	Config     table.TableConfig `json:"config"`
	CreatedBy  uuid.UUID         `json:"created_by"`
}

type JoinTableCommand struct {
	TableID  uuid.UUID `json:"table_id"`
	PlayerID uuid.UUID `json:"player_id"`
	Username string    `json:"username"`
	Avatar   string    `json:"avatar,omitempty"`
}

type LeaveTableCommand struct {
	TableID  uuid.UUID `json:"table_id"`
	PlayerID uuid.UUID `json:"player_id"`
	Reason   string    `json:"reason"`
}

type SeatPlayerCommand struct {
	TableID     uuid.UUID `json:"table_id"`
	PlayerID    uuid.UUID `json:"player_id"`
	SessionID   uuid.UUID `json:"session_id"`
	SeatNumber  int       `json:"seat_number"`
	BuyInAmount int64     `json:"buy_in_amount"`
}

type PlayerActionCommand struct {
	TableID  uuid.UUID `json:"table_id"`
	PlayerID uuid.UUID `json:"player_id"`
	Action   string    `json:"action"` // "bet", "call", "raise", "check", "fold"
	Amount   int64     `json:"amount,omitempty"`
}
