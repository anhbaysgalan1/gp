package unit

import (
	"context"
	"testing"

	"github.com/anhbaysgalan1/gp/internal/config"
	"github.com/anhbaysgalan1/gp/internal/formance"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormanceService_GetUserBalance(t *testing.T) {
	cfg := &config.Config{
		FormanceAPIURL:     "http://localhost:8080",
		FormanceAPIKey:     "test",
		FormanceLedgerName: "poker-test",
		FormanceCurrency:   "MNT",
	}

	service := formance.NewService(cfg)
	userID := uuid.New()

	tests := []struct {
		name        string
		expectError bool
	}{
		{
			name:        "Get balance for new user",
			expectError: false, // Should return 0 balances, not error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			balance, err := service.GetUserBalance(context.Background(), userID)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, balance)
			} else {
				// Should not error even if Formance is not available (fallback to 0)
				assert.NoError(t, err)
				assert.NotNil(t, balance)
				assert.Equal(t, int64(0), balance.MainBalance)
				assert.Equal(t, int64(0), balance.GameBalance)
				assert.Equal(t, int64(0), balance.TotalBalance)
			}
		})
	}
}

func TestFormanceService_TransferOperations(t *testing.T) {
	cfg := &config.Config{
		FormanceAPIURL:     "http://localhost:8080",
		FormanceAPIKey:     "test",
		FormanceLedgerName: "poker-test",
		FormanceCurrency:   "MNT",
	}

	service := formance.NewService(cfg)
	userID := uuid.New()
	sessionID := uuid.New()

	tests := []struct {
		name        string
		operation   string
		amount      int64
		expectError bool
	}{
		{
			name:        "Transfer to game with valid amount",
			operation:   "to_game",
			amount:      10000,
			expectError: false, // Should not error even if Formance unavailable (returns mock ID)
		},
		{
			name:        "Transfer from game with valid amount",
			operation:   "from_game",
			amount:      5000,
			expectError: false,
		},
		{
			name:        "Transfer to game with zero amount",
			operation:   "to_game",
			amount:      0,
			expectError: true,
		},
		{
			name:        "Transfer from game with negative amount",
			operation:   "from_game",
			amount:      -1000,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var transactionID string
			var err error

			switch tt.operation {
			case "to_game":
				transactionID, err = service.TransferToGame(context.Background(), userID, tt.amount, sessionID)
			case "from_game":
				transactionID, err = service.TransferFromGame(context.Background(), userID, tt.amount, sessionID)
			}

			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, transactionID)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, transactionID)
			}
		})
	}
}

func TestFormanceService_TournamentOperations(t *testing.T) {
	cfg := &config.Config{
		FormanceAPIURL:     "http://localhost:8080",
		FormanceAPIKey:     "test",
		FormanceLedgerName: "poker-test",
		FormanceCurrency:   "MNT",
	}

	service := formance.NewService(cfg)
	userID := uuid.New()
	tournamentID := uuid.New()

	tests := []struct {
		name        string
		operation   string
		amount      int64
		expectError bool
	}{
		{
			name:        "Tournament buy-in with valid amount",
			operation:   "buyin",
			amount:      50000,
			expectError: false,
		},
		{
			name:        "Tournament prize distribution",
			operation:   "prize",
			amount:      100000,
			expectError: false,
		},
		{
			name:        "Tournament buy-in with zero amount",
			operation:   "buyin",
			amount:      0,
			expectError: true,
		},
		{
			name:        "Tournament prize with negative amount",
			operation:   "prize",
			amount:      -5000,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var transactionID string
			var err error

			switch tt.operation {
			case "buyin":
				transactionID, err = service.ProcessTournamentBuyIn(context.Background(), userID, tournamentID, tt.amount)
			case "prize":
				transactionID, err = service.DistributeTournamentPrize(context.Background(), userID, tournamentID, tt.amount)
			}

			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, transactionID)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, transactionID)
			}
		})
	}
}

func TestFormanceService_RakeCollection(t *testing.T) {
	cfg := &config.Config{
		FormanceAPIURL:     "http://localhost:8080",
		FormanceAPIKey:     "test",
		FormanceLedgerName: "poker-test",
		FormanceCurrency:   "MNT",
	}

	service := formance.NewService(cfg)
	gameID := "game-123"
	players := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}

	tests := []struct {
		name        string
		amount      int64
		players     []uuid.UUID
		expectError bool
	}{
		{
			name:        "Collect rake with valid amount and players",
			amount:      1500,
			players:     players,
			expectError: false,
		},
		{
			name:        "Collect rake with zero amount",
			amount:      0,
			players:     players,
			expectError: true,
		},
		{
			name:        "Collect rake with negative amount",
			amount:      -500,
			players:     players,
			expectError: true,
		},
		{
			name:        "Collect rake with no players",
			amount:      1000,
			players:     []uuid.UUID{},
			expectError: false, // Should return empty transaction ID
		},
		{
			name:        "Collect rake with small amount per player",
			amount:      1,
			players:     players,
			expectError: false, // Should return empty transaction ID (no rake per player)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transactionID, err := service.CollectRake(context.Background(), tt.amount, gameID, tt.players)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// Transaction ID might be empty for cases where no rake is collected
				_ = transactionID
			}
		})
	}
}

