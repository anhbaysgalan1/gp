package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/anhbaysgalan1/gp/internal/models"
	"github.com/anhbaysgalan1/gp/poker"
	"github.com/google/uuid"
)

const gameAdminName string = "system"

// getEngineView safely casts interface{} to *EngineGameView
func getEngineView(viewInterface interface{}) (*EngineGameView, bool) {
	engineView, ok := viewInterface.(*EngineGameView)
	return engineView, ok
}

// safeSend safely sends a message to a client's send channel without panicking on closed channels
func safeSend(c *Client, message []byte) {
	defer func() {
		if r := recover(); r != nil {
			slog.Default().Warn("Attempted to send message to closed channel", "user_id", c.userID)
		}
	}()

	select {
	case c.send <- message:
		// Message sent successfully
	default:
		// Channel is full or closed, skip sending
		slog.Default().Warn("Unable to send message to client, channel unavailable", "user_id", c.userID)
	}
}

func handleJoinTable(c *Client, tablename string) {
	table := c.hub.findTableByName(tablename)
	if table == nil {
		table = c.hub.createTable(tablename)
	}
	c.table = table
	table.register <- c
}

func handleLeaveTable(c *Client, tablename string) {
	table := c.hub.findTableByName(tablename)

	// Handle cash-out before leaving table
	handlePlayerCashOut(c)

	// Clear session ID since player is leaving table
	c.sessionID = uuid.Nil

	// Clear player UUID to ensure clean state
	c.uuid = ""

	slog.Info("Player left table", "user_id", c.userID, "table", tablename)

	table.unregister <- c
}

func handleSendMessage(c *Client, username string, message string) {
	c.table.broadcast <- createNewMessage(username, message)
}

func handleSendLog(c *Client, message string) {
	c.table.broadcast <- createNewLog(message)
}

func handleNewPlayer(c *Client, username string) {
	c.username = username
	safeSend(c, createUpdatedGame(c))
	c.table.broadcast <- createNewMessage(gameAdminName, fmt.Sprintf("%s has joined", username))
}

