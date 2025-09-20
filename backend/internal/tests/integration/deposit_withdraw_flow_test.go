package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anhbaysgalan1/gp/internal/config"
	"github.com/anhbaysgalan1/gp/internal/database"
	"github.com/anhbaysgalan1/gp/internal/formance"
	"github.com/anhbaysgalan1/gp/internal/handlers"
	"github.com/anhbaysgalan1/gp/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type DepositWithdrawFlowTestSuite struct {
	suite.Suite
	db              *database.DB
	formanceService *formance.Service
	adminHandler    *handlers.AdminHandler
	tableHandler    *handlers.TableHandler
	balanceHandler  *handlers.BalanceHandler
	router          *chi.Mux
	testUser        *models.User
	testTable       *models.PokerTable
}

func TestDepositWithdrawFlowSuite(t *testing.T) {
	suite.Run(t, new(DepositWithdrawFlowTestSuite))
}

func (s *DepositWithdrawFlowTestSuite) SetupSuite() {
	// Load test configuration
	cfg := &config.Config{
		DatabaseURL:      "postgres://user:password@localhost:5432/test_db?sslmode=disable",
		JWTSecret:        "test-secret",
		FormanceURL:      "http://localhost:3068",
		FormanceLedger:   "test-ledger",
		FormanceCurrency: "MNT",
		Environment:      "test",
	}

	// Setup test database
	var err error
	s.db, err = database.NewConnection(cfg)
	s.Require().NoError(err)

	// Run migrations
	err = s.db.AutoMigrate()
	s.Require().NoError(err)

	// Setup Formance service (this will use mock in test environment)
	s.formanceService = formance.NewService(cfg)

	// Setup handlers
	s.adminHandler = handlers.NewAdminHandler(s.db, s.formanceService)
	s.tableHandler = handlers.NewTableHandler(s.db, s.formanceService)
	s.balanceHandler = handlers.NewBalanceHandler(s.formanceService)

	// Setup router
	s.router = chi.NewRouter()
	s.router.Mount("/admin", s.adminHandler.Routes(nil)) // No role middleware for tests
	s.router.Mount("/tables", s.tableHandler.Routes())
	s.router.Mount("/balance", s.balanceHandler.Routes())
}

func (s *DepositWithdrawFlowTestSuite) SetupTest() {
	// Clean up database
	s.db.Exec("DELETE FROM game_sessions")
	s.db.Exec("DELETE FROM poker_tables")
	s.db.Exec("DELETE FROM users")

	// Create test user
	s.testUser = &models.User{
		ID:         uuid.New(),
		Username:   "testuser",
		Email:      "test@example.com",
		IsVerified: true,
		Role:       models.UserRolePlayer,
	}
	err := s.db.Create(s.testUser).Error
	s.Require().NoError(err)

	// Create test table
	s.testTable = &models.PokerTable{
		ID:             uuid.New(),
		Name:           "Test Table",
		TableType:      "cash",
		GameType:       "texas_holdem",
		MaxPlayers:     9,
		MinBuyIn:       10000,  // 100 MNT
		MaxBuyIn:       100000, // 1000 MNT
		SmallBlind:     500,    // 5 MNT
		BigBlind:       1000,   // 10 MNT
		IsPrivate:      false,
		Status:         "waiting",
		CurrentPlayers: 0,
		CreatedBy:      s.testUser.ID,
	}
	err = s.db.Create(s.testTable).Error
	s.Require().NoError(err)
}

func (s *DepositWithdrawFlowTestSuite) TearDownSuite() {
	if s.db != nil {
		s.db.Close()
	}
}

