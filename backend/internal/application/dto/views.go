package dto

import (
	"github.com/anhbaysgalan1/gp/internal/engine/domain/table"
	"github.com/google/uuid"
)

// Views for queries
type GameStateView struct {
	TableID        uuid.UUID         `json:"table_id"`
	TableName      string            `json:"table_name"`
	Status         table.TableStatus `json:"status"`
	Stage          string            `json:"stage"`
	IsRunning      bool              `json:"is_running"`
	HandID         *uuid.UUID        `json:"hand_id,omitempty"`
	Players        []PlayerView      `json:"players"`
	CommunityCards []CardView        `json:"community_cards"`
	Pots           []PotView         `json:"pots"`
	DealerSeat     int               `json:"dealer_seat"`
	ActionSeat     int               `json:"action_seat"`
	CurrentBet     int64             `json:"current_bet"`
	MinRaise       int64             `json:"min_raise"`
	SmallBlind     int64             `json:"small_blind"`
	BigBlind       int64             `json:"big_blind"`
	HandNumber     int64             `json:"hand_number"`
}

type PlayerView struct {
	ID           uuid.UUID  `json:"id"`
	Username     string     `json:"username"`
	SeatNumber   int        `json:"seat_number"`
	Chips        int64      `json:"chips"`
	CurrentBet   int64      `json:"current_bet"`
	TotalBet     int64      `json:"total_bet"`
	IsActive     bool       `json:"is_active"`
	IsFolded     bool       `json:"is_folded"`
	IsAllIn      bool       `json:"is_all_in"`
	HasActed     bool       `json:"has_acted"`
	IsDealer     bool       `json:"is_dealer"`
	IsSmallBlind bool       `json:"is_small_blind"`
	IsBigBlind   bool       `json:"is_big_blind"`
	HoleCards    []CardView `json:"hole_cards,omitempty"`
}

type CardView struct {
	Suit  string `json:"suit"`
	Rank  string `json:"rank"`
	Value int    `json:"value"`
}

type PotView struct {
	ID              uuid.UUID   `json:"id"`
	Amount          int64       `json:"amount"`
	EligiblePlayers []uuid.UUID `json:"eligible_players"`
	WinningPlayers  []uuid.UUID `json:"winning_players,omitempty"`
	IsSidePot       bool        `json:"is_side_pot"`
}

type HandHistory struct {
	HandID         uuid.UUID           `json:"hand_id"`
	TableID        uuid.UUID           `json:"table_id"`
	HandNumber     int64               `json:"hand_number"`
	StartTime      string              `json:"start_time"`
	EndTime        string              `json:"end_time,omitempty"`
	DealerSeat     int                 `json:"dealer_seat"`
	SmallBlind     int64               `json:"small_blind"`
	BigBlind       int64               `json:"big_blind"`
	Players        []HandHistoryPlayer `json:"players"`
	CommunityCards []CardView          `json:"community_cards"`
	Actions        []HandHistoryAction `json:"actions"`
	Pots           []HandHistoryPot    `json:"pots"`
	Winners        []HandHistoryWinner `json:"winners"`
}

type HandHistoryPlayer struct {
	PlayerID   uuid.UUID  `json:"player_id"`
	Username   string     `json:"username"`
	SeatNumber int        `json:"seat_number"`
	StartChips int64      `json:"start_chips"`
	EndChips   int64      `json:"end_chips"`
	HoleCards  []CardView `json:"hole_cards,omitempty"`
}

type HandHistoryAction struct {
	PlayerID  uuid.UUID `json:"player_id"`
	Action    string    `json:"action"`
	Amount    int64     `json:"amount"`
	Stage     string    `json:"stage"`
	Timestamp string    `json:"timestamp"`
}

type HandHistoryPot struct {
	Amount          int64       `json:"amount"`
	EligiblePlayers []uuid.UUID `json:"eligible_players"`
}

type HandHistoryWinner struct {
	PlayerID    uuid.UUID  `json:"player_id"`
	Amount      int64      `json:"amount"`
	HandRank    string     `json:"hand_rank"`
	WinningHand []CardView `json:"winning_hand"`
}
