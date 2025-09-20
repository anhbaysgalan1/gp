package server

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/alexclewontin/riverboat/eval"
	"github.com/anhbaysgalan1/gp/internal/engine"
	"github.com/anhbaysgalan1/gp/internal/models"
	"github.com/anhbaysgalan1/gp/internal/services"
	"github.com/anhbaysgalan1/gp/poker"
	"github.com/google/uuid"
)

// EnginePlayer represents a player in the engine-based game view
type EnginePlayer struct {
	Username   string `json:"username"`
	UUID       string `json:"uuid"`
	Position   uint   `json:"position"`
	SeatID     uint   `json:"seatID"`
	Ready      bool   `json:"ready"`
	In         bool   `json:"in"`
	Called     bool   `json:"called"`
	Left       bool   `json:"left"`
	TotalBuyIn uint   `json:"totalBuyIn"`
	Stack      uint   `json:"stack"`
	Bet        uint   `json:"bet"`
	TotalBet   uint   `json:"totalBet"`
	Cards      []int  `json:"cards"`
}

// EngineGameConfig represents pure engine-based game config
type EngineGameConfig struct {
	MaxBuy     uint `json:"maxBuy"`
	BigBlind   uint `json:"bb"`
	SmallBlind uint `json:"sb"`
}

// EnginePot represents pure engine-based pot
type EnginePot struct {
	Amt                uint   `json:"amount"`
	EligiblePlayerNums []uint `json:"eligiblePlayerNums"`
	WinningPlayerNums  []uint `json:"winningPlayerNums"`
}

// EngineGameView represents a pure engine-based game view
type EngineGameView struct {
	Running        bool             `json:"running"`
	DealerNum      uint             `json:"dealer"`
	ActionNum      uint             `json:"action"`
	UTGNum         uint             `json:"utg"`
	SBNum          uint             `json:"sb"`
	BBNum          uint             `json:"bb"`
	CommunityCards []eval.Card      `json:"communityCards"`
	Stage          int              `json:"stage"`
	Betting        bool             `json:"betting"`
	Config         EngineGameConfig `json:"config"`
	Players        []EnginePlayer   `json:"players"`
	Pots           []EnginePot      `json:"pots"`
	MinRaise       uint             `json:"minRaise"`
	ReadyCount     uint             `json:"readyCount"`
}

// SimpleGameAdapter provides a clean, simple bridge between legacy poker.Game
// and direct database operations, replacing complex event sourcing
type SimpleGameAdapter struct {
	tableService *services.TableService
	tableName    string
	tableRecord  *models.PokerTable
	// Embed a legacy poker game for compatibility during migration
	legacyGame *poker.Game
	// Compatibility fields for events.go direct field access
	engine  engine.PokerEngine // Always nil since we don't use complex engine
	tableID uuid.UUID          // Set to actual table ID from database
	// Map player positions to actual user UUIDs for frontend sync
	playerPositionToUUID map[uint]string
	// Map user UUIDs to their current player positions for reconnection
	userUUIDToPosition map[string]uint
}

// NewSimpleGameAdapter creates a new simplified adapter
func NewSimpleGameAdapter(tableService *services.TableService, tableName string) *SimpleGameAdapter {
	// Create a legacy poker game for backward compatibility
	legacyGame := poker.NewGame()

	return &SimpleGameAdapter{
		tableService:         tableService,
		tableName:            tableName,
		tableRecord:          nil,
		legacyGame:           legacyGame,
		engine:               nil,      // Always nil for simplified approach
		tableID:              uuid.Nil, // Will be set when table is created
		playerPositionToUUID: make(map[uint]string),
		userUUIDToPosition:   make(map[string]uint),
	}
}

// ensureTableExists creates a virtual table for WebSocket-only operations
func (sga *SimpleGameAdapter) ensureTableExists() error {
	if sga.tableRecord != nil {
		return nil // Virtual table already exists
	}

	// For WebSocket-only tables, create a virtual table record without database operations
	// This avoids complex database constraints and foreign key issues
	sga.tableRecord = &models.PokerTable{
		ID:             uuid.New(),
		Name:           sga.tableName,
		TableType:      "cash",
		GameType:       "texas_holdem",
		MaxPlayers:     9,
		MinBuyIn:       100,   // 100 MNT
		MaxBuyIn:       10000, // 10000 MNT
		SmallBlind:     50,    // 50 MNT
		BigBlind:       100,   // 100 MNT
		IsPrivate:      false,
		Status:         "waiting",
		CurrentPlayers: 0,
		CreatedBy:      uuid.New(), // Virtual creator ID
	}
	sga.tableID = sga.tableRecord.ID

	slog.Info("Virtual table created for WebSocket-only operations", "table_id", sga.tableRecord.ID, "table_name", sga.tableName)
	return nil
}

// GetLegacyGame returns the legacy poker game for direct access
func (sga *SimpleGameAdapter) GetLegacyGame() *poker.Game {
	return sga.legacyGame
}

