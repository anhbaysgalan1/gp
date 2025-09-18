package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/evanofslack/go-poker/internal/auth"
	"github.com/evanofslack/go-poker/internal/formance"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BalanceHandler struct {
	formanceService *formance.Service
	db              *gorm.DB
}

func NewBalanceHandler(formanceService *formance.Service, db *gorm.DB) *BalanceHandler {
	return &BalanceHandler{
		formanceService: formanceService,
		db:              db,
	}
}

func (h *BalanceHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// All balance routes require authentication
	r.Get("/", h.GetBalance)
	r.Post("/transfer-to-game", h.TransferToGame)
	r.Post("/transfer-from-game", h.TransferFromGame)
	r.Post("/withdraw", h.WithdrawMoney)
	r.Get("/transactions", h.GetTransactionHistory)

	return r
}

// GetBalance returns the user's balance information
func (h *BalanceHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		writeErrorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	balance, err := h.formanceService.GetUserBalance(r.Context(), userID, h.db)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to get balance")
		return
	}

	writeJSONResponse(w, http.StatusOK, balance)
}

// TransferToGameRequest represents the request to transfer money to game account
type TransferToGameRequest struct {
	Amount    int64     `json:"amount" validate:"required,gt=0"`
	SessionID uuid.UUID `json:"session_id" validate:"required"`
}

