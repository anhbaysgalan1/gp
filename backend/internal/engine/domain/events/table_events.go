package events

import (
	"github.com/google/uuid"
)

// TableCreated event is emitted when a new table is created
type TableCreated struct {
	BaseEvent
	TableName   string            `json:"table_name"`
	MaxPlayers  int               `json:"max_players"`
	SmallBlind  int64             `json:"small_blind"`
	BigBlind    int64             `json:"big_blind"`
	MaxBuyIn    int64             `json:"max_buy_in"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// NewTableCreated creates a new TableCreated event
func NewTableCreated(tableID uuid.UUID, tableName string, maxPlayers int, smallBlind, bigBlind, maxBuyIn int64, metadata map[string]string) *TableCreated {
	return &TableCreated{
		BaseEvent:   NewBaseEvent(TableCreatedEvent, tableID, 1, nil),
		TableName:   tableName,
		MaxPlayers:  maxPlayers,
		SmallBlind:  smallBlind,
		BigBlind:    bigBlind,
		MaxBuyIn:    maxBuyIn,
		Metadata:    metadata,
	}
}

// PlayerJoined event is emitted when a player joins a table
type PlayerJoined struct {
	BaseEvent
	PlayerID    uuid.UUID `json:"player_id"`
	Username    string    `json:"username"`
	Avatar      string    `json:"avatar,omitempty"`
}

// NewPlayerJoined creates a new PlayerJoined event
func NewPlayerJoined(tableID, playerID uuid.UUID, username, avatar string, version int64) *PlayerJoined {
	return &PlayerJoined{
		BaseEvent:   NewBaseEvent(PlayerJoinedEvent, tableID, version, &playerID),
		PlayerID:    playerID,
		Username:    username,
		Avatar:      avatar,
	}
}

// PlayerLeft event is emitted when a player leaves a table
type PlayerLeft struct {
	BaseEvent
	PlayerID    uuid.UUID `json:"player_id"`
	Reason      string    `json:"reason"`
	FinalChips  int64     `json:"final_chips"`
}

// NewPlayerLeft creates a new PlayerLeft event
func NewPlayerLeft(tableID, playerID uuid.UUID, reason string, finalChips int64, version int64) *PlayerLeft {
	return &PlayerLeft{
		BaseEvent:   NewBaseEvent(PlayerLeftEvent, tableID, version, &playerID),
		PlayerID:    playerID,
		Reason:      reason,
		FinalChips:  finalChips,
	}
}

// PlayerSeated event is emitted when a player takes a seat at the table
type PlayerSeated struct {
	BaseEvent
	PlayerID    uuid.UUID `json:"player_id"`
	SeatNumber  int       `json:"seat_number"`
	BuyInAmount int64     `json:"buy_in_amount"`
	SessionID   uuid.UUID `json:"session_id"`
}

// NewPlayerSeated creates a new PlayerSeated event
func NewPlayerSeated(tableID, playerID, sessionID uuid.UUID, seatNumber int, buyInAmount int64, version int64) *PlayerSeated {
	return &PlayerSeated{
		BaseEvent:   NewBaseEvent(PlayerSeatedEvent, tableID, version, &playerID),
		PlayerID:    playerID,
		SeatNumber:  seatNumber,
		BuyInAmount: buyInAmount,
		SessionID:   sessionID,
	}
}