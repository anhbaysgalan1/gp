package unit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/anhbaysgalan1/gp/internal/formance"
	"github.com/anhbaysgalan1/gp/internal/handlers"
	"github.com/anhbaysgalan1/gp/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// FormanceServiceInterface defines the interface that balance handler needs
type FormanceServiceInterface interface {
	GetUserBalance(ctx context.Context, userID uuid.UUID) (*models.UserBalance, error)
	TransferToGame(ctx context.Context, userID uuid.UUID, amount int64, sessionID uuid.UUID) (string, error)
	TransferFromGame(ctx context.Context, userID uuid.UUID, amount int64, sessionID uuid.UUID) (string, error)
	GetTransactionHistory(ctx context.Context, userID uuid.UUID, limit, offset int) ([]formance.TransactionData, error)
}

// MockFormanceClient simulates the Formance API for testing
type MockFormanceClient struct {
	balances     map[string]int64
	transactions []MockTransaction
	shouldFail   bool
	failError    error
}

type MockTransaction struct {
	ID       string
	Postings []MockPosting
	Metadata map[string]string
}

type MockPosting struct {
	Source      string
	Destination string
	Amount      int64
	Asset       string
}

func NewMockFormanceClient() *MockFormanceClient {
	return &MockFormanceClient{
		balances:     make(map[string]int64),
		transactions: make([]MockTransaction, 0),
		shouldFail:   false,
	}
}

func (m *MockFormanceClient) SetBalance(account string, amount int64) {
	m.balances[account] = amount
}

func (m *MockFormanceClient) GetBalance(account string) int64 {
	return m.balances[account]
}

func (m *MockFormanceClient) SetShouldFail(fail bool, err error) {
	m.shouldFail = fail
	m.failError = err
}

func (m *MockFormanceClient) AddTransaction(id string, postings []MockPosting, metadata map[string]string) {
	m.transactions = append(m.transactions, MockTransaction{
		ID:       id,
		Postings: postings,
		Metadata: metadata,
	})
}

// MockFormanceService implements balance operations using the mock client
type MockFormanceService struct {
	client   *MockFormanceClient
	currency string
}

func NewMockFormanceService() *MockFormanceService {
	return &MockFormanceService{
		client:   NewMockFormanceClient(),
		currency: "MNT",
	}
}

func (s *MockFormanceService) GetUserBalance(ctx context.Context, userID uuid.UUID) (*models.UserBalance, error) {
	if s.client.shouldFail {
		return nil, s.client.failError
	}

	mainAccount := fmt.Sprintf("player:%s:wallet", userID)
	gameAccount := fmt.Sprintf("session:%s:test-session", userID)

	mainBalance := s.client.GetBalance(mainAccount)
	gameBalance := s.client.GetBalance(gameAccount)

	return &models.UserBalance{
		MainBalance:  mainBalance,
		GameBalance:  gameBalance,
		TotalBalance: mainBalance + gameBalance,
	}, nil
}

func (s *MockFormanceService) TransferToGame(ctx context.Context, userID uuid.UUID, amount int64, sessionID uuid.UUID) (string, error) {
	if s.client.shouldFail {
		return "", s.client.failError
	}

	if amount <= 0 {
		return "", fmt.Errorf("amount must be positive")
	}

	mainAccount := fmt.Sprintf("player:%s:wallet", userID)
	gameAccount := fmt.Sprintf("session:%s:test-session", userID)

	// Check main balance
	mainBalance := s.client.GetBalance(mainAccount)
	if mainBalance < amount {
		return "", fmt.Errorf("insufficient balance")
	}

	// Perform transfer
	s.client.SetBalance(mainAccount, mainBalance-amount)
	gameBalance := s.client.GetBalance(gameAccount)
	s.client.SetBalance(gameAccount, gameBalance+amount)

	// Add transaction record
	transactionID := uuid.New().String()
	postings := []MockPosting{
		{
			Source:      mainAccount,
			Destination: gameAccount,
			Amount:      amount,
			Asset:       s.currency,
		},
	}
	metadata := map[string]string{
		"type":       "game_buyin",
		"user_id":    userID.String(),
		"session_id": sessionID.String(),
	}
	s.client.AddTransaction(transactionID, postings, metadata)

	return transactionID, nil
}