// TransferToGame transfers money from main account to game account
func (h *BalanceHandler) TransferToGame(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		writeErrorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req TransferToGameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Basic validation
	if req.Amount <= 0 {
		writeErrorResponse(w, http.StatusBadRequest, "Amount must be positive")
		return
	}

	if req.SessionID == uuid.Nil {
		writeErrorResponse(w, http.StatusBadRequest, "Session ID is required")
		return
	}

	// Validate session exists
	if err := h.formanceService.ValidateSessionExists(r.Context(), userID, req.SessionID, h.db); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("Invalid session: %v", err))
		return
	}

	// Check if user has sufficient main balance
	if err := h.formanceService.ValidateMainBalance(r.Context(), userID, req.Amount); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("Insufficient main balance: %v", err))
		return
	}

	transactionID, err := h.formanceService.TransferToGame(r.Context(), userID, req.Amount, req.SessionID)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	response := map[string]interface{}{
		"message":        "Transfer successful",
		"transaction_id": transactionID,
		"amount":         req.Amount,
		"session_id":     req.SessionID,
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// TransferFromGameRequest represents the request to transfer money from game account
type TransferFromGameRequest struct {
	Amount    int64     `json:"amount" validate:"required,gt=0"`
	SessionID uuid.UUID `json:"session_id" validate:"required"`
}

// TransferFromGame transfers money from game account back to main account
func (h *BalanceHandler) TransferFromGame(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		writeErrorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req TransferFromGameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Basic validation
	if req.Amount <= 0 {
		writeErrorResponse(w, http.StatusBadRequest, "Amount must be positive")
		return
	}

	if req.SessionID == uuid.Nil {
		writeErrorResponse(w, http.StatusBadRequest, "Session ID is required")
		return
	}

	// Validate session exists
	if err := h.formanceService.ValidateSessionExists(r.Context(), userID, req.SessionID, h.db); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("Invalid session: %v", err))
		return
	}

	// Check if user has sufficient session balance
	if err := h.formanceService.ValidateSessionBalance(r.Context(), userID, req.SessionID, req.Amount); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("Insufficient session balance: %v", err))
		return
	}

	transactionID, err := h.formanceService.TransferFromGame(r.Context(), userID, req.Amount, req.SessionID)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	response := map[string]interface{}{
		"message":        "Transfer successful",
		"transaction_id": transactionID,
		"amount":         req.Amount,
		"session_id":     req.SessionID,
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// GetTransactionHistory returns the transaction history for the user
func (h *BalanceHandler) GetTransactionHistory(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		writeErrorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	limit := 10 // default
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	offsetStr := r.URL.Query().Get("offset")
	offset := 0 // default
	if offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	// Fetch transaction history from Formance
	transactions, err := h.formanceService.GetTransactionHistory(r.Context(), userID, limit, offset)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to fetch transaction history")
		return
	}

	// Convert to response format
	responseTransactions := make([]map[string]interface{}, len(transactions))
	for i, tx := range transactions {
		// Extract transaction type from metadata
		transactionType := "unknown"
		if txType, exists := tx.Metadata["type"]; exists {
			if typeStr, ok := txType.(string); ok {
				transactionType = typeStr
			}
		}

		// Calculate net amount for this user
		var netAmount int64
		var description string
		userWalletAccount := formance.PlayerWalletAccount(userID)
		userSessionPrefix := formance.SessionPrefix(userID)

		for _, posting := range tx.Postings {
			// Check wallet account
			if posting.Destination == userWalletAccount {
				netAmount += posting.Amount
			}
			if posting.Source == userWalletAccount {
				netAmount -= posting.Amount
			}
			// Check session accounts (any session for this user)
			if posting.Destination != "" && len(posting.Destination) > len(userSessionPrefix) && posting.Destination[:len(userSessionPrefix)] == userSessionPrefix {
				netAmount += posting.Amount
			}
			if posting.Source != "" && len(posting.Source) > len(userSessionPrefix) && posting.Source[:len(userSessionPrefix)] == userSessionPrefix {
				netAmount -= posting.Amount
			}
		}

		// Set description based on transaction type
		switch transactionType {
		case "game_buyin":
			description = "Transfer to game account"
		case "game_cashout":
			description = "Transfer from game account"
		case "tournament_buyin":
			description = "Tournament buy-in"
		case "tournament_prize":
			description = "Tournament prize"
		case "rake_collection":
			description = "Rake collection"
		case "withdrawal":
			description = "Withdrawal to external account"
		default:
			description = "Transaction"
		}

		responseTransactions[i] = map[string]interface{}{
			"id":          fmt.Sprintf("%d", tx.ID),
			"user_id":     userID.String(),
			"type":        transactionType,
			"amount":      netAmount,
			"currency":    "MNT", // TODO: Extract from posting asset
			"created_at":  tx.Date,
			"description": description,
		}
	}

	response := map[string]interface{}{
		"transactions": responseTransactions,
		"pagination": map[string]interface{}{
			"limit":  limit,
			"offset": offset,
			"total":  len(transactions), // Note: This is not the true total, just current batch size
		},
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// UserWithdrawRequest represents the request to withdraw money from main account
type UserWithdrawRequest struct {
	Amount int64 `json:"amount" validate:"required,gt=0"`
}

// WithdrawMoney allows users to withdraw money from their main account
func (h *BalanceHandler) WithdrawMoney(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		writeErrorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req UserWithdrawRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Basic validation
	if req.Amount <= 0 {
		writeErrorResponse(w, http.StatusBadRequest, "Amount must be positive")
		return
	}

	// Set minimum withdrawal amount (e.g., 1000 MNT = ~$0.40)
	if req.Amount < 1000 {
		writeErrorResponse(w, http.StatusBadRequest, "Minimum withdrawal amount is 1,000 MNT")
		return
	}

	// Check if user has sufficient main balance
	if err := h.formanceService.ValidateMainBalance(r.Context(), userID, req.Amount); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("Insufficient main balance: %v", err))
		return
	}

	// Process withdrawal through Formance
	transactionID, err := h.formanceService.WithdrawMoney(r.Context(), userID, req.Amount)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Withdrawal failed: %v", err))
		return
	}

	response := map[string]interface{}{
		"message":        "Withdrawal successful",
		"transaction_id": transactionID,
		"amount":         req.Amount,
		"status":         "completed",
	}

	writeJSONResponse(w, http.StatusOK, response)
}