// Legacy interface methods that delegate to embedded poker.Game
func (sga *SimpleGameAdapter) AddPlayer() uint {
	if err := sga.ensureTableExists(); err != nil {
		slog.Error("Failed to ensure table exists in AddPlayer", "error", err)
		return 0
	}

	// Delegate to legacy game for now
	return sga.legacyGame.AddPlayer()
}

func (sga *SimpleGameAdapter) GenerateOmniView() interface{} {
	// Generate view directly from legacy game state
	legacyView := sga.legacyGame.GenerateOmniView()

	// Convert legacy view to engine view format for compatibility
	return sga.convertLegacyToEngineView(legacyView)
}

// getEmptyEngineView returns a consistent empty engine view
func (sga *SimpleGameAdapter) getEmptyEngineView() *EngineGameView {
	return &EngineGameView{
		Running:        false,
		DealerNum:      0,
		ActionNum:      0,
		UTGNum:         0,
		SBNum:          0,
		BBNum:          0,
		CommunityCards: []eval.Card{0, 0, 0, 0, 0},
		Stage:          1,
		Betting:        false,
		Config: EngineGameConfig{
			MaxBuy:     10000,
			BigBlind:   100,
			SmallBlind: 50,
		},
		Players:    []EnginePlayer{}, // Empty players array
		Pots:       []EnginePot{},
		MinRaise:   0,
		ReadyCount: 0,
	}
}

func (sga *SimpleGameAdapter) Start() error {
	if err := sga.ensureTableExists(); err != nil {
		return err
	}

	// Update virtual table status to active
	sga.tableRecord.Status = "active"
	slog.Info("Virtual table status updated to active", "table_id", sga.tableRecord.ID)

	// Start the legacy game
	return sga.legacyGame.Start()
}

func (sga *SimpleGameAdapter) Reset() {
	// Reset legacy game
	sga.legacyGame.Reset()

	// Update virtual table status back to waiting
	if sga.tableRecord != nil {
		sga.tableRecord.Status = "waiting"
		slog.Info("Virtual table status reset to waiting", "table_id", sga.tableRecord.ID)
	}
}

// convertTableToGameView converts the table record to a game view
func (sga *SimpleGameAdapter) convertTableToGameView() *EngineGameView {
	if sga.tableRecord == nil {
		return sga.getEmptyEngineView()
	}

	// Create a game view based on the table record
	gameView := &EngineGameView{
		Running:        sga.tableRecord.Status == "active",
		DealerNum:      0,                          // Will be managed by legacy game
		ActionNum:      0,                          // Will be managed by legacy game
		UTGNum:         0,                          // Will be managed by legacy game
		SBNum:          0,                          // Will be managed by legacy game
		BBNum:          0,                          // Will be managed by legacy game
		CommunityCards: []eval.Card{0, 0, 0, 0, 0}, // Will be managed by legacy game
		Stage:          1,                          // Will be managed by legacy game
		Betting:        sga.tableRecord.Status == "active",
		Config: EngineGameConfig{
			MaxBuy:     uint(sga.tableRecord.MaxBuyIn),
			BigBlind:   uint(sga.tableRecord.BigBlind),
			SmallBlind: uint(sga.tableRecord.SmallBlind),
		},
		Players:    []EnginePlayer{},               // Will be populated by legacy game
		Pots:       []EnginePot{},                  // Will be populated by legacy game
		MinRaise:   uint(sga.tableRecord.BigBlind), // Default to big blind
		ReadyCount: uint(sga.tableRecord.CurrentPlayers),
	}

	slog.Info("Converted table to game view",
		"table_id", sga.tableRecord.ID,
		"table_name", sga.tableRecord.Name,
		"status", sga.tableRecord.Status,
		"running", gameView.Running)

	return gameView
}

// Simple methods for direct database operations (replacing complex engine operations)

// JoinTable processes a player joining through direct legacy game operations
func (sga *SimpleGameAdapter) JoinTable(ctx context.Context, playerID uuid.UUID, username, avatar string) error {
	// For simplified approach, just use legacy game operations
	// No complex database table management needed for WebSocket-only tables
	slog.Info("Player joining table (simplified)", "player_id", playerID, "username", username, "table_name", sga.tableName)

	// Legacy game handles player joining internally
	return nil
}

