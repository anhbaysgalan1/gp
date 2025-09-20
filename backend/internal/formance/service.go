package formance

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/anhbaysgalan1/gp/internal/config"
	"github.com/anhbaysgalan1/gp/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Service struct {
	client   *Client // Improved client with better filtering
	currency string
}

func NewService(cfg *config.Config) *Service {
	return &Service{
		client:   NewClient(cfg),
		currency: cfg.FormanceCurrency,
	}
}

// Initialize creates the ledger and system accounts
func (s *Service) Initialize(ctx context.Context) error {
	// Create ledger using legacy client for now
	if err := s.client.CreateLedger(ctx); err != nil {
		return fmt.Errorf("failed to create ledger: %w", err)
	}

	// Create initial funding transaction to establish accounts
	postings := []PostingSimple{
		{
			Source:      WorldAccount,
			Destination: SystemHouseAccount,
			Amount:      0, // Zero amount just to create accounts
			Asset:       s.currency,
		},
	}

	_, err := s.client.CreateTransaction(ctx, postings, map[string]string{
		"type": "system_initialization",
	})
	if err != nil {
		return fmt.Errorf("failed to initialize system accounts: %w", err)
	}

	slog.Info("Formance service initialized")
	return nil
}

// GetUserBalance gets main balance and total game balance across all active sessions
func (s *Service) GetUserBalance(ctx context.Context, userID uuid.UUID, db *gorm.DB) (*models.UserBalance, error) {
	mainAccount := PlayerWalletAccount(userID)

	mainBalance, err := s.client.GetBalance(ctx, mainAccount)
	if err != nil {
		// If balance fetch fails, try to create the wallet first
		slog.Warn("Failed to get balance, attempting to create wallet", "user_id", userID, "error", err)

		if createErr := s.CreateUserWallet(ctx, userID); createErr != nil {
			return nil, fmt.Errorf("failed to get balance and failed to create wallet: %w, original error: %w", createErr, err)
		}

		// Retry balance fetch after wallet creation
		mainBalance, err = s.client.GetBalance(ctx, mainAccount)
		if err != nil {
			return nil, fmt.Errorf("failed to get main balance after wallet creation: %w", err)
		}

		slog.Info("Wallet created and balance retrieved successfully", "user_id", userID, "balance", mainBalance)
	}

	// Sum balances from all active game sessions
	totalGameBalance, err := s.GetTotalSessionBalances(ctx, userID, db)
	if err != nil {
		return nil, fmt.Errorf("failed to get session balances: %w", err)
	}

	return &models.UserBalance{
		MainBalance:  mainBalance,
		GameBalance:  totalGameBalance,
		TotalBalance: mainBalance + totalGameBalance,
	}, nil
}

// GetSessionBalance gets the balance for a specific game session
func (s *Service) GetSessionBalance(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID) (int64, error) {
	sessionAccount := SessionAccount(userID, sessionID)
	return s.client.GetBalance(ctx, sessionAccount)
}

// GetTotalSessionBalances sums balances from all active game sessions for a user
func (s *Service) GetTotalSessionBalances(ctx context.Context, userID uuid.UUID, db *gorm.DB) (int64, error) {
	// Query active sessions for the user
	var activeSessions []models.GameSession

	err := db.Where("user_id = ? AND status = ?", userID, models.GameSessionStatusActive).Find(&activeSessions).Error
	if err != nil {
		return 0, fmt.Errorf("failed to query active sessions: %w", err)
	}

	totalBalance := int64(0)

	// For each active session, get the balance from Formance
	for _, session := range activeSessions {
		sessionBalance, err := s.GetSessionBalance(ctx, userID, session.ID)
		if err != nil {
			// Log error but continue with other sessions
			slog.Warn("Failed to get session balance", "user_id", userID, "session_id", session.ID, "error", err)
			continue
		}
		totalBalance += sessionBalance
	}

	slog.Debug("Calculated total session balances", "user_id", userID, "total_balance", totalBalance, "active_sessions", len(activeSessions))
	return totalBalance, nil
}