func handleTakeSeat(c *Client, username string, seatID uint, buyIn uint) {
	slog.Default().Info("Processing take seat request", "user_id", c.userID, "username", username, "seat_id", seatID, "buy_in", buyIn)
	// Check if client is authenticated
	if c.userID == uuid.Nil || c.formanceService == nil {
		safeSend(c, createErrorMessage("Authentication required for seat actions"))
		return
	}

	// Validate buy-in amount
	if buyIn <= 0 {
		safeSend(c, createErrorMessage("Buy-in amount must be positive"))
		return
	}

	buyInAmount := int64(buyIn)

	// Check user balance
	ctx := context.Background()
	balance, err := c.formanceService.GetUserBalance(ctx, c.userID, c.db)
	if err != nil {
		slog.Default().Warn("Failed to get user balance", "user_id", c.userID, "error", err)
		safeSend(c, createErrorMessage("Failed to check balance. Please try again."))
		return
	}

	// Check if user has sufficient main balance
	if balance.MainBalance < buyInAmount {
		safeSend(c, createErrorMessage(fmt.Sprintf("Insufficient balance for buy-in. You have %d MNT but need %d MNT. Please deposit more funds in your wallet.", balance.MainBalance, buyInAmount)))
		return
	}

	// Add balance warnings for low balance situations
	remainingBalance := balance.MainBalance - buyInAmount

	// Define minimum amounts for warnings (based on table blinds)
	minBuyIn := int64(100)            // Minimum buy-in amount from virtual table config
	criticalThreshold := minBuyIn * 2 // 200 MNT - enough for 2 more buy-ins
	warningThreshold := minBuyIn * 5  // 500 MNT - enough for 5 more buy-ins

	if remainingBalance <= 0 {
		// This shouldn't happen due to earlier check, but safety net
		safeSend(c, createWarningMessage("Warning: This buy-in will use all your remaining balance."))
	} else if remainingBalance < criticalThreshold {
		safeSend(c, createWarningMessage(fmt.Sprintf("Critical: Low balance warning! After this buy-in, you'll have only %d MNT remaining. Consider depositing more funds.", remainingBalance)))
	} else if remainingBalance < warningThreshold {
		safeSend(c, createWarningMessage(fmt.Sprintf("Warning: After this buy-in, you'll have %d MNT remaining. You may want to deposit more funds soon.", remainingBalance)))
	}

	// Check if user already has an active session
	var existingSession models.GameSession
	err = c.db.Where("user_id = ? AND status = 'active'", c.userID).First(&existingSession).Error

	var sessionID uuid.UUID
	var transactionID string

	if err == nil {
		// User has existing session, use that
		sessionID = existingSession.ID
		c.sessionID = sessionID

		// Update the seat number in the existing session
		seatNumberInt := int(seatID)
		existingSession.SeatNumber = &seatNumberInt
		if err := c.db.Save(&existingSession).Error; err != nil {
			slog.Default().Warn("Failed to update seat number in session", "user_id", c.userID, "seat_id", seatID, "error", err)
			safeSend(c, createErrorMessage("Failed to assign seat. Please try again."))
			return
		}

		slog.Info("User taking seat with existing session",
			"user_id", c.userID,
			"session_id", sessionID,
			"seat_id", seatID)
	} else {
		// Create new database session for real money games
		sessionID = uuid.New()
		c.sessionID = sessionID

		// Transfer funds from main account to game account
		transactionID, err = c.formanceService.TransferToGame(ctx, c.userID, buyInAmount, sessionID)
		if err != nil {
			slog.Default().Warn("Failed to transfer funds to game", "user_id", c.userID, "amount", buyInAmount, "error", err)
			safeSend(c, createErrorMessage("Failed to transfer funds for buy-in. Please try again."))
			return
		}

		// Create database session record for real money game
		if c.table.sessionService != nil {
			// Get or create a virtual table ID for the session
			var tableID uuid.UUID
			if c.table.game.GetTableID() != nil {
				tableID = *c.table.game.GetTableID()
			} else {
				// Use a default table ID for virtual tables
				tableID = uuid.New()
			}

			seatNumberInt := int(seatID)
			_, err = c.table.sessionService.CreateSession(ctx, c.userID, tableID, buyInAmount, seatNumberInt)
			if err != nil {
				slog.Default().Warn("Failed to create database session", "user_id", c.userID, "error", err)
				// Don't fail the buy-in, just log the warning - WebSocket session still works
			} else {
				slog.Info("Database session created successfully", "user_id", c.userID, "session_id", sessionID, "table_id", tableID)
			}
		}

		slog.Info("User taking seat with new session",
			"user_id", c.userID,
			"session_id", sessionID,
			"seat_id", seatID,
			"buy_in", buyIn)
	}

	// Use simplified approach - directly seat player in legacy game

	// Join table through simplified adapter
	err = c.table.game.JoinTable(ctx, c.userID, username, "")
	if err != nil {
		slog.Default().Warn("Join table failed", "error", err)
		// Clear session on failure
		c.sessionID = uuid.Nil
		safeSend(c, createErrorMessage("Failed to join table. Please try again."))
		return
	}

	// Seat player through simplified adapter
	err = c.table.game.SeatPlayer(ctx, c.userID, sessionID, c.username, int(seatID), buyInAmount)
	if err != nil {
		slog.Default().Warn("Seat player failed", "error", err)
		// Clear session on failure
		c.sessionID = uuid.Nil
		safeSend(c, createErrorMessage("Failed to take seat. Please try again."))
		return
	}

	// Store player UUID for frontend sync
	c.uuid = c.userID.String()

	// Seating succeeded, broadcast updated state
	slog.Info("Seating successful", "user_id", c.userID, "seat_id", seatID)

	// Log successful buy-in
	slog.Info("Player successfully bought in",
		"user_id", c.userID,
		"username", username,
		"seat_id", seatID,
		"buy_in", buyIn,
		"transaction_id", transactionID,
		"session_id", sessionID)

	// Send success message to client
	safeSend(c, createSuccessMessage(fmt.Sprintf("Successfully bought in for %d MNT. Transaction ID: %s", buyIn, transactionID)))

	// Send real-time balance update
	sendBalanceUpdateToClient(c, "buy_in", -buyInAmount, transactionID)

	// Send player UUID update to sync frontend
	safeSend(c, createUpdatedPlayerUUID(c))

	// Broadcast updated game state
	c.table.broadcast <- createUpdatedGame(c)
}

