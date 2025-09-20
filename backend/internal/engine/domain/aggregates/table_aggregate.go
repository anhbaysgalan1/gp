package aggregates

import (
	"errors"

	"github.com/anhbaysgalan1/gp/internal/engine/domain/events"
	"github.com/anhbaysgalan1/gp/internal/engine/domain/game"
	"github.com/anhbaysgalan1/gp/internal/engine/domain/table"
	"github.com/google/uuid"
)

var (
	ErrTableNotFound       = errors.New("table not found")
	ErrTableFull           = errors.New("table is full")
	ErrPlayerAlreadySeated = errors.New("player already seated")
	ErrInvalidSeatNumber   = errors.New("invalid seat number")
	ErrSeatOccupied        = errors.New("seat is occupied")
	ErrInvalidBuyInAmount  = errors.New("invalid buy-in amount")
	ErrPlayerNotFound      = errors.New("player not found")
	ErrGameNotRunning      = errors.New("game is not running")
	ErrNotPlayerTurn       = errors.New("not player's turn")
	ErrInvalidAction       = errors.New("invalid action")
	ErrInsufficientChips   = errors.New("insufficient chips")
)

// TableAggregate represents a poker table as an event-sourced aggregate
type TableAggregate struct {
	AggregateRoot
	Table *table.Table `json:"table"`
}

// NewTableAggregate creates a new table aggregate
func NewTableAggregate(tableID uuid.UUID, name string, tableType table.TableType, maxPlayers int, smallBlind, bigBlind int64, config table.TableConfig) *TableAggregate {
	aggregate := &TableAggregate{
		AggregateRoot: AggregateRoot{
			ID:      tableID,
			Version: 0,
			Changes: make([]events.DomainEvent, 0),
		},
		Table: table.NewTable(name, tableType, maxPlayers, smallBlind, bigBlind, config),
	}

	// Apply table created event
	event := events.NewTableCreated(tableID, name, maxPlayers, smallBlind, bigBlind, config.MaxBuyIn, nil)
	aggregate.ApplyChange(event)

	return aggregate
}

// LoadFromHistory rebuilds the aggregate state from events
func (ta *TableAggregate) LoadFromHistory(domainEvents []events.DomainEvent) {
	for _, event := range domainEvents {
		ta.applyEvent(event)
		ta.Version = event.GetVersion()
	}
}

// AddPlayer adds a player to the table
func (ta *TableAggregate) AddPlayer(playerID uuid.UUID, username, avatar string) error {
	if ta.Table.IsFull() {
		return ErrTableFull
	}

	// Check if player is already at the table
	if ta.Table.Game.GetPlayer(playerID) != nil {
		return ErrPlayerAlreadySeated
	}

	event := events.NewPlayerJoined(ta.ID, playerID, username, avatar, ta.Version+1)
	ta.ApplyChange(event)

	return nil
}

// SeatPlayer seats a player at a specific seat with a buy-in
func (ta *TableAggregate) SeatPlayer(playerID, sessionID uuid.UUID, seatNumber int, buyInAmount int64) error {
	// Validate seat number
	if seatNumber < 1 || seatNumber > ta.Table.MaxPlayers {
		return ErrInvalidSeatNumber
	}

	// Check if seat is available
	if ta.Table.Game.GetPlayerBySeat(seatNumber) != nil {
		return ErrSeatOccupied
	}

	// Validate buy-in amount
	if !ta.Table.IsValidBuyIn(buyInAmount) {
		return ErrInvalidBuyInAmount
	}

	// Check if player exists at table
	player := ta.Table.Game.GetPlayer(playerID)
	if player == nil {
		return ErrPlayerNotFound
	}

	// Use game engine to add player to the game
	err := ta.Table.Game.Actions.AddPlayer(ta.Table.Game, playerID.String(), player.Username, seatNumber, buyInAmount)
	if err != nil {
		return err
	}

	event := events.NewPlayerSeated(ta.ID, playerID, sessionID, seatNumber, buyInAmount, ta.Version+1)
	ta.ApplyChange(event)

	return nil
}

// RemovePlayer removes a player from the table
func (ta *TableAggregate) RemovePlayer(playerID uuid.UUID, reason string) error {
	player := ta.Table.Game.GetPlayer(playerID)
	if player == nil {
		return ErrPlayerNotFound
	}

	event := events.NewPlayerLeft(ta.ID, playerID, reason, player.Chips, ta.Version+1)
	ta.ApplyChange(event)

	return nil
}

// StartHand starts a new hand using the game engine
func (ta *TableAggregate) StartHand() error {
	if !ta.Table.Game.CanStart() {
		return ErrGameNotRunning
	}

	// Use the game actions to start the hand
	err := ta.Table.Game.Actions.StartHand(ta.Table.Game)
	if err != nil {
		return err
	}

	handID := *ta.Table.Game.HandID
	activePlayers := ta.Table.Game.GetActivePlayers()
	playerIDs := make([]uuid.UUID, len(activePlayers))
	for i, player := range activePlayers {
		playerIDs[i] = player.ID
	}

	// Find small blind and big blind seats
	smallBlindSeat := 0
	bigBlindSeat := 0
	for _, player := range activePlayers {
		if player.Position.IsSmallBlind {
			smallBlindSeat = player.SeatNumber
		}
		if player.Position.IsBigBlind {
			bigBlindSeat = player.SeatNumber
		}
	}

	event := events.NewHandStarted(
		ta.ID,
		handID,
		ta.Table.Game.DealerSeat,
		smallBlindSeat,
		bigBlindSeat,
		playerIDs,
		ta.Table.SmallBlind,
		ta.Table.BigBlind,
		ta.Version+1,
	)

	ta.ApplyChange(event)
	return nil
}