// TransferToGame transfers MNT from user main account to session-specific account
func (s *Service) TransferToGame(ctx context.Context, userID uuid.UUID, amount int64, sessionID uuid.UUID) (string, error) {
	if amount <= 0 {
		return "", fmt.Errorf("amount must be positive")
	}

	mainAccount := PlayerWalletAccount(userID)
	sessionAccount := SessionAccount(userID, sessionID)

	postings := []PostingSimple{
		{
			Source:      mainAccount,
			Destination: sessionAccount,
			Amount:      amount,
			Asset:       s.currency,
		},
	}

	metadata := map[string]string{
		"type":       "game_buyin",
		"user_id":    userID.String(),
		"session_id": sessionID.String(),
	}

	transactionID, err := s.client.CreateTransaction(ctx, postings, metadata)
	if err != nil {
		return "", fmt.Errorf("failed to transfer to game: %w", err)
	}

	slog.Info("Transferred to game account", "user_id", userID, "amount", amount, "transaction_id", transactionID)
	return transactionID, nil
}

// TransferFromGame transfers MNT from user session account back to main account
func (s *Service) TransferFromGame(ctx context.Context, userID uuid.UUID, amount int64, sessionID uuid.UUID) (string, error) {
	if amount <= 0 {
		return "", fmt.Errorf("amount must be positive")
	}

	mainAccount := PlayerWalletAccount(userID)
	sessionAccount := SessionAccount(userID, sessionID)

	postings := []PostingSimple{
		{
			Source:      sessionAccount,
			Destination: mainAccount,
			Amount:      amount,
			Asset:       s.currency,
		},
	}

	metadata := map[string]string{
		"type":       "game_cashout",
		"user_id":    userID.String(),
		"session_id": sessionID.String(),
	}

	transactionID, err := s.client.CreateTransaction(ctx, postings, metadata)
	if err != nil {
		return "", fmt.Errorf("failed to transfer from game: %w", err)
	}

	slog.Info("Transferred from game account", "user_id", userID, "amount", amount, "transaction_id", transactionID)
	return transactionID, nil
}

// ProcessTournamentBuyIn transfers MNT from user main account to tournament pool
func (s *Service) ProcessTournamentBuyIn(ctx context.Context, userID uuid.UUID, tournamentID uuid.UUID, buyIn int64) (string, error) {
	if buyIn <= 0 {
		return "", fmt.Errorf("buy-in amount must be positive")
	}

	userAccount := PlayerWalletAccount(userID)
	tournamentAccount := TournamentPoolAccount(tournamentID)

	postings := []PostingSimple{
		{
			Source:      userAccount,
			Destination: tournamentAccount,
			Amount:      buyIn,
			Asset:       s.currency,
		},
	}

	metadata := map[string]string{
		"type":          "tournament_buyin",
		"user_id":       userID.String(),
		"tournament_id": tournamentID.String(),
	}

	transactionID, err := s.client.CreateTransaction(ctx, postings, metadata)
	if err != nil {
		return "", fmt.Errorf("failed to process tournament buy-in: %w", err)
	}

	slog.Info("Processed tournament buy-in", "user_id", userID, "tournament_id", tournamentID, "amount", buyIn, "transaction_id", transactionID)
	return transactionID, nil
}

// DistributeTournamentPrize transfers prize money from tournament pool to user
func (s *Service) DistributeTournamentPrize(ctx context.Context, userID uuid.UUID, tournamentID uuid.UUID, prize int64) (string, error) {
	if prize <= 0 {
		return "", fmt.Errorf("prize amount must be positive")
	}

	userAccount := PlayerWalletAccount(userID)
	tournamentAccount := TournamentPoolAccount(tournamentID)

	postings := []PostingSimple{
		{
			Source:      tournamentAccount,
			Destination: userAccount,
			Amount:      prize,
			Asset:       s.currency,
		},
	}

	metadata := map[string]string{
		"type":          "tournament_prize",
		"user_id":       userID.String(),
		"tournament_id": tournamentID.String(),
	}

	transactionID, err := s.client.CreateTransaction(ctx, postings, metadata)
	if err != nil {
		return "", fmt.Errorf("failed to distribute tournament prize: %w", err)
	}

	slog.Info("Distributed tournament prize", "user_id", userID, "tournament_id", tournamentID, "amount", prize, "transaction_id", transactionID)
	return transactionID, nil
}

// RakeStrategy defines different rake collection methods
type RakeStrategy string