func handleStartGame(c *Client) {
	// Try engine-based approach first
	if c.table.game.engine != nil {
		ctx := context.Background()
		err := c.table.game.engine.StartHand(ctx, c.table.game.tableID)
		if err != nil {
			slog.Default().Warn("Engine start hand failed, falling back to legacy", "error", err)
		} else {
			// Engine succeeded, broadcast updated state
			broadcastDeal(c.table)
			c.table.broadcast <- createUpdatedGame(c)
			return
		}
	}

	// Legacy approach as fallback
	err := c.table.game.Start()
	if err != nil {
		fmt.Println(err)
	}
	broadcastDeal(c.table)
	c.table.broadcast <- createUpdatedGame(c)
}

func handleResetGame(c *Client) {
	c.table.game.Reset()
	c.table.broadcast <- createUpdatedGame(c)
}

func handleDealGame(c *Client) {
	broadcastDeal(c.table)

	viewInterface := c.table.game.GenerateOmniView()
	engineView, ok := getEngineView(viewInterface)
	if !ok {
		slog.Default().Error("Failed to cast view to EngineGameView in handleDealGame")
		return
	}

	err := poker.Deal(c.table.game.GetLegacyGame(), engineView.DealerNum, 0)
	if err != nil {
		slog.Default().Warn("Deal table", "error", err)
	}
	c.table.broadcast <- createUpdatedGame(c)
}

func handleCall(c *Client) {
	// Try engine-based approach first if client is authenticated
	if c.userID != uuid.Nil && c.table.game.engine != nil {
		ctx := context.Background()
		err := c.table.game.HandlePlayerAction(ctx, c.userID, "call", 0)
		if err != nil {
			slog.Default().Warn("Engine call action failed, falling back to legacy", "error", err)
		} else {
			// Engine succeeded, broadcast updated state
			c.table.broadcast <- createUpdatedGame(c)
			return
		}
	}

	// Legacy approach as fallback
	viewInterface := c.table.game.GenerateOmniView()
	engineView, ok := getEngineView(viewInterface)
	if !ok {
		slog.Default().Error("Failed to cast view to EngineGameView in handleCall")
		return
	}

	pn := engineView.ActionNum
	currentPlayer := engineView.Players[pn]

	// compute amount needed to call
	maxBet := engineView.Players[0].TotalBet
	for _, p := range engineView.Players {
		if p.TotalBet > maxBet {
			maxBet = p.TotalBet
		}
	}
	callAmount := maxBet - currentPlayer.TotalBet

	// if player must go all in to call
	if callAmount >= currentPlayer.Stack {
		callAmount = currentPlayer.Stack
	}

	err := poker.Bet(c.table.game.GetLegacyGame(), pn, callAmount)
	if err != nil {
		slog.Default().Warn("Handle call", "error", err)
	}

	// Check if hand ended and handle pot distribution
	handlePotDistribution(c)

	c.table.broadcast <- createUpdatedGame(c)
}

func handleRaise(c *Client, raise uint) {
	// Try engine-based approach first if client is authenticated
	if c.userID != uuid.Nil && c.table.game.engine != nil {
		ctx := context.Background()
		err := c.table.game.HandlePlayerAction(ctx, c.userID, "raise", int64(raise))
		if err != nil {
			slog.Default().Warn("Engine raise action failed, falling back to legacy", "error", err)
		} else {
			// Engine succeeded, broadcast updated state
			c.table.broadcast <- createUpdatedGame(c)
			return
		}
	}

	// Legacy approach as fallback
	viewInterface := c.table.game.GenerateOmniView()
	engineView, ok := getEngineView(viewInterface)
	if !ok {
		slog.Default().Error("Failed to cast view to EngineGameView in handleRaise")
		return
	}

	pn := engineView.ActionNum
	err := poker.Bet(c.table.game.GetLegacyGame(), pn, raise)
	if err != nil {
		slog.Default().Warn("Handle raise", "error", err)
	}

	// Check if hand ended and handle pot distribution
	handlePotDistribution(c)

	c.table.broadcast <- createUpdatedGame(c)
}

