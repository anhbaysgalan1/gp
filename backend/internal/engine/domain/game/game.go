package game

import (
	"fmt"
	"sort"

	"github.com/google/uuid"
)

// GameStage represents the current stage of a poker hand
type GameStage int

const (
	PreDeal GameStage = iota
	PreFlop
	Flop
	Turn
	River
	Showdown
)

func (gs GameStage) String() string {
	switch gs {
	case PreDeal:
		return "pre_deal"
	case PreFlop:
		return "pre_flop"
	case Flop:
		return "flop"
	case Turn:
		return "turn"
	case River:
		return "river"
	case Showdown:
		return "showdown"
	default:
		return "unknown"
	}
}

// Card represents a playing card
type Card struct {
	Suit  string `json:"suit"`
	Rank  string `json:"rank"`
	Value int    `json:"value"`
}

// String returns a string representation of the card
func (c Card) String() string {
	return fmt.Sprintf("%s%s", c.Rank, c.Suit)
}

// Pot represents a pot in the game (main pot or side pot)
type Pot struct {
	ID               uuid.UUID   `json:"id"`
	Amount           int64       `json:"amount"`
	EligiblePlayers  []uuid.UUID `json:"eligible_players"`
	WinningPlayers   []uuid.UUID `json:"winning_players,omitempty"`
	WinningHand      []Card      `json:"winning_hand,omitempty"`
	HandRank         string      `json:"hand_rank,omitempty"`
	IsSidePot        bool        `json:"is_side_pot"`
	MaxContribution  int64       `json:"max_contribution,omitempty"`
}

// Player represents a player in the game
type Player struct {
	ID             uuid.UUID `json:"id"`
	Username       string    `json:"username"`
	SeatNumber     int       `json:"seat_number"`
	Chips          int64     `json:"chips"`
	CurrentBet     int64     `json:"current_bet"`
	TotalBet       int64     `json:"total_bet"`
	HoleCards      []Card    `json:"hole_cards,omitempty"`
	IsActive       bool      `json:"is_active"`
	IsFolded       bool      `json:"is_folded"`
	IsAllIn        bool      `json:"is_all_in"`
	HasActed       bool      `json:"has_acted"`
	SessionID      uuid.UUID `json:"session_id"`
	Position       PlayerPosition `json:"position"`
}

// PlayerPosition represents a player's position in the current hand
type PlayerPosition struct {
	IsDealer    bool `json:"is_dealer"`
	IsSmallBlind bool `json:"is_small_blind"`
	IsBigBlind  bool `json:"is_big_blind"`
	IsUTG       bool `json:"is_utg"`
}

// CanAct returns true if the player can take an action
func (p *Player) CanAct() bool {
	return p.IsActive && !p.IsFolded && !p.IsAllIn
}

// IsInHand returns true if the player is still in the current hand
func (p *Player) IsInHand() bool {
	return p.IsActive && !p.IsFolded
}

// Game represents the state of a poker game
type Game struct {
	ID             uuid.UUID     `json:"id"`
	TableID        uuid.UUID     `json:"table_id"`
	HandID         *uuid.UUID    `json:"hand_id,omitempty"`
	Players        []*Player     `json:"players"`
	CommunityCards []Card        `json:"community_cards"`
	Pots           []Pot         `json:"pots"`
	Stage          GameStage     `json:"stage"`
	IsRunning      bool          `json:"is_running"`
	DealerSeat     int           `json:"dealer_seat"`
	ActionSeat     int           `json:"action_seat"`
	SmallBlind     int64         `json:"small_blind"`
	BigBlind       int64         `json:"big_blind"`
	MinRaise       int64         `json:"min_raise"`
	MaxPlayers     int           `json:"max_players"`
	HandNumber     int64         `json:"hand_number"`
	Deck           *Deck         `json:"-"` // Don't serialize deck
	Actions        *GameActions  `json:"-"` // Game actions helper
}

