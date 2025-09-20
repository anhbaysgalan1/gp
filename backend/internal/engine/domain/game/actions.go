package game

import (
	"errors"
	"sort"

	"github.com/google/uuid"
)

var (
	ErrIllegalAction        = errors.New("illegal action")
	ErrInvalidPosition      = errors.New("invalid position")
	ErrInsufficientFunds    = errors.New("insufficient funds")
	ErrPlayerNotInHand      = errors.New("player not in hand")
	ErrNotPlayerTurn        = errors.New("not player's turn")
	ErrInvalidBuyIn         = errors.New("invalid buy-in")
	ErrGameNotRunning       = errors.New("game not running")
	ErrCannotStartGame      = errors.New("cannot start game")
)

// GameActions provides methods for all poker game actions
type GameActions struct{}

// NewGameActions creates a new GameActions instance
func NewGameActions() *GameActions {
	return &GameActions{}
}

// DealCards deals cards to players and community
func (ga *GameActions) DealCards(g *Game) error {
	if g.Stage == PreDeal {
		return ga.dealHoleCards(g)
	} else if g.Stage == PreFlop {
		return ga.dealFlop(g)
	} else if g.Stage == Flop {
		return ga.dealTurn(g)
	} else if g.Stage == Turn {
		return ga.dealRiver(g)
	}
	return ErrIllegalAction
}

// dealHoleCards deals 2 cards to each active player
func (ga *GameActions) dealHoleCards(g *Game) error {
	if g.Deck == nil {
		g.Deck = NewDeck()
	}

	// Shuffle deck multiple times for randomness
	for i := 0; i < 3; i++ {
		g.Deck.Shuffle()
	}

	// Deal 2 cards to each active player
	for _, player := range g.Players {
		if player.IsActive {
			player.HoleCards = []Card{
				g.Deck.Deal(),
				g.Deck.Deal(),
			}
		}
	}

	g.Stage = PreFlop
	return nil
}

// dealFlop deals the flop (3 community cards)
func (ga *GameActions) dealFlop(g *Game) error {
	// Burn one card
	g.Deck.Deal()

	// Deal 3 cards to community
	for i := 0; i < 3; i++ {
		g.CommunityCards = append(g.CommunityCards, g.Deck.Deal())
	}

	g.Stage = Flop
	ga.resetBettingRound(g)
	return nil
}

// dealTurn deals the turn card
func (ga *GameActions) dealTurn(g *Game) error {
	// Burn one card
	g.Deck.Deal()

	// Deal turn card
	g.CommunityCards = append(g.CommunityCards, g.Deck.Deal())

	g.Stage = Turn
	ga.resetBettingRound(g)
	return nil
}

// dealRiver deals the river card
func (ga *GameActions) dealRiver(g *Game) error {
	// Burn one card
	g.Deck.Deal()

	// Deal river card
	g.CommunityCards = append(g.CommunityCards, g.Deck.Deal())

	g.Stage = River
	ga.resetBettingRound(g)
	return nil
}

// PlayerBet handles player betting action
func (ga *GameActions) PlayerBet(g *Game, playerID string, amount int64) error {
	player := ga.findPlayerByStringID(g, playerID)
	if player == nil {
		return ErrPlayerNotInHand
	}

	if !player.CanAct() {
		return ErrIllegalAction
	}

	// Validate amount
	currentBet := g.GetCurrentBet()
	callAmount := currentBet - player.CurrentBet

	if amount < callAmount {
		return ErrIllegalAction
	}

	if amount > player.Chips {
		// All-in
		amount = player.Chips
		player.IsAllIn = true
	}

	// Make the bet
	player.CurrentBet += amount
	player.TotalBet += amount
	player.Chips -= amount
	player.HasActed = true

	// Update min raise if this is a raise
	if amount > callAmount {
		g.MinRaise = amount - callAmount
		ga.resetActedFlags(g, playerID)
	}

	return nil
}

// PlayerCheck handles player check action
func (ga *GameActions) PlayerCheck(g *Game, playerID string) error {
	player := ga.findPlayerByStringID(g, playerID)
	if player == nil {
		return ErrPlayerNotInHand
	}

	if !player.CanAct() {
		return ErrIllegalAction
	}

	// Can only check if no bet to call
	if g.GetCurrentBet() > player.CurrentBet {
		return ErrIllegalAction
	}

	player.HasActed = true
	return nil
}