func handleCheck(c *Client) {
	// Try engine-based approach first if client is authenticated
	if c.userID != uuid.Nil && c.table.game.engine != nil {
		ctx := context.Background()
		err := c.table.game.HandlePlayerAction(ctx, c.userID, "check", 0)
		if err != nil {
			slog.Default().Warn("Engine check action failed, falling back to legacy", "error", err)
		} else {
			// Engine succeeded, broadcast updated state
			c.table.broadcast <- createUpdatedGame(c)
			return
		}
	}

	// Legacy approach as fallback
	viewInterface := c.table.game.GenerateOmniView()
	engineView, ok := getEngineView(viewInterface)
	if !ok {
		slog.Default().Error("Failed to cast view to EngineGameView in handleCheck")
		return
	}

	pn := engineView.ActionNum
	err := poker.Bet(c.table.game.GetLegacyGame(), pn, 0)
	if err != nil {
		slog.Default().Warn("Handle check", "error", err)
	}

	// Check if hand ended and handle pot distribution
	handlePotDistribution(c)

	c.table.broadcast <- createUpdatedGame(c)
}

func handleFold(c *Client) {
	// Try engine-based approach first if client is authenticated
	if c.userID != uuid.Nil && c.table.game.engine != nil {
		ctx := context.Background()
		err := c.table.game.HandlePlayerAction(ctx, c.userID, "fold", 0)
		if err != nil {
			slog.Default().Warn("Engine fold action failed, falling back to legacy", "error", err)
		} else {
			// Engine succeeded, broadcast updated state
			c.table.broadcast <- createUpdatedGame(c)
			return
		}
	}

	// Legacy approach as fallback
	viewInterface := c.table.game.GenerateOmniView()
	engineView, ok := getEngineView(viewInterface)
	if !ok {
		slog.Default().Error("Failed to cast view to EngineGameView in handleFold")
		return
	}

	pn := engineView.ActionNum
	err := poker.Fold(c.table.game.GetLegacyGame(), pn, 0)
	if err != nil {
		slog.Default().Warn("Handle fold", "error", err)
		return
	}

	// Check if hand ended and handle pot distribution
	handlePotDistribution(c)

	c.table.broadcast <- createUpdatedGame(c)
}

func handleGetBalance(c *Client) {
	if c.formanceService == nil || c.userID == uuid.Nil {
		safeSend(c, createErrorMessage("Authentication required for balance information"))
		return
	}

	// Send current balance update with no transaction details
	sendBalanceUpdateToClient(c, "balance_check", 0, "")
}

func createNewMessage(username string, message string) []byte {
	new := newMessage{
		base{actionNewMessage},
		uuid.New().String(),
		message,
		username,
		currentTime(),
	}
	resp, err := json.Marshal(new)
	if err != nil {
		slog.Default().Warn("Marshal new message", "error", err)
	}
	return resp
}

func createNewLog(message string) []byte {
	log := newLog{
		base{actionNewLog},
		uuid.New().String(),
		message,
		currentTime(),
	}
	resp, err := json.Marshal(log)
	if err != nil {
		slog.Default().Warn("Marshal new log", "error", err)
	}
	return resp
}

func createUpdatedGame(c *Client) []byte {
	// Get session info for the current client
	var sessionInfo *SessionInfo
	if c.userID != uuid.Nil {
		sessionInfo = getClientSessionInfo(c)
	}

	gameState := c.table.game.GenerateOmniView()

	// Debug logging to see what WebSocket sends
	if engineView, ok := getEngineView(gameState); ok {
		slog.Info("Broadcasting game state",
			"players_count", len(engineView.Players),
			"user_id", c.userID,
			"has_session", sessionInfo != nil && sessionInfo.HasSession,
		)
		// Log first player details if any exist
		if len(engineView.Players) > 0 {
			slog.Info("First player details",
				"username", engineView.Players[0].Username,
				"seat_id", engineView.Players[0].SeatID,
				"uuid", engineView.Players[0].UUID,
			)
		}
	} else {
		slog.Warn("Could not cast game state to EngineGameView for debugging")
	}

	game := updateGame{
		base{actionUpdateGame},
		gameState,
		sessionInfo,
	}

	resp, err := json.Marshal(game)
	if err != nil {
		slog.Default().Warn("Marshal update game", "error", err)
	}
	return resp
}