// SeatPlayer processes seating a player through legacy game operations
func (sga *SimpleGameAdapter) SeatPlayer(ctx context.Context, playerID, sessionID uuid.UUID, username string, seatNumber int, buyInAmount int64) error {
	slog.Info("Seating player (simplified)", "player_id", playerID, "username", username, "seat_number", seatNumber, "buy_in", buyInAmount, "table_name", sga.tableName)

	playerIDStr := playerID.String()

	// Check if this user is already seated (reconnection case)
	if existingPosition, exists := sga.userUUIDToPosition[playerIDStr]; exists {
		slog.Info("Player reconnecting to existing position", "player_id", playerID, "existing_position", existingPosition, "username", username)

		// Update the UUID mapping (in case it changed somehow)
		sga.playerPositionToUUID[existingPosition] = playerIDStr

		// No need to add to legacy game again, just return success
		slog.Info("Player reconnected successfully", "player_id", playerID, "position", existingPosition, "seat_number", seatNumber)
		return nil
	}

	// New player - add to legacy game
	playerPosition := sga.legacyGame.AddPlayer()

	// Set the player's username to their actual username for identification
	err := poker.SetUsername(sga.legacyGame, playerPosition, username)
	if err != nil {
		return fmt.Errorf("failed to set player username: %w", err)
	}

	// Buy in for the specified amount
	err = poker.BuyIn(sga.legacyGame, playerPosition, uint(buyInAmount))
	if err != nil {
		return fmt.Errorf("failed to buy in for player: %w", err)
	}

	// Set the seat ID (position at table)
	err = poker.SetSeatID(sga.legacyGame, playerPosition, uint(seatNumber))
	if err != nil {
		return fmt.Errorf("failed to set seat ID for player: %w", err)
	}

	// Mark player as ready to play
	err = poker.ToggleReady(sga.legacyGame, playerPosition, 0)
	if err != nil {
		return fmt.Errorf("failed to mark player as ready: %w", err)
	}

	// Store the bidirectional mapping between player position and real user UUID
	sga.playerPositionToUUID[playerPosition] = playerIDStr
	sga.userUUIDToPosition[playerIDStr] = playerPosition

	slog.Info("Player seated successfully in legacy game", "player_id", playerID, "position", playerPosition, "seat_number", seatNumber)
	return nil
}

// HandlePlayerAction processes a player action through direct operations
func (sga *SimpleGameAdapter) HandlePlayerAction(ctx context.Context, playerID uuid.UUID, action string, amount int64) error {
	if err := sga.ensureTableExists(); err != nil {
		return err
	}

	slog.Info("Player action", "table_id", sga.tableRecord.ID, "player_id", playerID, "action", action, "amount", amount)

	// This is where you would implement action handling
	// For now, we'll delegate to legacy game logic
	return nil
}

// GetTableID returns the table ID
func (sga *SimpleGameAdapter) GetTableID() *uuid.UUID {
	if sga.tableRecord == nil {
		return nil
	}
	return &sga.tableRecord.ID
}

// GetTableName returns the table name
func (sga *SimpleGameAdapter) GetTableName() string {
	return sga.tableName
}

// convertLegacyToEngineView converts a legacy poker.GameView to EngineGameView format
func (sga *SimpleGameAdapter) convertLegacyToEngineView(legacyView *poker.GameView) *EngineGameView {
	// Convert legacy players to engine players
	enginePlayers := make([]EnginePlayer, len(legacyView.Players))
	for i, legacyPlayer := range legacyView.Players {
		// Convert [2]eval.Card to []int
		cards := []int{int(legacyPlayer.Cards[0]), int(legacyPlayer.Cards[1])}

		// Use the real user UUID from our mapping instead of legacy game UUID
		realUUID := legacyPlayer.UUID // fallback to legacy UUID
		if mappedUUID, exists := sga.playerPositionToUUID[legacyPlayer.Position]; exists {
			realUUID = mappedUUID
		}

		enginePlayers[i] = EnginePlayer{
			Username:   legacyPlayer.Username,
			UUID:       realUUID,
			Position:   legacyPlayer.Position,
			SeatID:     legacyPlayer.SeatID,
			Ready:      legacyPlayer.Ready,
			In:         legacyPlayer.In,
			Called:     legacyPlayer.Called,
			Left:       legacyPlayer.Left,
			TotalBuyIn: legacyPlayer.TotalBuyIn,
			Stack:      legacyPlayer.Stack,
			Bet:        legacyPlayer.Bet,
			TotalBet:   legacyPlayer.TotalBet,
			Cards:      cards,
		}
	}

	// Convert legacy pots to engine pots
	enginePots := make([]EnginePot, len(legacyView.Pots))
	for i, legacyPot := range legacyView.Pots {
		enginePots[i] = EnginePot{
			Amt:                legacyPot.Amt,
			EligiblePlayerNums: legacyPot.EligiblePlayerNums,
			WinningPlayerNums:  legacyPot.WinningPlayerNums,
		}
	}

	// Convert stage - legacy uses GameStage enum, convert to int
	stage := 1 // Default stage
	switch legacyView.Stage {
	case 0: // PreDeal
		stage = 0
	case 1: // PreFlop
		stage = 1
	case 2: // Flop
		stage = 2
	case 3: // Turn
		stage = 3
	case 4: // River
		stage = 4
	}

	return &EngineGameView{
		Running:        legacyView.Running,
		DealerNum:      legacyView.DealerNum,
		ActionNum:      legacyView.ActionNum,
		UTGNum:         legacyView.UTGNum,
		SBNum:          legacyView.SBNum,
		BBNum:          legacyView.BBNum,
		CommunityCards: legacyView.CommunityCards,
		Stage:          stage,
		Betting:        legacyView.Betting,
		Config: EngineGameConfig{
			MaxBuy:     legacyView.Config.MaxBuy,
			BigBlind:   legacyView.Config.BigBlind,
			SmallBlind: legacyView.Config.SmallBlind,
		},
		Players:    enginePlayers,
		Pots:       enginePots,
		MinRaise:   legacyView.MinRaise,
		ReadyCount: legacyView.ReadyCount,
	}
}