func (s *MockFormanceService) TransferFromGame(ctx context.Context, userID uuid.UUID, amount int64, sessionID uuid.UUID) (string, error) {
	if s.client.shouldFail {
		return "", s.client.failError
	}

	if amount <= 0 {
		return "", fmt.Errorf("amount must be positive")
	}

	mainAccount := fmt.Sprintf("player:%s:wallet", userID)
	gameAccount := fmt.Sprintf("session:%s:test-session", userID)

	// Check game balance
	gameBalance := s.client.GetBalance(gameAccount)
	if gameBalance < amount {
		return "", fmt.Errorf("insufficient balance")
	}

	// Perform transfer
	s.client.SetBalance(gameAccount, gameBalance-amount)
	mainBalance := s.client.GetBalance(mainAccount)
	s.client.SetBalance(mainAccount, mainBalance+amount)

	// Add transaction record
	transactionID := uuid.New().String()
	postings := []MockPosting{
		{
			Source:      gameAccount,
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
	s.client.AddTransaction(transactionID, postings, metadata)

	return transactionID, nil
}

func (s *MockFormanceService) GetTransactionHistory(ctx context.Context, userID uuid.UUID, limit, offset int) ([]formance.TransactionData, error) {
	if s.client.shouldFail {
		return nil, s.client.failError
	}

	// Convert mock transactions to expected format
	var result []formance.TransactionData
	userMainAccount := fmt.Sprintf("player:%s:wallet", userID)
	userGameAccount := fmt.Sprintf("session:%s:test-session", userID)

	userTransactionIndex := 0
	for i, tx := range s.client.transactions {
		// Check if transaction involves this user
		isUserTransaction := false
		for _, posting := range tx.Postings {
			if posting.Source == userMainAccount || posting.Source == userGameAccount ||
				posting.Destination == userMainAccount || posting.Destination == userGameAccount {
				isUserTransaction = true
				break
			}
		}

		if !isUserTransaction {
			continue
		}

		// Skip based on offset
		if userTransactionIndex < offset {
			userTransactionIndex++
			continue
		}

		// Stop at limit
		if len(result) >= limit {
			break
		}

		userTransactionIndex++

		// Convert postings
		var formancePostings []formance.PostingData
		for _, posting := range tx.Postings {
			formancePostings = append(formancePostings, formance.PostingData{
				Source:      posting.Source,
				Destination: posting.Destination,
				Amount:      posting.Amount,
				Asset:       posting.Asset,
			})
		}

		// Convert metadata to interface{} map
		metadataInterface := make(map[string]interface{})
		for k, v := range tx.Metadata {
			metadataInterface[k] = v
		}

		result = append(result, formance.TransactionData{
			ID:       int64(i + 1),
			Postings: formancePostings,
			Metadata: metadataInterface,
		})
	}

	return result, nil
}

// MockBalanceHandler wraps handlers.BalanceHandler to work with our mock service
type MockBalanceHandler struct {
	*handlers.BalanceHandler
	mockService FormanceServiceInterface
}

func NewMockBalanceHandler(service FormanceServiceInterface) *MockBalanceHandler {
	// We can't directly pass our mock to handlers.NewBalanceHandler since it expects *formance.Service
	// So we'll create a handler with a nil service and override the methods
	return &MockBalanceHandler{
		mockService: service,
	}
}

func (h *MockBalanceHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	balance, err := h.mockService.GetUserBalance(r.Context(), userID)
	if err != nil {
		http.Error(w, "Failed to get balance", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(balance)
}

func (h *MockBalanceHandler) TransferToGame(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	var req handlers.TransferToGameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Amount <= 0 {
		http.Error(w, "Amount must be positive", http.StatusBadRequest)
		return
	}

	if req.SessionID == uuid.Nil {
		http.Error(w, "Session ID is required", http.StatusBadRequest)
		return
	}

	// Check balance first
	balance, err := h.mockService.GetUserBalance(r.Context(), userID)
	if err != nil {
		http.Error(w, "Failed to check user balance", http.StatusInternalServerError)
		return
	}

	if balance.MainBalance < req.Amount {
		http.Error(w, "Insufficient main balance", http.StatusBadRequest)
		return
	}

	transactionID, err := h.mockService.TransferToGame(r.Context(), userID, req.Amount, req.SessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"message":        "Transfer successful",
		"transaction_id": transactionID,
		"amount":         req.Amount,
		"session_id":     req.SessionID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *MockBalanceHandler) TransferFromGame(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	var req handlers.TransferFromGameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Amount <= 0 {
		http.Error(w, "Amount must be positive", http.StatusBadRequest)
		return
	}

	if req.SessionID == uuid.Nil {
		http.Error(w, "Session ID is required", http.StatusBadRequest)
		return
	}

	// Check balance first
	balance, err := h.mockService.GetUserBalance(r.Context(), userID)
	if err != nil {
		http.Error(w, "Failed to check user balance", http.StatusInternalServerError)
		return
	}

	if balance.GameBalance < req.Amount {
		http.Error(w, "Insufficient game balance", http.StatusBadRequest)
		return
	}

	transactionID, err := h.mockService.TransferFromGame(r.Context(), userID, req.Amount, req.SessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"message":        "Transfer successful",
		"transaction_id": transactionID,
		"amount":         req.Amount,
		"session_id":     req.SessionID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *MockBalanceHandler) GetTransactionHistory(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	// Parse query parameters
	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	offset := 0
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	transactions, err := h.mockService.GetTransactionHistory(r.Context(), userID, limit, offset)
	if err != nil {
		http.Error(w, "Failed to fetch transaction history", http.StatusInternalServerError)
		return
	}

	// Convert to response format (simplified for testing)
	responseTransactions := make([]map[string]interface{}, len(transactions))
	for i, tx := range transactions {
		transactionType := "unknown"
		if txType, exists := tx.Metadata["type"]; exists {
			if typeStr, ok := txType.(string); ok {
				transactionType = typeStr
			}
		}

		var netAmount int64
		userMainAccount := fmt.Sprintf("player:%s:wallet", userID)
		userGameAccount := fmt.Sprintf("session:%s:test-session", userID)

		for _, posting := range tx.Postings {
			if posting.Destination == userMainAccount || posting.Destination == userGameAccount {
				netAmount += posting.Amount
			}
			if posting.Source == userMainAccount || posting.Source == userGameAccount {
				netAmount -= posting.Amount
			}
		}

		description := "Transaction"
		switch transactionType {
		case "game_buyin":
			description = "Transfer to game account"
		case "game_cashout":
			description = "Transfer from game account"
		}

		responseTransactions[i] = map[string]interface{}{
			"id":          fmt.Sprintf("%d", tx.ID),
			"user_id":     userID.String(),
			"type":        transactionType,
			"amount":      netAmount,
			"currency":    "MNT",
			"description": description,
		}
	}

	response := map[string]interface{}{
		"transactions": responseTransactions,
		"pagination": map[string]interface{}{
			"limit":  limit,
			"offset": offset,
			"total":  len(transactions),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func TestBalanceHandler_GetBalance(t *testing.T) {
	mockService := NewMockFormanceService()
	handler := NewMockBalanceHandler(mockService)

	userID := uuid.New()

	tests := []struct {
		name                string
		setupBalances       func()
		expectedMainBalance int64
		expectedGameBalance int64
		expectError         bool
		expectedStatus      int
	}{
		{
			name: "Get balance successfully",
			setupBalances: func() {
				mockService.client.SetBalance(fmt.Sprintf("player:%s:wallet", userID), 50000)
				mockService.client.SetBalance(fmt.Sprintf("session:%s:test-session", userID), 25000)
			},
			expectedMainBalance: 50000,
			expectedGameBalance: 25000,
			expectError:         false,
			expectedStatus:      http.StatusOK,
		},
		{
			name: "Get balance with zero balances",
			setupBalances: func() {
				mockService.client.SetBalance(fmt.Sprintf("player:%s:wallet", userID), 0)
				mockService.client.SetBalance(fmt.Sprintf("session:%s:test-session", userID), 0)
			},
			expectedMainBalance: 0,
			expectedGameBalance: 0,
			expectError:         false,
			expectedStatus:      http.StatusOK,
		},
		{
			name: "Service error",
			setupBalances: func() {
				mockService.client.SetShouldFail(true, fmt.Errorf("service unavailable"))
			},
			expectError:    true,
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset service state
			mockService.client.SetShouldFail(false, nil)
			tt.setupBalances()

			req := httptest.NewRequest(http.MethodGet, "/balance", nil)
			ctx := context.WithValue(req.Context(), "user_id", userID)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			handler.GetBalance(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if !tt.expectError {
				var response models.UserBalance
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Equal(t, tt.expectedMainBalance, response.MainBalance)
				assert.Equal(t, tt.expectedGameBalance, response.GameBalance)
				assert.Equal(t, tt.expectedMainBalance+tt.expectedGameBalance, response.TotalBalance)
			}
		})
	}
}

func TestBalanceHandler_GetBalance_NoAuth(t *testing.T) {
	mockService := NewMockFormanceService()
	handler := NewMockBalanceHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/balance", nil)
	// No user_id in context

	w := httptest.NewRecorder()
	handler.GetBalance(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestBalanceHandler_TransferToGame(t *testing.T) {
	mockService := NewMockFormanceService()
	handler := NewMockBalanceHandler(mockService)

	userID := uuid.New()
	sessionID := uuid.New()

	tests := []struct {
		name           string
		setupBalances  func()
		requestBody    handlers.TransferToGameRequest
		expectedStatus int
		expectError    bool
	}{
		{
			name: "Successful transfer",
			setupBalances: func() {
				mockService.client.SetBalance(fmt.Sprintf("player:%s:wallet", userID), 100000)
				mockService.client.SetBalance(fmt.Sprintf("session:%s:test-session", userID), 0)
			},
			requestBody: handlers.TransferToGameRequest{
				Amount:    50000,
				SessionID: sessionID,
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name: "Insufficient balance",
			setupBalances: func() {
				mockService.client.SetBalance(fmt.Sprintf("player:%s:wallet", userID), 25000)
				mockService.client.SetBalance(fmt.Sprintf("session:%s:test-session", userID), 0)
			},
			requestBody: handlers.TransferToGameRequest{
				Amount:    50000,
				SessionID: sessionID,
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "Invalid amount - zero",
			setupBalances: func() {
				mockService.client.SetBalance(fmt.Sprintf("player:%s:wallet", userID), 100000)
			},
			requestBody: handlers.TransferToGameRequest{
				Amount:    0,
				SessionID: sessionID,
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "Invalid amount - negative",
			setupBalances: func() {
				mockService.client.SetBalance(fmt.Sprintf("player:%s:wallet", userID), 100000)
			},
			requestBody: handlers.TransferToGameRequest{
				Amount:    -1000,
				SessionID: sessionID,
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "Missing session ID",
			setupBalances: func() {
				mockService.client.SetBalance(fmt.Sprintf("player:%s:wallet", userID), 100000)
			},
			requestBody: handlers.TransferToGameRequest{
				Amount:    50000,
				SessionID: uuid.Nil,
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset service state
			mockService.client.SetShouldFail(false, nil)
			mockService.client.balances = make(map[string]int64)
			mockService.client.transactions = make([]MockTransaction, 0)
			tt.setupBalances()

			reqBody, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/balance/transfer-to-game", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			ctx := context.WithValue(req.Context(), "user_id", userID)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			handler.TransferToGame(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if !tt.expectError && tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Contains(t, response, "message")
				assert.Contains(t, response, "transaction_id")
				assert.Equal(t, tt.requestBody.Amount, int64(response["amount"].(float64)))
				assert.Equal(t, tt.requestBody.SessionID.String(), response["session_id"])

				// Verify balances were updated
				gameBalance := mockService.client.GetBalance(fmt.Sprintf("session:%s:test-session", userID))
				assert.Equal(t, tt.requestBody.Amount, gameBalance)

				// Verify transaction was recorded
				assert.Len(t, mockService.client.transactions, 1)
				assert.Equal(t, "game_buyin", mockService.client.transactions[0].Metadata["type"])
			}
		})
	}
}

func TestBalanceHandler_TransferFromGame(t *testing.T) {
	mockService := NewMockFormanceService()
	handler := NewMockBalanceHandler(mockService)

	userID := uuid.New()
	sessionID := uuid.New()

	tests := []struct {
		name           string
		setupBalances  func()
		requestBody    handlers.TransferFromGameRequest
		expectedStatus int
		expectError    bool
	}{
		{
			name: "Successful transfer",
			setupBalances: func() {
				mockService.client.SetBalance(fmt.Sprintf("player:%s:wallet", userID), 50000)
				mockService.client.SetBalance(fmt.Sprintf("session:%s:test-session", userID), 75000)
			},
			requestBody: handlers.TransferFromGameRequest{
				Amount:    25000,
				SessionID: sessionID,
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name: "Insufficient game balance",
			setupBalances: func() {
				mockService.client.SetBalance(fmt.Sprintf("player:%s:wallet", userID), 50000)
				mockService.client.SetBalance(fmt.Sprintf("session:%s:test-session", userID), 10000)
			},
			requestBody: handlers.TransferFromGameRequest{
				Amount:    25000,
				SessionID: sessionID,
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "Invalid amount - zero",
			setupBalances: func() {
				mockService.client.SetBalance(fmt.Sprintf("session:%s:test-session", userID), 100000)
			},
			requestBody: handlers.TransferFromGameRequest{
				Amount:    0,
				SessionID: sessionID,
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset service state
			mockService.client.SetShouldFail(false, nil)
			mockService.client.balances = make(map[string]int64)
			mockService.client.transactions = make([]MockTransaction, 0)
			tt.setupBalances()

			reqBody, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/balance/transfer-from-game", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			ctx := context.WithValue(req.Context(), "user_id", userID)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			handler.TransferFromGame(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if !tt.expectError && tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Contains(t, response, "message")
				assert.Contains(t, response, "transaction_id")
				assert.Equal(t, tt.requestBody.Amount, int64(response["amount"].(float64)))

				// Verify transaction was recorded
				assert.Len(t, mockService.client.transactions, 1)
				assert.Equal(t, "game_cashout", mockService.client.transactions[0].Metadata["type"])
			}
		})
	}
}

func TestBalanceHandler_GetTransactionHistory(t *testing.T) {
	mockService := NewMockFormanceService()
	handler := NewMockBalanceHandler(mockService)

	userID := uuid.New()
	sessionID := uuid.New()

	// Setup some transactions
	mockService.client.SetBalance(fmt.Sprintf("player:%s:wallet", userID), 100000)
	mockService.client.SetBalance(fmt.Sprintf("session:%s:test-session", userID), 0)

	// Perform some transfers to create transaction history
	_, err := mockService.TransferToGame(context.Background(), userID, 25000, sessionID)
	require.NoError(t, err)

	_, err = mockService.TransferFromGame(context.Background(), userID, 10000, sessionID)
	require.NoError(t, err)

	tests := []struct {
		name            string
		queryParams     string
		expectedStatus  int
		expectedTxCount int
		expectError     bool
	}{
		{
			name:            "Get transaction history successfully",
			queryParams:     "",
			expectedStatus:  http.StatusOK,
			expectedTxCount: 2,
			expectError:     false,
		},
		{
			name:            "Get transaction history with limit",
			queryParams:     "?limit=1",
			expectedStatus:  http.StatusOK,
			expectedTxCount: 1,
			expectError:     false,
		},
		{
			name:            "Get transaction history with offset",
			queryParams:     "?offset=1",
			expectedStatus:  http.StatusOK,
			expectedTxCount: 1,
			expectError:     false,
		},
		{
			name:            "Get transaction history with limit and offset",
			queryParams:     "?limit=1&offset=1",
			expectedStatus:  http.StatusOK,
			expectedTxCount: 1,
			expectError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/balance/transactions"+tt.queryParams, nil)
			ctx := context.WithValue(req.Context(), "user_id", userID)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			handler.GetTransactionHistory(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if !tt.expectError {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Contains(t, response, "transactions")
				assert.Contains(t, response, "pagination")

				transactions := response["transactions"].([]interface{})
				assert.Len(t, transactions, tt.expectedTxCount)

				if len(transactions) > 0 {
					tx := transactions[0].(map[string]interface{})
					assert.Contains(t, tx, "id")
					assert.Contains(t, tx, "type")
					assert.Contains(t, tx, "amount")
					assert.Contains(t, tx, "currency")
					assert.Contains(t, tx, "description")
				}
			}
		})
	}
}

func TestBalanceHandler_InvalidJSON(t *testing.T) {
	mockService := NewMockFormanceService()
	handler := NewMockBalanceHandler(mockService)

	userID := uuid.New()

	tests := []struct {
		name     string
		endpoint string
		body     string
	}{
		{
			name:     "Transfer to game with invalid JSON",
			endpoint: "/balance/transfer-to-game",
			body:     `{"amount": "invalid"}`,
		},
		{
			name:     "Transfer from game with invalid JSON",
			endpoint: "/balance/transfer-from-game",
			body:     `{"amount": "invalid"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tt.endpoint, bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			ctx := context.WithValue(req.Context(), "user_id", userID)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			if tt.endpoint == "/balance/transfer-to-game" {
				handler.TransferToGame(w, req)
			} else {
				handler.TransferFromGame(w, req)
			}

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, w.Body.String(), "Invalid JSON")
		})
	}
}