// getClientSessionInfo retrieves session information for a specific client
func getClientSessionInfo(c *Client) *SessionInfo {
	if c.userID == uuid.Nil {
		return nil
	}

	// Check if client has a WebSocket session ID (primary method)
	hasWebSocketSession := c.sessionID != uuid.Nil
	isSeated := false
	var seatNumber *int

	// Check if user is seated (has a player in the poker game)
	if c.table != nil && c.table.game != nil {
		viewInterface := c.table.game.GenerateOmniView()
		engineView, ok := getEngineView(viewInterface)
		if ok {
			for _, player := range engineView.Players {
				if c.uuid == player.UUID {
					isSeated = true
					// Extract seat number from the poker game player
					// The seat ID is stored in the poker game player data
					if player.SeatID > 0 {
						seatNumberInt := int(player.SeatID)
						seatNumber = &seatNumberInt
					}
					break
				}
			}
		}
	}

	// Note: Removed overly aggressive session clearing logic that was causing
	// sessions to be cleared immediately after successful seating due to timing issues

	// Try to get session info from database as fallback
	var dbSession models.GameSession
	dbSessionExists := c.db.Where("user_id = ? AND status = 'active'", c.userID).
		Order("created_at DESC").
		First(&dbSession).Error == nil

	// Use WebSocket session as primary, DB session as fallback
	hasSession := hasWebSocketSession || dbSessionExists

	sessionInfo := &SessionInfo{
		UserID:     c.userID.String(),
		HasSession: hasSession,
		IsSeated:   isSeated,
		SeatNumber: seatNumber,
	}

	// Set session ID (prefer WebSocket session ID)
	if hasWebSocketSession {
		sessionInfo.SessionID = c.sessionID.String()
	} else if dbSessionExists {
		sessionInfo.SessionID = dbSession.ID.String()
	}

	return sessionInfo
}

func createUpdatedPlayerUUID(c *Client) []byte {
	uuid := updatePlayerUUID{
		base{actionUpdatePlayerUUID},
		c.uuid,
	}
	resp, err := json.Marshal(uuid)
	if err != nil {
		slog.Default().Warn("Marshal player uuid", "error", err)
	}
	return resp
}

func broadcastDeal(table *table) {
	viewInterface := table.game.GenerateOmniView()
	engineView, ok := getEngineView(viewInterface)
	if !ok {
		slog.Default().Error("Failed to cast view to EngineGameView in broadcastDeal")
		return
	}

	startMsg := "starting new hand"
	table.broadcast <- createNewLog(startMsg)

	if len(engineView.Players) > int(engineView.SBNum) {
		sbUser := engineView.Players[engineView.SBNum].Username
		sb := engineView.Config.SmallBlind
		sbMsg := fmt.Sprintf("%s is small blind (%d)", sbUser, sb)
		table.broadcast <- createNewLog(sbMsg)
	}

	if len(engineView.Players) > int(engineView.BBNum) {
		bbUser := engineView.Players[engineView.BBNum].Username
		bb := engineView.Config.BigBlind
		bbMsg := fmt.Sprintf("%s is big blind (%d)", bbUser, bb)
		table.broadcast <- createNewLog(bbMsg)
	}
}

func currentTime() string {
	return fmt.Sprintf("%d:%02d", time.Now().Hour(), time.Now().Minute())
}

func createErrorMessage(message string) []byte {
	errorMsg := map[string]interface{}{
		"action":  "error",
		"message": message,
		"time":    currentTime(),
	}
	resp, err := json.Marshal(errorMsg)
	if err != nil {
		slog.Default().Warn("Marshal error message", "error", err)
	}
	return resp
}

func createWarningMessage(message string) []byte {
	warningMsg := map[string]interface{}{
		"action":  "warning",
		"message": message,
		"time":    currentTime(),
	}
	resp, err := json.Marshal(warningMsg)
	if err != nil {
		slog.Default().Warn("Marshal warning message", "error", err)
	}
	return resp
}

func createSuccessMessage(message string) []byte {
	successMsg := map[string]interface{}{
		"action":  "success",
		"message": message,
		"time":    currentTime(),
	}
	resp, err := json.Marshal(successMsg)
	if err != nil {
		slog.Default().Warn("Marshal success message", "error", err)
	}
	return resp
}

func createBalanceUpdate(mainBalance, gameBalance int64, currency, transactionID, changeType string, changeAmount int64) []byte {
	balanceUpdate := updateBalance{
		base{actionUpdateBalance},
		mainBalance,
		gameBalance,
		currency,
		transactionID,
		changeAmount,
		changeType,
		currentTime(),
	}
	resp, err := json.Marshal(balanceUpdate)
	if err != nil {
		slog.Default().Warn("Marshal balance update", "error", err)
	}
	return resp
}