// PlayerFold handles player fold action
func (ga *GameActions) PlayerFold(g *Game, playerID string) error {
	player := ga.findPlayerByStringID(g, playerID)
	if player == nil {
		return ErrPlayerNotInHand
	}

	if !player.CanAct() {
		return ErrIllegalAction
	}

	player.IsFolded = true
	player.HasActed = true

	return nil
}

// AddPlayer adds a new player to the game
func (ga *GameActions) AddPlayer(g *Game, playerID, username string, seatNumber int, chips int64) error {
	// Check if seat is available
	for _, player := range g.Players {
		if player.SeatNumber == seatNumber {
			return ErrInvalidPosition
		}
	}

	// Check max players
	if len(g.Players) >= g.MaxPlayers {
		return ErrIllegalAction
	}

	// Parse UUID from string
	playerUUID, err := uuid.Parse(playerID)
	if err != nil {
		return ErrIllegalAction
	}

	// Create new player
	newPlayer := &Player{
		ID:           playerUUID,
		Username:     username,
		SeatNumber:   seatNumber,
		Chips:        chips,
		IsActive:     true,
		IsFolded:     false,
		IsAllIn:      false,
		HasActed:     false,
		HoleCards:    nil,
		CurrentBet:   0,
		TotalBet:     0,
		Position:     PlayerPosition{},
	}

	g.Players = append(g.Players, newPlayer)

	// Sort players by seat number
	sort.Slice(g.Players, func(i, j int) bool {
		return g.Players[i].SeatNumber < g.Players[j].SeatNumber
	})

	return nil
}

// RemovePlayer removes a player from the game
func (ga *GameActions) RemovePlayer(g *Game, playerID string) error {
	playerUUID, err := uuid.Parse(playerID)
	if err != nil {
		return ErrIllegalAction
	}

	for i, player := range g.Players {
		if player.ID == playerUUID {
			// If player is in active hand, fold them first
			if player.IsInHand() {
				ga.PlayerFold(g, playerID)
			}

			// Remove from players slice
			g.Players = append(g.Players[:i], g.Players[i+1:]...)
			return nil
		}
	}
	return ErrPlayerNotInHand
}

// StartHand initializes a new hand
func (ga *GameActions) StartHand(g *Game) error {
	if !g.CanStart() {
		return ErrCannotStartGame
	}

	// Reset hand state
	g.Reset()
	g.IsRunning = true
	g.HandNumber++

	// Create new deck if needed
	if g.Deck == nil {
		g.Deck = NewDeck()
	}

	// Set positions
	ga.setPositions(g)

	// Post blinds
	ga.postBlinds(g)

	// Deal hole cards
	err := ga.dealHoleCards(g)
	if err != nil {
		return err
	}

	return nil
}

// EndHand ends the current hand and distributes pots
func (ga *GameActions) EndHand(g *Game) error {
	// Calculate pots and winners
	ga.calculatePots(g)
	ga.evaluateWinners(g)

	// Reset for next hand
	g.Stage = PreDeal
	g.IsRunning = false

	return nil
}

// Helper methods

func (ga *GameActions) findPlayerByStringID(g *Game, playerID string) *Player {
	playerUUID, err := uuid.Parse(playerID)
	if err != nil {
		return nil
	}

	for _, player := range g.Players {
		if player.ID == playerUUID {
			return player
		}
	}
	return nil
}

func (ga *GameActions) resetBettingRound(g *Game) {
	// Reset all player betting flags for new round
	for _, player := range g.Players {
		if player.IsInHand() && !player.IsAllIn {
			player.HasActed = false
		}
		player.CurrentBet = 0
	}

	g.MinRaise = g.BigBlind
	g.ActionSeat = ga.getNextActionSeat(g, g.DealerSeat)
}

func (ga *GameActions) resetActedFlags(g *Game, raisingPlayerID string) {
	// Reset acted flags for all players except the raiser
	raisingUUID, err := uuid.Parse(raisingPlayerID)
	if err != nil {
		return
	}

	for _, player := range g.Players {
		if player.ID != raisingUUID && player.IsInHand() && !player.IsAllIn {
			player.HasActed = false
		}
	}
}