func (s *DepositWithdrawFlowTestSuite) TestCompleteDepositJoinLeaveFlow() {
	userID := s.testUser.ID
	tableID := s.testTable.ID

	// Step 1: Check initial balance (should be 0)
	balance := s.getUserBalance(userID)
	s.Assert().Equal(int64(0), balance.MainBalance)
	s.Assert().Equal(int64(0), balance.GameBalance)
	s.Assert().Equal(int64(0), balance.TotalBalance)

	// Step 2: Deposit money to user account
	depositAmount := int64(500000) // 5000 MNT
	transactionID := s.depositMoney(userID, depositAmount)
	s.Assert().NotEmpty(transactionID)

	// Verify balance after deposit
	balance = s.getUserBalance(userID)
	s.Assert().Equal(depositAmount, balance.MainBalance)
	s.Assert().Equal(int64(0), balance.GameBalance)
	s.Assert().Equal(depositAmount, balance.TotalBalance)

	// Step 3: Join table with buy-in
	buyInAmount := int64(50000) // 500 MNT
	joinResponse := s.joinTable(userID, tableID, buyInAmount)
	s.Assert().Contains(joinResponse, "session_id")
	s.Assert().Contains(joinResponse, "transaction_id")

	sessionID := joinResponse["session_id"].(string)
	sessionUUID, err := uuid.Parse(sessionID)
	s.Require().NoError(err)

	// Verify balance after joining (funds moved from main to game account)
	balance = s.getUserBalance(userID)
	s.Assert().Equal(depositAmount-buyInAmount, balance.MainBalance)
	s.Assert().Equal(buyInAmount, balance.GameBalance)
	s.Assert().Equal(depositAmount, balance.TotalBalance)

	// Verify table player count increased
	var updatedTable models.PokerTable
	err = s.db.First(&updatedTable, "id = ?", tableID).Error
	s.Require().NoError(err)
	s.Assert().Equal(1, updatedTable.CurrentPlayers)
	s.Assert().Equal("active", updatedTable.Status)

	// Verify game session was created
	var session models.GameSession
	err = s.db.First(&session, "id = ?", sessionUUID).Error
	s.Require().NoError(err)
	s.Assert().Equal(userID, session.UserID)
	s.Assert().Equal(tableID, session.TableID)
	s.Assert().Equal(buyInAmount, session.BuyInAmount)
	s.Assert().Equal(buyInAmount, session.CurrentChips)
	s.Assert().Equal(models.GameSessionStatusActive, session.Status)

	// Step 4: Simulate some gameplay by updating current chips
	// (In real gameplay, chips would change based on wins/losses)
	newChipAmount := int64(75000) // Won 250 MNT
	err = s.db.Model(&session).Update("current_chips", newChipAmount).Error
	s.Require().NoError(err)

	// Step 5: Leave table (cash out)
	leaveResponse := s.leaveTable(userID, tableID)
	s.Assert().Contains(leaveResponse, "message")
	s.Assert().Contains(leaveResponse, "chips_returned")
	s.Assert().Contains(leaveResponse, "net_result")
	s.Assert().Contains(leaveResponse, "transaction_id")

	chipsReturned := int64(leaveResponse["chips_returned"].(float64))
	netResult := int64(leaveResponse["net_result"].(float64))
	s.Assert().Equal(newChipAmount, chipsReturned)
	s.Assert().Equal(newChipAmount-buyInAmount, netResult) // Net profit

	// Verify balance after leaving (funds moved back to main account)
	balance = s.getUserBalance(userID)
	expectedMainBalance := depositAmount - buyInAmount + newChipAmount
	s.Assert().Equal(expectedMainBalance, balance.MainBalance)
	s.Assert().Equal(int64(0), balance.GameBalance)
	s.Assert().Equal(expectedMainBalance, balance.TotalBalance)

	// Verify table player count decreased
	err = s.db.First(&updatedTable, "id = ?", tableID).Error
	s.Require().NoError(err)
	s.Assert().Equal(0, updatedTable.CurrentPlayers)
	s.Assert().Equal("waiting", updatedTable.Status)

	// Verify game session was finished
	err = s.db.First(&session, "id = ?", sessionUUID).Error
	s.Require().NoError(err)
	s.Assert().Equal(models.GameSessionStatusFinished, session.Status)
	s.Assert().NotNil(session.LeftAt)

	// Step 6: Test withdrawal
	withdrawAmount := int64(100000) // 1000 MNT
	withdrawTransactionID := s.withdrawMoney(userID, withdrawAmount)
	s.Assert().NotEmpty(withdrawTransactionID)

	// Verify balance after withdrawal
	balance = s.getUserBalance(userID)
	s.Assert().Equal(expectedMainBalance-withdrawAmount, balance.MainBalance)
	s.Assert().Equal(int64(0), balance.GameBalance)
	s.Assert().Equal(expectedMainBalance-withdrawAmount, balance.TotalBalance)
}