// sendBalanceUpdateToClient sends a real-time balance update to a specific client
func sendBalanceUpdateToClient(c *Client, changeType string, changeAmount int64, transactionID string) {
	if c.formanceService == nil || c.userID == uuid.Nil {
		return // Skip if no Formance service or not authenticated
	}

	ctx := context.Background()
	balance, err := c.formanceService.GetUserBalance(ctx, c.userID, c.db)
	if err != nil {
		slog.Default().Warn("Failed to get balance for update notification", "user_id", c.userID, "error", err)
		return
	}

	// Send balance update to this specific client
	balanceUpdate := createBalanceUpdate(
		balance.MainBalance,
		balance.GameBalance,
		"MNT", // TODO: Get from config
		transactionID,
		changeType,
		changeAmount,
	)
	safeSend(c, balanceUpdate)

	slog.Info("Sent balance update to client",
		"user_id", c.userID,
		"change_type", changeType,
		"change_amount", changeAmount,
		"main_balance", balance.MainBalance,
		"game_balance", balance.GameBalance)
}

// broadcastBalanceUpdateToUser sends balance update to all clients for a specific user
func broadcastBalanceUpdateToUser(hub *Hub, userID uuid.UUID, changeType string, changeAmount int64, transactionID string) {
	// Find all clients for this user across all tables
	for table := range hub.tables {
		for client := range table.clients {
			if client.userID == userID {
				sendBalanceUpdateToClient(client, changeType, changeAmount, transactionID)
			}
		}
	}
}

// handlePotDistribution checks if a hand has ended and distributes winnings via Formance
func handlePotDistribution(c *Client) {
	viewInterface := c.table.game.GenerateOmniView()
	engineView, ok := getEngineView(viewInterface)
	if !ok {
		slog.Default().Error("Failed to cast view to EngineGameView in handlePotDistribution")
		return
	}

	// Check if game has ended (stage 1 indicates showdown/end)
	if engineView.Stage != 1 || len(engineView.Pots) == 0 {
		return // Hand not finished yet
	}

	ctx := context.Background()

	// Determine if this is a practice game (no Formance service or issues with real money transfers)
	isPracticeGame := c.formanceService == nil

	// Process each pot (there can be multiple pots in case of side pots)
	for _, pot := range engineView.Pots {
		if len(pot.WinningPlayerNums) == 0 {
			continue // No winners for this pot
		}

		potAmount := int64(pot.Amt)
		winnerCount := len(pot.WinningPlayerNums)
		winningsPerPlayer := potAmount / int64(winnerCount)

		// Distribute winnings to each winner
		for _, winnerPosition := range pot.WinningPlayerNums {
			// Find the winner player and their user ID
			var winnerClient *Client
			var winnerUserID uuid.UUID

			// Find the client for this winner position
			for client := range c.table.clients {
				if client.userID != uuid.Nil {
					// Check if this client has a player at the winning position
					for _, player := range engineView.Players {
						if player.Position == winnerPosition && client.uuid == player.UUID {
							winnerClient = client
							winnerUserID = client.userID
							break
						}
					}
				}
				if winnerClient != nil {
					break
				}
			}

			if winnerClient == nil || winnerUserID == uuid.Nil {
				slog.Default().Warn("Could not find winner client for pot distribution",
					"winner_position", winnerPosition, "pot_amount", potAmount)
				continue
			}

			var transactionID string
			var shouldSendBalanceUpdate bool

			if !isPracticeGame {
				// Try real money transfer
				sessionID := winnerClient.sessionID
				if sessionID == uuid.Nil {
					sessionID = uuid.New()
					slog.Default().Warn("No session ID stored for winner client, generating new one for pot distribution", "user_id", winnerUserID)
				}

				var err error
				transactionID, err = c.formanceService.TransferFromGame(ctx, winnerUserID, winningsPerPlayer, sessionID)
				if err != nil {
					slog.Default().Error("Failed to transfer pot winnings to winner",
						"winner_user_id", winnerUserID,
						"amount", winningsPerPlayer,
						"pot_total", potAmount,
						"error", err)
					// Fallback to practice mode for this winner
					isPracticeGame = true
					transactionID = ""
					winnerClient.send <- createErrorMessage("Failed to transfer winnings. Game continuing in practice mode.")
				} else {
					shouldSendBalanceUpdate = true
					slog.Info("Real money pot distribution completed",
						"winner_user_id", winnerUserID,
						"amount", winningsPerPlayer,
						"pot_total", potAmount,
						"session_id", sessionID,
						"transaction_id", transactionID)
				}
			}

			if isPracticeGame {
				// Practice table - no real money transfer, just continue game
				slog.Info("Practice table pot distribution (no real money transfer)",
					"winner_user_id", winnerUserID,
					"amount", winningsPerPlayer,
					"pot_total", potAmount)
			}

			// Log successful pot distribution
			slog.Info("Pot winnings distributed to winner",
				"winner_user_id", winnerUserID,
				"amount", winningsPerPlayer,
				"pot_total", potAmount,
				"transaction_id", transactionID,
				"is_practice", isPracticeGame)

			// Send success message to winner
			if transactionID != "" && shouldSendBalanceUpdate {
				winnerClient.send <- createSuccessMessage(fmt.Sprintf("You won %d MNT! Transaction ID: %s", winningsPerPlayer, transactionID))
				// Send real-time balance update to winner
				sendBalanceUpdateToClient(winnerClient, "win", winningsPerPlayer, transactionID)
			} else {
				// For practice tables or when no transaction occurred
				winnerClient.send <- createSuccessMessage(fmt.Sprintf("You won %d chips!", winningsPerPlayer))
			}

			// Broadcast winning message to table
			winnerPlayer := engineView.Players[winnerPosition]
			if transactionID != "" {
				c.table.broadcast <- createNewLog(fmt.Sprintf("%s wins %d MNT from the pot", winnerPlayer.Username, winningsPerPlayer))
			} else {
				c.table.broadcast <- createNewLog(fmt.Sprintf("%s wins %d chips from the pot", winnerPlayer.Username, winningsPerPlayer))
			}
		}
	}

	// End the current hand by setting running = false and resetting for next hand
	// This ensures the game state is properly reset before auto-start
	if c.table.game != nil {
		legacyGame := c.table.game.GetLegacyGame()
		if legacyGame != nil {
			// End hand and reset for next hand (sets running = false)
			legacyGame.EndHandAndReset()
			slog.Info("Hand ended, game state reset", "table", c.table.name)
		}
	}

	// Always attempt auto-start after pot distribution processing is complete
	// This ensures the game continues even if there were payment failures
	slog.Info("Pot distribution completed, scheduling auto-start", "table", c.table.name, "is_practice", isPracticeGame)
	scheduleAutoHandStart(c.table)
}

