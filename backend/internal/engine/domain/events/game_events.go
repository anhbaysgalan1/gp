package events

import (
	"github.com/google/uuid"
)

// Card represents a playing card in the game
type Card struct {
	Suit  string `json:"suit"`
	Rank  string `json:"rank"`
	Value int    `json:"value"`
}

// HandStarted event is emitted when a new hand begins
type HandStarted struct {
	BaseEvent
	HandID      uuid.UUID   `json:"hand_id"`
	DealerSeat  int         `json:"dealer_seat"`
	SmallBlind  int         `json:"small_blind_seat"`
	BigBlind    int         `json:"big_blind_seat"`
	Players     []uuid.UUID `json:"players"`
	SmallBlindAmount int64  `json:"small_blind_amount"`
	BigBlindAmount   int64  `json:"big_blind_amount"`
}

// NewHandStarted creates a new HandStarted event
func NewHandStarted(tableID, handID uuid.UUID, dealerSeat, smallBlind, bigBlind int, players []uuid.UUID, sbAmount, bbAmount int64, version int64) *HandStarted {
	return &HandStarted{
		BaseEvent:        NewBaseEvent(HandStartedEvent, tableID, version, nil),
		HandID:           handID,
		DealerSeat:       dealerSeat,
		SmallBlind:       smallBlind,
		BigBlind:         bigBlind,
		Players:          players,
		SmallBlindAmount: sbAmount,
		BigBlindAmount:   bbAmount,
	}
}

// CardsDealt event is emitted when hole cards are dealt to players
type CardsDealt struct {
	BaseEvent
	HandID      uuid.UUID              `json:"hand_id"`
	PlayerCards map[uuid.UUID][]Card   `json:"player_cards"`
}

// NewCardsDealt creates a new CardsDealt event
func NewCardsDealt(tableID, handID uuid.UUID, playerCards map[uuid.UUID][]Card, version int64) *CardsDealt {
	return &CardsDealt{
		BaseEvent:   NewBaseEvent(CardsDealtEvent, tableID, version, nil),
		HandID:      handID,
		PlayerCards: playerCards,
	}
}

// CommunityCardsDealt event is emitted when community cards are revealed
type CommunityCardsDealt struct {
	BaseEvent
	HandID      uuid.UUID `json:"hand_id"`
	Stage       string    `json:"stage"` // "flop", "turn", "river"
	Cards       []Card    `json:"cards"`
}

// NewCommunityCardsDealt creates a new CommunityCardsDealt event
func NewCommunityCardsDealt(tableID, handID uuid.UUID, stage string, cards []Card, version int64) *CommunityCardsDealt {
	return &CommunityCardsDealt{
		BaseEvent: NewBaseEvent(CommunityCardsDealtEvent, tableID, version, nil),
		HandID:    handID,
		Stage:     stage,
		Cards:     cards,
	}
}

// PlayerAction represents a player's action in the game
type PlayerAction struct {
	BaseEvent
	HandID      uuid.UUID `json:"hand_id"`
	PlayerID    uuid.UUID `json:"player_id"`
	Action      string    `json:"action"` // "bet", "call", "raise", "check", "fold", "all_in"
	Amount      int64     `json:"amount"`
	TotalBet    int64     `json:"total_bet"`
	RemainingChips int64  `json:"remaining_chips"`
	IsAllIn     bool      `json:"is_all_in"`
}

// NewPlayerBet creates a new PlayerAction event for a bet
func NewPlayerBet(tableID, handID, playerID uuid.UUID, amount, totalBet, remainingChips int64, isAllIn bool, version int64) *PlayerAction {
	actionType := PlayerBetEvent
	if isAllIn {
		actionType = PlayerAllInEvent
	}

	return &PlayerAction{
		BaseEvent:      NewBaseEvent(actionType, tableID, version, &playerID),
		HandID:         handID,
		PlayerID:       playerID,
		Action:         "bet",
		Amount:         amount,
		TotalBet:       totalBet,
		RemainingChips: remainingChips,
		IsAllIn:        isAllIn,
	}
}

// NewPlayerCall creates a new PlayerAction event for a call
func NewPlayerCall(tableID, handID, playerID uuid.UUID, amount, totalBet, remainingChips int64, isAllIn bool, version int64) *PlayerAction {
	actionType := PlayerCallEvent
	if isAllIn {
		actionType = PlayerAllInEvent
	}

	return &PlayerAction{
		BaseEvent:      NewBaseEvent(actionType, tableID, version, &playerID),
		HandID:         handID,
		PlayerID:       playerID,
		Action:         "call",
		Amount:         amount,
		TotalBet:       totalBet,
		RemainingChips: remainingChips,
		IsAllIn:        isAllIn,
	}
}

// NewPlayerRaise creates a new PlayerAction event for a raise
func NewPlayerRaise(tableID, handID, playerID uuid.UUID, amount, totalBet, remainingChips int64, isAllIn bool, version int64) *PlayerAction {
	actionType := PlayerRaiseEvent
	if isAllIn {
		actionType = PlayerAllInEvent
	}

	return &PlayerAction{
		BaseEvent:      NewBaseEvent(actionType, tableID, version, &playerID),
		HandID:         handID,
		PlayerID:       playerID,
		Action:         "raise",
		Amount:         amount,
		TotalBet:       totalBet,
		RemainingChips: remainingChips,
		IsAllIn:        isAllIn,
	}
}

// NewPlayerCheck creates a new PlayerAction event for a check
func NewPlayerCheck(tableID, handID, playerID uuid.UUID, version int64) *PlayerAction {
	return &PlayerAction{
		BaseEvent: NewBaseEvent(PlayerCheckEvent, tableID, version, &playerID),
		HandID:    handID,
		PlayerID:  playerID,
		Action:    "check",
		Amount:    0,
	}
}

// NewPlayerFold creates a new PlayerAction event for a fold
func NewPlayerFold(tableID, handID, playerID uuid.UUID, version int64) *PlayerAction {
	return &PlayerAction{
		BaseEvent: NewBaseEvent(PlayerFoldEvent, tableID, version, &playerID),
		HandID:    handID,
		PlayerID:  playerID,
		Action:    "fold",
		Amount:    0,
	}
}

// HandEnded event is emitted when a hand is completed
type HandEnded struct {
	BaseEvent
	HandID      uuid.UUID               `json:"hand_id"`
	Winners     []uuid.UUID             `json:"winners"`
	Pots        []PotResult             `json:"pots"`
	PlayerCards map[uuid.UUID][]Card    `json:"player_cards,omitempty"`
	WinType     string                  `json:"win_type"` // "showdown", "fold"
}

// PotResult represents the result of a pot
type PotResult struct {
	PotID       uuid.UUID   `json:"pot_id"`
	Amount      int64       `json:"amount"`
	Winners     []uuid.UUID `json:"winners"`
	WinningHand []Card      `json:"winning_hand,omitempty"`
	HandRank    string      `json:"hand_rank,omitempty"`
}

// NewHandEnded creates a new HandEnded event
func NewHandEnded(tableID, handID uuid.UUID, winners []uuid.UUID, pots []PotResult, playerCards map[uuid.UUID][]Card, winType string, version int64) *HandEnded {
	return &HandEnded{
		BaseEvent:   NewBaseEvent(HandEndedEvent, tableID, version, nil),
		HandID:      handID,
		Winners:     winners,
		Pots:        pots,
		PlayerCards: playerCards,
		WinType:     winType,
	}
}