const (
	RakeStrategyPerHand    RakeStrategy = "per_hand"   // Percentage of pot
	RakeStrategyTimeBased  RakeStrategy = "time_based" // Fixed amount per time period
	RakeStrategyTournament RakeStrategy = "tournament" // Built into buy-in
)

// RakeConfig holds configuration for rake collection
type RakeConfig struct {
	Strategy   RakeStrategy
	Percentage float64 // For per-hand rake (e.g., 0.05 for 5%)
	MaxRake    int64   // Maximum rake per hand
	MinPot     int64   // Minimum pot size to collect rake
	TimeAmount int64   // Fixed amount for time-based rake
	TableID    uuid.UUID
	HandID     string
}

// CollectRake transfers rake to house account using specified strategy
func (s *Service) CollectRake(ctx context.Context, config RakeConfig, playerSessions map[uuid.UUID]uuid.UUID) (string, error) {
	if len(playerSessions) == 0 {
		return "", nil // No players, no rake to collect
	}

	switch config.Strategy {
	case RakeStrategyPerHand:
		return s.collectPerHandRake(ctx, config, playerSessions)
	case RakeStrategyTimeBased:
		return s.collectTimeBasedRake(ctx, config, playerSessions)
	case RakeStrategyTournament:
		return s.collectTournamentRake(ctx, config, playerSessions)
	default:
		return "", fmt.Errorf("unsupported rake strategy: %s", config.Strategy)
	}
}

// collectPerHandRake collects percentage-based rake from pot
func (s *Service) collectPerHandRake(ctx context.Context, config RakeConfig, playerSessions map[uuid.UUID]uuid.UUID) (string, error) {
	potAmount := int64(0)

	// Calculate total pot from all player sessions
	for playerID, sessionID := range playerSessions {
		balance, err := s.GetSessionBalance(ctx, playerID, sessionID)
		if err != nil {
			slog.Warn("Failed to get session balance for rake calculation", "player_id", playerID, "session_id", sessionID)
			continue
		}
		potAmount += balance
	}

	if potAmount < config.MinPot {
		return "", nil // Pot too small for rake
	}

	rakeAmount := int64(float64(potAmount) * config.Percentage)
	if rakeAmount > config.MaxRake {
		rakeAmount = config.MaxRake
	}

	if rakeAmount <= 0 {
		return "", nil
	}

	// Distribute rake collection among players proportionally
	rakePerPlayer := rakeAmount / int64(len(playerSessions))
	if rakePerPlayer <= 0 {
		return "", nil
	}

	var postings []PostingSimple
	for playerID, sessionID := range playerSessions {
		sessionAccount := SessionAccount(playerID, sessionID)
		postings = append(postings, PostingSimple{
			Source:      sessionAccount,
			Destination: "revenue:rake",
			Amount:      rakePerPlayer,
			Asset:       s.currency,
		})
	}

	metadata := map[string]string{
		"type":       "rake_collection",
		"strategy":   string(RakeStrategyPerHand),
		"table_id":   config.TableID.String(),
		"hand_id":    config.HandID,
		"pot_amount": fmt.Sprintf("%d", potAmount),
		"rake_rate":  fmt.Sprintf("%.2f", config.Percentage),
		"players":    fmt.Sprintf("%d", len(playerSessions)),
	}

	transactionID, err := s.client.CreateTransaction(ctx, postings, metadata)
	if err != nil {
		return "", fmt.Errorf("failed to collect per-hand rake: %w", err)
	}

	slog.Info("Collected per-hand rake",
		"table_id", config.TableID,
		"hand_id", config.HandID,
		"pot_amount", potAmount,
		"rake_amount", rakeAmount,
		"players", len(playerSessions),
		"transaction_id", transactionID)

	return transactionID, nil
}