// handlePlayerCashOut transfers any remaining funds from player's game session back to main wallet
func handlePlayerCashOut(c *Client) {
	if c.formanceService == nil || c.userID == uuid.Nil {
		return // Skip if no Formance service or not authenticated
	}

	// Check if player has any active game balance to cash out
	// For now, we'll implement a simple approach where we check the user's current game balance
	// and transfer it all back to main wallet

	ctx := context.Background()

	// Get current user balance to check game balance
	balance, err := c.formanceService.GetUserBalance(ctx, c.userID, c.db)
	if err != nil {
		slog.Default().Warn("Failed to get balance for cash out", "user_id", c.userID, "error", err)
		return
	}

	// If there's game balance, transfer it back to main wallet
	if balance.GameBalance > 0 {
		// Use the stored session ID if available, otherwise create a new one
		sessionID := c.sessionID
		if sessionID == uuid.Nil {
			sessionID = uuid.New()
			slog.Default().Warn("No session ID stored for client, generating new one for cash-out", "user_id", c.userID)
		}

		transactionID, err := c.formanceService.TransferFromGame(ctx, c.userID, balance.GameBalance, sessionID)
		if err != nil {
			slog.Default().Error("Failed to cash out game balance",
				"user_id", c.userID,
				"amount", balance.GameBalance,
				"error", err)
			safeSend(c, createErrorMessage("Failed to cash out remaining balance. Please contact support."))
			return
		}

		// Log successful cash-out
		slog.Info("Player cashed out successfully",
			"user_id", c.userID,
			"amount", balance.GameBalance,
			"transaction_id", transactionID,
			"session_id", sessionID)

		// Send success message to player
		if balance.GameBalance > 0 {
			safeSend(c, createSuccessMessage(fmt.Sprintf("Cashed out %d MNT to your wallet. Transaction ID: %s", balance.GameBalance, transactionID)))

			// Send real-time balance update
			sendBalanceUpdateToClient(c, "cash_out", balance.GameBalance, transactionID)
		}
	}
}

