package events

import (
	"github.com/google/uuid"
)

// BuyIn event is emitted when a player buys into a table
type BuyIn struct {
	BaseEvent
	PlayerID      uuid.UUID `json:"player_id"`
	SessionID     uuid.UUID `json:"session_id"`
	Amount        int64     `json:"amount"`
	TransactionID string    `json:"transaction_id"`
	SeatNumber    int       `json:"seat_number"`
}

// NewBuyIn creates a new BuyIn event
func NewBuyIn(tableID, playerID, sessionID uuid.UUID, amount int64, transactionID string, seatNumber int, version int64) *BuyIn {
	return &BuyIn{
		BaseEvent:     NewBaseEvent(BuyInEvent, tableID, version, &playerID),
		PlayerID:      playerID,
		SessionID:     sessionID,
		Amount:        amount,
		TransactionID: transactionID,
		SeatNumber:    seatNumber,
	}
}

// CashOut event is emitted when a player cashes out from a table
type CashOut struct {
	BaseEvent
	PlayerID      uuid.UUID `json:"player_id"`
	SessionID     uuid.UUID `json:"session_id"`
	Amount        int64     `json:"amount"`
	TransactionID string    `json:"transaction_id"`
	Reason        string    `json:"reason"` // "manual", "leaving_table", "session_end"
}

// NewCashOut creates a new CashOut event
func NewCashOut(tableID, playerID, sessionID uuid.UUID, amount int64, transactionID, reason string, version int64) *CashOut {
	return &CashOut{
		BaseEvent:     NewBaseEvent(CashOutEvent, tableID, version, &playerID),
		PlayerID:      playerID,
		SessionID:     sessionID,
		Amount:        amount,
		TransactionID: transactionID,
		Reason:        reason,
	}
}

// WinningsDistributed event is emitted when pot winnings are distributed to players
type WinningsDistributed struct {
	BaseEvent
	HandID        uuid.UUID           `json:"hand_id"`
	Distributions []WinningDistribution `json:"distributions"`
}

// WinningDistribution represents winnings distributed to a single player
type WinningDistribution struct {
	PlayerID      uuid.UUID `json:"player_id"`
	SessionID     uuid.UUID `json:"session_id"`
	Amount        int64     `json:"amount"`
	PotID         uuid.UUID `json:"pot_id"`
	TransactionID string    `json:"transaction_id"`
	HandRank      string    `json:"hand_rank,omitempty"`
	WinningHand   []Card    `json:"winning_hand,omitempty"`
}

// NewWinningsDistributed creates a new WinningsDistributed event
func NewWinningsDistributed(tableID, handID uuid.UUID, distributions []WinningDistribution, version int64) *WinningsDistributed {
	return &WinningsDistributed{
		BaseEvent:     NewBaseEvent(WinningsDistributedEvent, tableID, version, nil),
		HandID:        handID,
		Distributions: distributions,
	}
}