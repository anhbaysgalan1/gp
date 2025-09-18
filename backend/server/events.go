package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/evanofslack/go-poker/internal/models"
	"github.com/evanofslack/go-poker/poker"
	"github.com/google/uuid"
)

const gameAdminName string = "system"

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
		// Create new session - note: for WebSocket-only tables, we won't create DB session
		// We'll just track the session in the WebSocket client
		sessionID = uuid.New()
		c.sessionID = sessionID

		// Transfer funds from main account to game account
		transactionID, err = c.formanceService.TransferToGame(ctx, c.userID, buyInAmount, sessionID)
		if err != nil {
			slog.Default().Warn("Failed to transfer funds to game", "user_id", c.userID, "amount", buyInAmount, "error", err)
			safeSend(c, createErrorMessage("Failed to transfer funds for buy-in. Please try again."))
			return
		}

		// For WebSocket tables, we track session in the client and don't require database session
		slog.Info("User taking seat with new session (WebSocket-only)",
			"user_id", c.userID,
			"session_id", sessionID,
			"seat_id", seatID,
			"buy_in", buyIn)
	}

	// Now proceed with the original poker game logic
	position := c.table.game.AddPlayer()
	c.uuid = c.table.game.GenerateOmniView().Players[position].UUID
	safeSend(c, createUpdatedPlayerUUID(c))

	err = poker.SetUsername(c.table.game, position, username)
	if err != nil {
		slog.Default().Warn("Set username", "error", err)
		// If poker game setup fails, we should reverse the fund transfer
		// TODO: Implement reversal logic
	}

	err = poker.BuyIn(c.table.game, position, buyIn)
	if err != nil {
		slog.Default().Warn("Buy in", "error", err)
		// If poker game setup fails, we should reverse the fund transfer
		// TODO: Implement reversal logic
	}

	// set player ready
	// TODO make this a separate action
	err = poker.ToggleReady(c.table.game, position, 0)
	if err != nil {
		slog.Default().Warn("Toggle ready", "error", err)
	}

	err = poker.SetSeatID(c.table.game, position, seatID)
	if err != nil {
		slog.Default().Warn("Set seat id", "error", err)
	}

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

	// Broadcast updated game state
	c.table.broadcast <- createUpdatedGame(c)
}

func handleStartGame(c *Client) {
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

	view := c.table.game.GenerateOmniView()
	err := poker.Deal(c.table.game, view.DealerNum, 0)
	if err != nil {
		slog.Default().Warn("Deal table", "error", err)
	}
	c.table.broadcast <- createUpdatedGame(c)
}

func handleCall(c *Client) {
	view := c.table.game.GenerateOmniView()
	pn := view.ActionNum
	currentPlayer := view.Players[pn]

	// compute amount needed to call
	maxBet := view.Players[0].TotalBet
	for _, p := range view.Players {
		if p.TotalBet > maxBet {
			maxBet = p.TotalBet
		}
	}
	callAmount := maxBet - currentPlayer.TotalBet

	// if player must go all in to call
	if callAmount >= currentPlayer.Stack {
		callAmount = currentPlayer.Stack
	}

	err := poker.Bet(c.table.game, pn, callAmount)
	if err != nil {
		slog.Default().Warn("Handle call", "error", err)
	}

	// Check if hand ended and handle pot distribution
	handlePotDistribution(c)

	c.table.broadcast <- createUpdatedGame(c)
}

func handleRaise(c *Client, raise uint) {
	view := c.table.game.GenerateOmniView()
	pn := view.ActionNum
	err := poker.Bet(c.table.game, pn, raise)
	if err != nil {
		slog.Default().Warn("Handle raise", "error", err)
	}

	// Check if hand ended and handle pot distribution
	handlePotDistribution(c)

	c.table.broadcast <- createUpdatedGame(c)
}

func handleCheck(c *Client) {
	view := c.table.game.GenerateOmniView()
	pn := view.ActionNum
	err := poker.Bet(c.table.game, pn, 0)
	if err != nil {
		slog.Default().Warn("Handle check", "error", err)
	}

	// Check if hand ended and handle pot distribution
	handlePotDistribution(c)

	c.table.broadcast <- createUpdatedGame(c)
}

func handleFold(c *Client) {
	view := c.table.game.GenerateOmniView()
	pn := view.ActionNum
	err := poker.Fold(c.table.game, pn, 0)
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

	game := updateGame{
		base{actionUpdateGame},
		c.table.game.GenerateOmniView(),
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
		view := c.table.game.GenerateOmniView()
		for _, player := range view.Players {
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
	view := table.game.GenerateOmniView()

	startMsg := "starting new hand"
	table.broadcast <- createNewLog(startMsg)

	sbUser := view.Players[view.SBNum].Username
	sb := view.Config.SmallBlind
	sbMsg := fmt.Sprintf("%s is small blind (%d)", sbUser, sb)
	table.broadcast <- createNewLog(sbMsg)

	bbUser := view.Players[view.BBNum].Username
	bb := view.Config.BigBlind
	bbMsg := fmt.Sprintf("%s is big blind (%d)", bbUser, bb)
	table.broadcast <- createNewLog(bbMsg)
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
	if c.formanceService == nil {
		return // Skip if no Formance service available
	}

	view := c.table.game.GenerateOmniView()

	// Check if game has ended (stage 1 indicates showdown/end)
	if view.Stage != 1 || len(view.Pots) == 0 {
		return // Hand not finished yet
	}

	ctx := context.Background()

	// Process each pot (there can be multiple pots in case of side pots)
	for _, pot := range view.Pots {
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
					for _, player := range view.Players {
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

			// Use the winner's session ID if available, otherwise create a new one
			sessionID := winnerClient.sessionID
			if sessionID == uuid.Nil {
				sessionID = uuid.New()
				slog.Default().Warn("No session ID stored for winner client, generating new one for pot distribution", "user_id", winnerUserID)
			}

			// Transfer winnings from game account back to main wallet
			transactionID, err := c.formanceService.TransferFromGame(ctx, winnerUserID, winningsPerPlayer, sessionID)
			if err != nil {
				slog.Default().Error("Failed to transfer pot winnings to winner",
					"winner_user_id", winnerUserID,
					"amount", winningsPerPlayer,
					"pot_total", potAmount,
					"error", err)
				// Send error message to winner
				winnerClient.send <- createErrorMessage("Failed to transfer winnings. Please contact support.")
				continue
			}

			// Log successful pot distribution
			slog.Info("Pot winnings distributed to winner",
				"winner_user_id", winnerUserID,
				"amount", winningsPerPlayer,
				"pot_total", potAmount,
				"transaction_id", transactionID,
				"session_id", sessionID)

			// Send success message to winner
			winnerClient.send <- createSuccessMessage(fmt.Sprintf("You won %d MNT! Transaction ID: %s", winningsPerPlayer, transactionID))

			// Send real-time balance update to winner
			sendBalanceUpdateToClient(winnerClient, "win", winningsPerPlayer, transactionID)

			// Broadcast winning message to table
			winnerPlayer := view.Players[winnerPosition]
			c.table.broadcast <- createNewLog(fmt.Sprintf("%s wins %d MNT from the pot", winnerPlayer.Username, winningsPerPlayer))
		}
	}
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