// collectTimeBasedRake collects fixed rake amount per time period
func (s *Service) collectTimeBasedRake(ctx context.Context, config RakeConfig, playerSessions map[uuid.UUID]uuid.UUID) (string, error) {
	if config.TimeAmount <= 0 {
		return "", fmt.Errorf("time-based rake amount must be positive")
	}

	rakePerPlayer := config.TimeAmount / int64(len(playerSessions))
	if rakePerPlayer <= 0 {
		return "", nil
	}

	var postings []PostingSimple
	for playerID, sessionID := range playerSessions {
		sessionAccount := SessionAccount(playerID, sessionID)
		postings = append(postings, PostingSimple{
			Source:      sessionAccount,
			Destination: "revenue:rake",
			Amount:      rakePerPlayer,
			Asset:       s.currency,
		})
	}

	metadata := map[string]string{
		"type":     "rake_collection",
		"strategy": string(RakeStrategyTimeBased),
		"table_id": config.TableID.String(),
		"amount":   fmt.Sprintf("%d", config.TimeAmount),
		"players":  fmt.Sprintf("%d", len(playerSessions)),
	}

	transactionID, err := s.client.CreateTransaction(ctx, postings, metadata)
	if err != nil {
		return "", fmt.Errorf("failed to collect time-based rake: %w", err)
	}

	slog.Info("Collected time-based rake",
		"table_id", config.TableID,
		"rake_amount", config.TimeAmount,
		"players", len(playerSessions),
		"transaction_id", transactionID)

	return transactionID, nil
}

// collectTournamentRake collects rake as part of tournament buy-in (no actual collection needed)
func (s *Service) collectTournamentRake(ctx context.Context, config RakeConfig, playerSessions map[uuid.UUID]uuid.UUID) (string, error) {
	// Tournament rake is collected during buy-in, this is just for logging
	slog.Info("Tournament rake already collected during buy-in",
		"table_id", config.TableID,
		"players", len(playerSessions),
		"strategy", string(RakeStrategyTournament))

	return "tournament-rake-collected", nil
}

// DepositMoney adds money to a user's main account from the world (development)
func (s *Service) DepositMoney(ctx context.Context, userID uuid.UUID, amount int64) (string, error) {
	if amount <= 0 {
		return "", fmt.Errorf("amount must be positive")
	}

	userAccount := PlayerWalletAccount(userID)

	postings := []PostingSimple{
		{
			Source:      WorldAccount,
			Destination: userAccount,
			Amount:      amount,
			Asset:       s.currency,
		},
	}

	metadata := map[string]string{
		"type":    "deposit",
		"user_id": userID.String(),
	}

	transactionID, err := s.client.CreateTransaction(ctx, postings, metadata)
	if err != nil {
		return "", fmt.Errorf("failed to deposit money: %w", err)
	}

	slog.Info("Deposited money to user account", "user_id", userID, "amount", amount, "transaction_id", transactionID)
	return transactionID, nil
}

// WithdrawMoney removes money from a user's main account to the world (development)
func (s *Service) WithdrawMoney(ctx context.Context, userID uuid.UUID, amount int64) (string, error) {
	if amount <= 0 {
		return "", fmt.Errorf("amount must be positive")
	}

	userAccount := PlayerWalletAccount(userID)

	postings := []PostingSimple{
		{
			Source:      userAccount,
			Destination: WorldAccount,
			Amount:      amount,
			Asset:       s.currency,
		},
	}

	metadata := map[string]string{
		"type":    "withdrawal",
		"user_id": userID.String(),
	}

	transactionID, err := s.client.CreateTransaction(ctx, postings, metadata)
	if err != nil {
		return "", fmt.Errorf("failed to withdraw money: %w", err)
	}

	slog.Info("Withdrew money from user account", "user_id", userID, "amount", amount, "transaction_id", transactionID)
	return transactionID, nil
}

// ValidateSessionBalance checks if a session has sufficient balance for an operation
func (s *Service) ValidateSessionBalance(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID, amount int64) error {
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}

	if userID == uuid.Nil {
		return fmt.Errorf("user ID cannot be nil")
	}

	if sessionID == uuid.Nil {
		return fmt.Errorf("session ID cannot be nil")
	}

	sessionBalance, err := s.GetSessionBalance(ctx, userID, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session balance: %w", err)
	}

	if sessionBalance < amount {
		return fmt.Errorf("insufficient session balance: have %d, need %d", sessionBalance, amount)
	}

	return nil
}

