package table

import (
	"time"

	"github.com/anhbaysgalan1/gp/internal/engine/domain/game"
	"github.com/google/uuid"
)

// TableStatus represents the current state of a table
type TableStatus string

const (
	TableStatusWaiting TableStatus = "waiting"
	TableStatusActive  TableStatus = "active"
	TableStatusPaused  TableStatus = "paused"
	TableStatusClosed  TableStatus = "closed"
)

// TableType represents the type of poker table
type TableType string

const (
	TableTypeCashGame   TableType = "cash_game"
	TableTypeTournament TableType = "tournament"
	TableTypeSitAndGo   TableType = "sit_and_go"
)

// Table represents a poker table
type Table struct {
	ID         uuid.UUID   `json:"id"`
	Name       string      `json:"name"`
	Type       TableType   `json:"type"`
	Status     TableStatus `json:"status"`
	MaxPlayers int         `json:"max_players"`
	SmallBlind int64       `json:"small_blind"`
	BigBlind   int64       `json:"big_blind"`
	MaxBuyIn   int64       `json:"max_buy_in"`
	MinBuyIn   int64       `json:"min_buy_in"`
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at"`

	// Current game state
	Game *game.Game `json:"game,omitempty"`

	// Table configuration
	Config TableConfig `json:"config"`

	// Metadata for additional table information
	Metadata map[string]string `json:"metadata,omitempty"`
}

// TableConfig contains configuration settings for the table
type TableConfig struct {
	MaxBuyIn        int64         `json:"max_buy_in"`
	MinBuyIn        int64         `json:"min_buy_in"`
	HandTimeout     time.Duration `json:"hand_timeout"`
	ActionTimeout   time.Duration `json:"action_timeout"`
	AllowSpectators bool          `json:"allow_spectators"`
	RequireAuth     bool          `json:"require_auth"`
	IsPrivate       bool          `json:"is_private"`
	RakePercentage  float64       `json:"rake_percentage,omitempty"`
	MaxRake         int64         `json:"max_rake,omitempty"`
}

// Seat represents a seat at the table
type Seat struct {
	Number     int        `json:"number"`
	PlayerID   *uuid.UUID `json:"player_id,omitempty"`
	IsOccupied bool       `json:"is_occupied"`
	ReservedAt *time.Time `json:"reserved_at,omitempty"`
	ReservedBy *uuid.UUID `json:"reserved_by,omitempty"`
}

// PlayerSession represents a player's session at the table
type PlayerSession struct {
	ID           uuid.UUID  `json:"id"`
	PlayerID     uuid.UUID  `json:"player_id"`
	Username     string     `json:"username"`
	SeatNumber   int        `json:"seat_number"`
	BuyInAmount  int64      `json:"buy_in_amount"`
	CurrentChips int64      `json:"current_chips"`
	JoinedAt     time.Time  `json:"joined_at"`
	LastAction   *time.Time `json:"last_action,omitempty"`
	IsActive     bool       `json:"is_active"`
}

// NewTable creates a new poker table
func NewTable(name string, tableType TableType, maxPlayers int, smallBlind, bigBlind int64, config TableConfig) *Table {
	tableID := uuid.New()

	// Set default min/max buy-in if not specified
	if config.MinBuyIn == 0 {
		config.MinBuyIn = bigBlind * 20 // 20 big blinds minimum
	}
	if config.MaxBuyIn == 0 {
		config.MaxBuyIn = bigBlind * 200 // 200 big blinds maximum
	}

	// Set default timeouts if not specified
	if config.HandTimeout == 0 {
		config.HandTimeout = 30 * time.Second
	}
	if config.ActionTimeout == 0 {
		config.ActionTimeout = 30 * time.Second
	}

	return &Table{
		ID:         tableID,
		Name:       name,
		Type:       tableType,
		Status:     TableStatusWaiting,
		MaxPlayers: maxPlayers,
		SmallBlind: smallBlind,
		BigBlind:   bigBlind,
		MaxBuyIn:   config.MaxBuyIn,
		MinBuyIn:   config.MinBuyIn,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Game:       game.NewGame(tableID, smallBlind, bigBlind, maxPlayers),
		Config:     config,
		Metadata:   make(map[string]string),
	}
}

// GetAvailableSeats returns the list of available seat numbers
func (t *Table) GetAvailableSeats() []int {
	availableSeats := make([]int, 0)
	occupiedSeats := make(map[int]bool)

	// Mark occupied seats
	if t.Game != nil {
		for _, player := range t.Game.Players {
			if player.IsActive {
				occupiedSeats[player.SeatNumber] = true
			}
		}
	}

	// Find available seats
	for i := 1; i <= t.MaxPlayers; i++ {
		if !occupiedSeats[i] {
			availableSeats = append(availableSeats, i)
		}
	}

	return availableSeats
}

// GetPlayerCount returns the number of active players at the table
func (t *Table) GetPlayerCount() int {
	if t.Game == nil {
		return 0
	}
	return len(t.Game.GetActivePlayers())
}

// IsFull returns true if the table is full
func (t *Table) IsFull() bool {
	return t.GetPlayerCount() >= t.MaxPlayers
}

// CanStart returns true if the table can start a game
func (t *Table) CanStart() bool {
	return t.Status == TableStatusWaiting && t.GetPlayerCount() >= 2
}

// IsValidBuyIn returns true if the buy-in amount is valid for this table
func (t *Table) IsValidBuyIn(amount int64) bool {
	return amount >= t.MinBuyIn && amount <= t.MaxBuyIn
}

// UpdateStatus updates the table status and timestamp
func (t *Table) UpdateStatus(status TableStatus) {
	t.Status = status
	t.UpdatedAt = time.Now()
}

// CalculateRake calculates the rake amount for a given pot
func (t *Table) CalculateRake(potAmount int64) int64 {
	if t.Config.RakePercentage == 0 {
		return 0
	}

	rake := int64(float64(potAmount) * t.Config.RakePercentage / 100.0)

	// Apply maximum rake limit if specified
	if t.Config.MaxRake > 0 && rake > t.Config.MaxRake {
		rake = t.Config.MaxRake
	}

	return rake
}

// GetTableInfo returns a summary of table information
func (t *Table) GetTableInfo() TableInfo {
	return TableInfo{
		ID:          t.ID,
		Name:        t.Name,
		Type:        t.Type,
		Status:      t.Status,
		MaxPlayers:  t.MaxPlayers,
		PlayerCount: t.GetPlayerCount(),
		SmallBlind:  t.SmallBlind,
		BigBlind:    t.BigBlind,
		MinBuyIn:    t.MinBuyIn,
		MaxBuyIn:    t.MaxBuyIn,
		IsPrivate:   t.Config.IsPrivate,
		CreatedAt:   t.CreatedAt,
	}
}

// TableInfo represents a summary of table information
type TableInfo struct {
	ID          uuid.UUID   `json:"id"`
	Name        string      `json:"name"`
	Type        TableType   `json:"type"`
	Status      TableStatus `json:"status"`
	MaxPlayers  int         `json:"max_players"`
	PlayerCount int         `json:"player_count"`
	SmallBlind  int64       `json:"small_blind"`
	BigBlind    int64       `json:"big_blind"`
	MinBuyIn    int64       `json:"min_buy_in"`
	MaxBuyIn    int64       `json:"max_buy_in"`
	IsPrivate   bool        `json:"is_private"`
	CreatedAt   time.Time   `json:"created_at"`
}