func (s *DepositWithdrawFlowTestSuite) TestInsufficientBalanceForJoin() {
	userID := s.testUser.ID
	tableID := s.testTable.ID

	// Try to join table without sufficient balance
	buyInAmount := int64(50000) // 500 MNT

	reqBody := map[string]interface{}{
		"buy_in_amount": buyInAmount,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/tables/%s/join", tableID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), "user_id", userID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Assert().Equal(http.StatusBadRequest, w.Code)
	s.Assert().Contains(w.Body.String(), "Insufficient balance")
}

func (s *DepositWithdrawFlowTestSuite) TestDuplicateJoinPrevention() {
	userID := s.testUser.ID
	tableID := s.testTable.ID

	// Deposit money first
	s.depositMoney(userID, 100000)

	// Join table first time
	buyInAmount := int64(50000)
	s.joinTable(userID, tableID, buyInAmount)

	// Try to join same table again
	reqBody := map[string]interface{}{
		"buy_in_amount": buyInAmount,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/tables/%s/join", tableID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), "user_id", userID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Assert().Equal(http.StatusBadRequest, w.Code)
	s.Assert().Contains(w.Body.String(), "already has an active session")
}

func (s *DepositWithdrawFlowTestSuite) TestLeaveTableNotAtTable() {
	userID := s.testUser.ID
	tableID := s.testTable.ID

	// Try to leave table without being at the table
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/tables/%s/leave", tableID), nil)
	ctx := context.WithValue(req.Context(), "user_id", userID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Assert().Equal(http.StatusBadRequest, w.Code)
	s.Assert().Contains(w.Body.String(), "not currently at this table")
}

// Helper methods

func (s *DepositWithdrawFlowTestSuite) getUserBalance(userID uuid.UUID) *models.UserBalance {
	req := httptest.NewRequest(http.MethodGet, "/balance", nil)
	ctx := context.WithValue(req.Context(), "user_id", userID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Require().Equal(http.StatusOK, w.Code)

	var balance models.UserBalance
	err := json.Unmarshal(w.Body.Bytes(), &balance)
	s.Require().NoError(err)

	return &balance
}

func (s *DepositWithdrawFlowTestSuite) depositMoney(userID uuid.UUID, amount int64) string {
	reqBody := map[string]interface{}{
		"amount": amount,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/admin/users/%s/deposit", userID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), "user_id", userID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Require().Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.Require().NoError(err)

	transactionID, ok := response["transaction_id"].(string)
	s.Require().True(ok)

	return transactionID
}

func (s *DepositWithdrawFlowTestSuite) withdrawMoney(userID uuid.UUID, amount int64) string {
	reqBody := map[string]interface{}{
		"amount": amount,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/admin/users/%s/withdraw", userID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), "user_id", userID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Require().Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.Require().NoError(err)

	transactionID, ok := response["transaction_id"].(string)
	s.Require().True(ok)

	return transactionID
}

func (s *DepositWithdrawFlowTestSuite) joinTable(userID, tableID uuid.UUID, buyInAmount int64) map[string]interface{} {
	reqBody := map[string]interface{}{
		"buy_in_amount": buyInAmount,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/tables/%s/join", tableID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), "user_id", userID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Require().Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.Require().NoError(err)

	return response
}

func (s *DepositWithdrawFlowTestSuite) leaveTable(userID, tableID uuid.UUID) map[string]interface{} {
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/tables/%s/leave", tableID), nil)
	ctx := context.WithValue(req.Context(), "user_id", userID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Require().Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.Require().NoError(err)

	return response
}