// NewGame creates a new poker game
func NewGame(tableID uuid.UUID, smallBlind, bigBlind int64, maxPlayers int) *Game {
	return &Game{
		ID:             uuid.New(),
		TableID:        tableID,
		Players:        make([]*Player, 0, maxPlayers),
		CommunityCards: make([]Card, 0, 5),
		Pots:           make([]Pot, 0),
		Stage:          PreDeal,
		IsRunning:      false,
		DealerSeat:     0,
		ActionSeat:     0,
		SmallBlind:     smallBlind,
		BigBlind:       bigBlind,
		MinRaise:       bigBlind,
		MaxPlayers:     maxPlayers,
		HandNumber:     0,
		Deck:           NewDeck(),
		Actions:        NewGameActions(),
	}
}

// GetPlayer returns a player by ID
func (g *Game) GetPlayer(playerID uuid.UUID) *Player {
	for _, player := range g.Players {
		if player.ID == playerID {
			return player
		}
	}
	return nil
}

// GetPlayerBySeat returns a player by seat number
func (g *Game) GetPlayerBySeat(seatNumber int) *Player {
	for _, player := range g.Players {
		if player.SeatNumber == seatNumber {
			return player
		}
	}
	return nil
}

// GetActivePlayers returns all active players
func (g *Game) GetActivePlayers() []*Player {
	activePlayers := make([]*Player, 0)
	for _, player := range g.Players {
		if player.IsActive {
			activePlayers = append(activePlayers, player)
		}
	}
	return activePlayers
}

// GetPlayersInHand returns all players still in the current hand
func (g *Game) GetPlayersInHand() []*Player {
	playersInHand := make([]*Player, 0)
	for _, player := range g.Players {
		if player.IsInHand() {
			playersInHand = append(playersInHand, player)
		}
	}
	return playersInHand
}

// GetCurrentBet returns the current highest bet
func (g *Game) GetCurrentBet() int64 {
	var highestBet int64 = 0
	for _, player := range g.Players {
		if player.CurrentBet > highestBet {
			highestBet = player.CurrentBet
		}
	}
	return highestBet
}

// GetTotalPot returns the total amount in all pots
func (g *Game) GetTotalPot() int64 {
	var total int64 = 0
	for _, pot := range g.Pots {
		total += pot.Amount
	}
	return total
}

// CanStart returns true if the game can be started
func (g *Game) CanStart() bool {
	activePlayers := g.GetActivePlayers()
	return len(activePlayers) >= 2 && !g.IsRunning
}

// SortPlayersByPosition sorts players by their seat number
func (g *Game) SortPlayersByPosition() {
	sort.Slice(g.Players, func(i, j int) bool {
		return g.Players[i].SeatNumber < g.Players[j].SeatNumber
	})
}

// GetNextActiveSeat returns the next active seat after the given seat
func (g *Game) GetNextActiveSeat(currentSeat int) int {
	activePlayers := g.GetActivePlayers()
	if len(activePlayers) == 0 {
		return currentSeat
	}

	// Create a sorted list of active seat numbers
	activeSeats := make([]int, 0, len(activePlayers))
	for _, player := range activePlayers {
		activeSeats = append(activeSeats, player.SeatNumber)
	}
	sort.Ints(activeSeats)

	// Find the next active seat
	for _, seat := range activeSeats {
		if seat > currentSeat {
			return seat
		}
	}

	// If no seat found after current, return the first active seat
	return activeSeats[0]
}

// IsActionComplete returns true if all players have acted in the current betting round
func (g *Game) IsActionComplete() bool {
	playersInHand := g.GetPlayersInHand()
	if len(playersInHand) <= 1 {
		return true
	}

	currentBet := g.GetCurrentBet()

	for _, player := range playersInHand {
		if player.CanAct() {
			// Player must have acted and either matched the current bet or be all-in
			if !player.HasActed || (player.CurrentBet < currentBet && !player.IsAllIn) {
				return false
			}
		}
	}

	return true
}

// Reset resets the game state for a new hand
func (g *Game) Reset() {
	g.HandID = nil
	g.CommunityCards = make([]Card, 0, 5)
	g.Pots = make([]Pot, 0)
	g.Stage = PreDeal
	g.ActionSeat = 0
	g.MinRaise = g.BigBlind

	// Reset player state for new hand
	for _, player := range g.Players {
		player.CurrentBet = 0
		player.TotalBet = 0
		player.HoleCards = nil
		player.IsFolded = false
		player.IsAllIn = false
		player.HasActed = false
		player.Position = PlayerPosition{}
	}
}