// scheduleAutoHandStart schedules automatic next hand start after a delay
func scheduleAutoHandStart(table *table) {
	go func() {
		// Wait 3 seconds to allow players to see the hand results
		time.Sleep(3 * time.Second)

		// Check if we should auto-start the next hand
		if shouldAutoStartNextHand(table) {
			slog.Info("Auto-starting next hand", "table", table.name)

			// Broadcast notification that next hand is starting
			table.broadcast <- createNewLog("Next hand starting automatically...")

			// Wait 1 more second for the message to be seen
			time.Sleep(1 * time.Second)

			// Trigger start game logic - need a dummy client for the existing handler
			autoStartNextHand(table)
		} else {
			slog.Info("Not auto-starting next hand - insufficient players or game conditions not met", "table", table.name)
		}
	}()
}

// shouldAutoStartNextHand checks if conditions are met for automatic hand start
func shouldAutoStartNextHand(table *table) bool {
	if table.game == nil {
		slog.Info("Auto-start validation failed: no game", "table", table.name)
		return false
	}

	// Get current game view
	gameView := table.game.GenerateOmniView()
	engineView, ok := getEngineView(gameView)
	if !ok {
		slog.Info("Auto-start validation failed: cannot get engine view", "table", table.name)
		return false
	}

	// Count active players with chips and busted players (for practice mode)
	activePlayersWithChips := 0
	bustedPlayers := 0
	totalConnectedPlayers := 0

	for i, player := range engineView.Players {
		slog.Info("Checking player for auto-start",
			"table", table.name,
			"player_index", i,
			"username", player.Username,
			"ready", player.Ready,
			"stack", player.Stack,
			"in", player.In)

		// Count all players who are connected (regardless of chips)
		if player.Username != "" { // Non-empty username means connected player
			totalConnectedPlayers++
		}

		if player.Ready && player.Stack > 0 {
			activePlayersWithChips++
		} else if player.Username != "" && player.Stack == 0 {
			// Player is connected but has no chips (busted in practice mode)
			bustedPlayers++
		}
	}

	slog.Info("Auto-start validation",
		"table", table.name,
		"active_players_with_chips", activePlayersWithChips,
		"busted_players", bustedPlayers,
		"total_connected_players", totalConnectedPlayers,
		"running", engineView.Running,
		"betting", engineView.Betting,
		"stage", engineView.Stage)

	// Log the situation for debugging
	if totalConnectedPlayers >= 2 && activePlayersWithChips < 2 {
		slog.Info("Insufficient active players for auto-start",
			"table", table.name,
			"total_connected", totalConnectedPlayers,
			"active_with_chips", activePlayersWithChips,
			"busted", bustedPlayers)
	}

	// Need at least 2 players with chips to continue
	if activePlayersWithChips < 2 {
		slog.Info("Auto-start validation failed: insufficient players", "table", table.name, "count", activePlayersWithChips)
		return false
	}

	// Check game is in correct state (not already running)
	if engineView.Running {
		slog.Info("Auto-start validation failed: game already running", "table", table.name)
		return false
	}

	// Game should be in PreDeal stage and not betting
	if engineView.Betting {
		slog.Info("Auto-start validation failed: game is betting", "table", table.name)
		return false
	}

	slog.Info("Auto-start validation passed", "table", table.name)
	return true
}

// autoStartNextHand triggers the start game logic automatically
func autoStartNextHand(table *table) {
	// Create a dummy client context to trigger the start game handler
	// We need to find any active client at this table to use as context
	for client := range table.clients {
		if client != nil && client.table == table {
			// Use this client's context to trigger start game
			handleStartGame(client)
			return
		}
	}

	// If no clients found, try legacy game start directly
	if table.game != nil {
		legacyGame := table.game.GetLegacyGame()
		if legacyGame != nil {
			err := legacyGame.Start()
			if err != nil {
				slog.Warn("Auto-start failed with legacy game", "error", err, "table", table.name)
			} else {
				// Broadcast game state update
				table.broadcast <- createUpdatedGame(nil)
				slog.Info("Auto-started next hand successfully", "table", table.name)
			}
		}
	}
}