func (ga *GameActions) setPositions(g *Game) {
	activePlayers := g.GetActivePlayers()
	if len(activePlayers) < 2 {
		return
	}

	// Reset all positions
	for _, player := range g.Players {
		player.Position = PlayerPosition{}
	}

	// Set dealer
	if g.DealerSeat < len(activePlayers) {
		activePlayers[g.DealerSeat].Position.IsDealer = true
	}

	// Set blinds
	if len(activePlayers) == 2 {
		// Heads up: dealer is small blind
		activePlayers[g.DealerSeat].Position.IsSmallBlind = true
		activePlayers[(g.DealerSeat+1)%len(activePlayers)].Position.IsBigBlind = true
	} else {
		// Regular: small blind is left of dealer, big blind is left of small blind
		activePlayers[(g.DealerSeat+1)%len(activePlayers)].Position.IsSmallBlind = true
		activePlayers[(g.DealerSeat+2)%len(activePlayers)].Position.IsBigBlind = true
		activePlayers[(g.DealerSeat+3)%len(activePlayers)].Position.IsUTG = true
	}
}

func (ga *GameActions) postBlinds(g *Game) {
	for _, player := range g.Players {
		if player.Position.IsSmallBlind {
			blindAmount := g.SmallBlind
			if blindAmount > player.Chips {
				blindAmount = player.Chips
				player.IsAllIn = true
			}
			player.CurrentBet = blindAmount
			player.TotalBet = blindAmount
			player.Chips -= blindAmount
		} else if player.Position.IsBigBlind {
			blindAmount := g.BigBlind
			if blindAmount > player.Chips {
				blindAmount = player.Chips
				player.IsAllIn = true
			}
			player.CurrentBet = blindAmount
			player.TotalBet = blindAmount
			player.Chips -= blindAmount
		}
	}
}

func (ga *GameActions) getNextActionSeat(g *Game, currentSeat int) int {
	activePlayers := g.GetActivePlayers()
	if len(activePlayers) == 0 {
		return currentSeat
	}

	// Find next active player who can act
	for i := 1; i <= len(activePlayers); i++ {
		nextSeat := (currentSeat + i) % len(activePlayers)
		if nextSeat < len(g.Players) && g.Players[nextSeat].CanAct() {
			return nextSeat
		}
	}

	return currentSeat
}

func (ga *GameActions) calculatePots(g *Game) {
	// Implementation of pot calculation with side pots
	// This is complex logic that handles all-in situations
	g.Pots = []Pot{} // Reset pots

	// Get all players with bets
	playersWithBets := []*Player{}
	for _, player := range g.Players {
		if player.TotalBet > 0 {
			playersWithBets = append(playersWithBets, player)
		}
	}

	if len(playersWithBets) == 0 {
		return
	}

	// Sort players by total bet amount
	sort.Slice(playersWithBets, func(i, j int) bool {
		return playersWithBets[i].TotalBet < playersWithBets[j].TotalBet
	})

	// Create pots for each betting level
	prevBetLevel := int64(0)
	for i, player := range playersWithBets {
		betLevel := player.TotalBet
		if betLevel > prevBetLevel {
			// Create pot for this betting level
			potAmount := (betLevel - prevBetLevel) * int64(len(playersWithBets)-i)

			eligiblePlayers := []uuid.UUID{}
			for j := i; j < len(playersWithBets); j++ {
				if playersWithBets[j].IsInHand() {
					eligiblePlayers = append(eligiblePlayers, playersWithBets[j].ID)
				}
			}

			if potAmount > 0 && len(eligiblePlayers) > 0 {
				pot := Pot{
					ID:              uuid.New(),
					Amount:          potAmount,
					EligiblePlayers: eligiblePlayers,
					IsSidePot:       len(g.Pots) > 0,
				}
				g.Pots = append(g.Pots, pot)
			}

			prevBetLevel = betLevel
		}
	}
}

func (ga *GameActions) evaluateWinners(g *Game) {
	if len(g.CommunityCards) != 5 {
		return // Can't evaluate without all community cards
	}

	for i := range g.Pots {
		pot := &g.Pots[i]
		bestScore := int(^uint(0) >> 1) // Max int
		winners := []uuid.UUID{}

		// Evaluate each eligible player's hand
		for _, playerID := range pot.EligiblePlayers {
			player := g.GetPlayer(playerID)
			if player != nil && len(player.HoleCards) == 2 {
				_, score, handRank := EvaluateHand(player.HoleCards, g.CommunityCards)

				if score < bestScore {
					bestScore = score
					winners = []uuid.UUID{playerID}
					pot.WinningHand = player.HoleCards
					pot.HandRank = handRank
				} else if score == bestScore {
					winners = append(winners, playerID)
				}
			}
		}

		pot.WinningPlayers = winners
	}
}