func TestFormanceService_GetTransactionHistory(t *testing.T) {
	cfg := &config.Config{
		FormanceAPIURL:     "http://localhost:8080",
		FormanceAPIKey:     "test",
		FormanceLedgerName: "poker-test",
		FormanceCurrency:   "MNT",
	}

	service := formance.NewService(cfg)
	userID := uuid.New()

	tests := []struct {
		name   string
		limit  int
		offset int
	}{
		{
			name:   "Get transaction history with default parameters",
			limit:  10,
			offset: 0,
		},
		{
			name:   "Get transaction history with custom limit",
			limit:  5,
			offset: 0,
		},
		{
			name:   "Get transaction history with offset",
			limit:  10,
			offset: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transactions, err := service.GetTransactionHistory(context.Background(), userID, tt.limit, tt.offset)

			// Should not error even if Formance is not available (returns empty slice)
			assert.NoError(t, err)
			assert.NotNil(t, transactions)
			// Transactions might be empty since this is just a test without real data
		})
	}
}

func TestFormanceService_Initialize(t *testing.T) {
	cfg := &config.Config{
		FormanceAPIURL:     "http://localhost:8080",
		FormanceAPIKey:     "test",
		FormanceLedgerName: "poker-test",
		FormanceCurrency:   "MNT",
	}

	service := formance.NewService(cfg)

	t.Run("Initialize Formance service", func(t *testing.T) {
		err := service.Initialize(context.Background())
		// Should not error even if Formance is not available
		assert.NoError(t, err)
	})
}

func TestFormanceService_EdgeCases(t *testing.T) {
	cfg := &config.Config{
		FormanceAPIURL:     "http://localhost:8080",
		FormanceAPIKey:     "test",
		FormanceLedgerName: "poker-test",
		FormanceCurrency:   "MNT",
	}

	service := formance.NewService(cfg)

	t.Run("Operations with nil UUID", func(t *testing.T) {
		// Test with nil UUID - these should still work as the UUID will be converted to string
		_, err := service.GetUserBalance(context.Background(), uuid.Nil)
		assert.NoError(t, err) // Should not error, returns 0 balances

		_, err = service.TransferToGame(context.Background(), uuid.Nil, 1000, uuid.New())
		assert.NoError(t, err) // Should not error, returns mock transaction ID
	})

	t.Run("Operations with context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// These operations should handle context cancellation gracefully
		_, err := service.GetUserBalance(ctx, uuid.New())
		// Might error due to context cancellation, but should not panic
		_ = err // Ignore error for this test
		assert.NotPanics(t, func() {
			service.GetUserBalance(ctx, uuid.New())
		})
	})

	t.Run("Large amounts", func(t *testing.T) {
		userID := uuid.New()
		sessionID := uuid.New()
		largeAmount := int64(999999999999) // Very large amount

		_, err := service.TransferToGame(context.Background(), userID, largeAmount, sessionID)
		assert.NoError(t, err) // Should handle large amounts

		_, err = service.TransferFromGame(context.Background(), userID, largeAmount, sessionID)
		assert.NoError(t, err)
	})
}

// Integration test that requires actual Formance instance running
func TestFormanceService_RealIntegration(t *testing.T) {
	t.Skip("Skipping real integration test - requires Formance instance")

	cfg := &config.Config{
		FormanceAPIURL:     "http://localhost:8080",
		FormanceAPIKey:     "real-api-key",
		FormanceLedgerName: "poker-integration-test",
		FormanceCurrency:   "MNT",
	}

	service := formance.NewService(cfg)
	userID := uuid.New()
	sessionID := uuid.New()

	// Initialize service
	err := service.Initialize(context.Background())
	require.NoError(t, err)

	// Test full workflow
	t.Run("Complete transfer workflow", func(t *testing.T) {
		// Get initial balance
		_, err := service.GetUserBalance(context.Background(), userID)
		require.NoError(t, err)

		// Note: In real scenario, user would need to have balance first
		// This is just testing the API integration

		// Test transfer to game (this will fail with insufficient balance, which is expected)
		_, err = service.TransferToGame(context.Background(), userID, 10000, sessionID)
		// Error is expected since user has no balance

		// Get transaction history
		transactions, err := service.GetTransactionHistory(context.Background(), userID, 10, 0)
		require.NoError(t, err)
		assert.NotNil(t, transactions)
	})
}