// ValidateMainBalance checks if a user has sufficient main wallet balance
func (s *Service) ValidateMainBalance(ctx context.Context, userID uuid.UUID, amount int64) error {
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}

	if userID == uuid.Nil {
		return fmt.Errorf("user ID cannot be nil")
	}

	mainAccount := PlayerWalletAccount(userID)
	mainBalance, err := s.client.GetBalance(ctx, mainAccount)
	if err != nil {
		return fmt.Errorf("failed to get main balance: %w", err)
	}

	if mainBalance < amount {
		return fmt.Errorf("insufficient main balance: have %d, need %d", mainBalance, amount)
	}

	return nil
}

// ValidateSessionExists checks if a session exists and is active
func (s *Service) ValidateSessionExists(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID, db *gorm.DB) error {
	if userID == uuid.Nil {
		return fmt.Errorf("user ID cannot be nil")
	}

	if sessionID == uuid.Nil {
		return fmt.Errorf("session ID cannot be nil")
	}

	var session models.GameSession
	err := db.Where("id = ? AND user_id = ? AND status = ?", sessionID, userID, models.GameSessionStatusActive).First(&session).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("active session not found for user %s and session %s", userID, sessionID)
		}
		return fmt.Errorf("failed to query session: %w", err)
	}

	return nil
}

// CreateUserWallet initializes a wallet for a new user with zero balance
func (s *Service) CreateUserWallet(ctx context.Context, userID uuid.UUID) error {
	if userID == uuid.Nil {
		return fmt.Errorf("user ID cannot be nil")
	}

	userAccount := PlayerWalletAccount(userID)

	// Create wallet with zero balance transaction to establish the account
	postings := []PostingSimple{
		{
			Source:      WorldAccount,
			Destination: userAccount,
			Amount:      0, // Zero amount just to create the account
			Asset:       s.currency,
		},
	}

	metadata := map[string]string{
		"type":    "wallet_creation",
		"user_id": userID.String(),
	}

	transactionID, err := s.client.CreateTransaction(ctx, postings, metadata)
	if err != nil {
		return fmt.Errorf("failed to create user wallet: %w", err)
	}

	slog.Info("Created user wallet", "user_id", userID, "account", userAccount, "transaction_id", transactionID)
	return nil
}

// GetTransactionHistory fetches transaction history for a user (legacy method)
func (s *Service) GetTransactionHistory(ctx context.Context, userID uuid.UUID, limit, offset int) ([]TransactionData, error) {
	return s.client.GetTransactionHistory(ctx, userID.String(), limit, offset)
}

// GetWalletTransactions fetches only wallet-related transactions (deposits/withdrawals)
func (s *Service) GetWalletTransactions(ctx context.Context, userID uuid.UUID, limit, offset int) ([]TransactionData, error) {
	// Get all transactions and filter client-side for now
	allTransactions, err := s.client.GetTransactionHistory(ctx, userID.String(), limit*2, offset)
	if err != nil {
		return nil, err
	}

	var walletTransactions []TransactionData
	for _, tx := range allTransactions {
		if txType, exists := tx.Metadata["type"]; exists {
			if typeStr, ok := txType.(string); ok {
				// Only include wallet-level transactions
				if typeStr == "deposit" || typeStr == "withdrawal" || typeStr == "tournament_buyin" ||
					typeStr == "tournament_prize" || typeStr == "rake_collection" {
					walletTransactions = append(walletTransactions, tx)
					if len(walletTransactions) >= limit {
						break
					}
				}
			}
		}
	}

	return walletTransactions, nil
}

// GetGameTransactions fetches only game-related transactions (buyin/cashout)
func (s *Service) GetGameTransactions(ctx context.Context, userID uuid.UUID, limit, offset int) ([]TransactionData, error) {
	// Get all transactions and filter client-side for now
	allTransactions, err := s.client.GetTransactionHistory(ctx, userID.String(), limit*2, offset)
	if err != nil {
		return nil, err
	}

	var gameTransactions []TransactionData
	for _, tx := range allTransactions {
		if txType, exists := tx.Metadata["type"]; exists {
			if typeStr, ok := txType.(string); ok {
				// Only include game-level transactions
				if typeStr == "game_buyin" || typeStr == "game_cashout" {
					gameTransactions = append(gameTransactions, tx)
					if len(gameTransactions) >= limit {
						break
					}
				}
			}
		}
	}

	return gameTransactions, nil
}