// PlayerAction handles a player action using the game engine
func (ta *TableAggregate) PlayerAction(playerID uuid.UUID, action string, amount int64) error {
	if !ta.Table.Game.IsRunning {
		return ErrGameNotRunning
	}

	player := ta.Table.Game.GetPlayer(playerID)
	if player == nil {
		return ErrPlayerNotFound
	}

	if !player.CanAct() {
		return ErrInvalidAction
	}

	// Store player state before action for event data
	playerIDStr := playerID.String()
	handID := ta.Table.Game.HandID

	var err error
	var event events.DomainEvent

	switch action {
	case "check":
		err = ta.Table.Game.Actions.PlayerCheck(ta.Table.Game, playerIDStr)
		if err == nil {
			event = events.NewPlayerCheck(ta.ID, *handID, playerID, ta.Version+1)
		}
	case "fold":
		err = ta.Table.Game.Actions.PlayerFold(ta.Table.Game, playerIDStr)
		if err == nil {
			event = events.NewPlayerFold(ta.ID, *handID, playerID, ta.Version+1)
		}
	case "call", "bet", "raise":
		// For betting actions, use the game engine to handle the bet
		err = ta.Table.Game.Actions.PlayerBet(ta.Table.Game, playerIDStr, amount)
		if err == nil {
			newTotalBet := player.TotalBet
			newChips := player.Chips
			isAllIn := player.IsAllIn

			switch action {
			case "call":
				event = events.NewPlayerCall(ta.ID, *handID, playerID, amount, newTotalBet, newChips, isAllIn, ta.Version+1)
			case "bet":
				event = events.NewPlayerBet(ta.ID, *handID, playerID, amount, newTotalBet, newChips, isAllIn, ta.Version+1)
			case "raise":
				event = events.NewPlayerRaise(ta.ID, *handID, playerID, amount, newTotalBet, newChips, isAllIn, ta.Version+1)
			}
		}
	default:
		return ErrInvalidAction
	}

	if err != nil {
		return err
	}

	ta.ApplyChange(event)
	return nil
}

// applyEvent applies an event to the aggregate state
func (ta *TableAggregate) applyEvent(event events.DomainEvent) {
	switch e := event.(type) {
	case *events.TableCreated:
		ta.applyTableCreated(e)
	case *events.PlayerJoined:
		ta.applyPlayerJoined(e)
	case *events.PlayerSeated:
		ta.applyPlayerSeated(e)
	case *events.PlayerLeft:
		ta.applyPlayerLeft(e)
	case *events.HandStarted:
		ta.applyHandStarted(e)
	case *events.PlayerAction:
		ta.applyPlayerAction(e)
	case *events.HandEnded:
		ta.applyHandEnded(e)
	}
}

func (ta *TableAggregate) applyTableCreated(event *events.TableCreated) {
	// Table is already created in constructor
}

func (ta *TableAggregate) applyPlayerJoined(event *events.PlayerJoined) {
	player := &game.Player{
		ID:         event.PlayerID,
		Username:   event.Username,
		SeatNumber: 0,        // Will be set when seated
		Chips:      0,        // Will be set when seated
		IsActive:   false,    // Will be set when seated
		SessionID:  uuid.Nil, // Will be set when seated
	}
	ta.Table.Game.Players = append(ta.Table.Game.Players, player)
}

func (ta *TableAggregate) applyPlayerSeated(event *events.PlayerSeated) {
	player := ta.Table.Game.GetPlayer(event.PlayerID)
	if player != nil {
		player.SeatNumber = event.SeatNumber
		player.Chips = event.BuyInAmount
		player.SessionID = event.SessionID
		player.IsActive = true
	}
}

func (ta *TableAggregate) applyPlayerLeft(event *events.PlayerLeft) {
	// Remove player from game
	for i, player := range ta.Table.Game.Players {
		if player.ID == event.PlayerID {
			ta.Table.Game.Players = append(ta.Table.Game.Players[:i], ta.Table.Game.Players[i+1:]...)
			break
		}
	}
}

func (ta *TableAggregate) applyHandStarted(event *events.HandStarted) {
	ta.Table.Game.HandID = &event.HandID
	ta.Table.Game.IsRunning = true
	ta.Table.Game.HandNumber++
	ta.Table.Game.Stage = game.PreFlop
	ta.Table.UpdateStatus(table.TableStatusActive)
}

func (ta *TableAggregate) applyPlayerAction(event *events.PlayerAction) {
	player := ta.Table.Game.GetPlayer(event.PlayerID)
	if player != nil {
		switch event.Action {
		case "fold":
			player.IsFolded = true
		case "call", "bet", "raise":
			player.CurrentBet += event.Amount
			player.TotalBet += event.Amount
			player.Chips -= event.Amount
			if event.IsAllIn {
				player.IsAllIn = true
			}
		}
		player.HasActed = true
	}
}

func (ta *TableAggregate) applyHandEnded(event *events.HandEnded) {
	ta.Table.Game.Reset()
	ta.Table.Game.IsRunning = false
}
