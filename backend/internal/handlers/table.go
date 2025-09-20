package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/anhbaysgalan1/gp/internal/auth"
	"github.com/anhbaysgalan1/gp/internal/database"
	"github.com/anhbaysgalan1/gp/internal/formance"
	"github.com/anhbaysgalan1/gp/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type TableHandler struct {
	db              *database.DB
	formanceService *formance.Service
}

func NewTableHandler(db *database.DB, formanceService *formance.Service) *TableHandler {
	return &TableHandler{
		db:              db,
		formanceService: formanceService,
	}
}

func (h *TableHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.ListTables)
	r.Post("/", h.CreateTable)
	r.Get("/{tableID}", h.GetTable)
	r.Put("/{tableID}", h.UpdateTable)
	r.Delete("/{tableID}", h.DeleteTable)
	r.Post("/{tableID}/join", h.JoinTable)
	r.Post("/{tableID}/leave", h.LeaveTable)

	return r
}

type CreateTableRequest struct {
	Name       string `json:"name" validate:"required,min=3,max=100"`
	TableType  string `json:"table_type" validate:"required,oneof=cash tournament"`
	GameType   string `json:"game_type" validate:"oneof=texas_holdem omaha stud"`
	MaxPlayers int    `json:"max_players" validate:"min=2,max=10"`
	MinBuyIn   int64  `json:"min_buy_in" validate:"required,gt=0"`
	MaxBuyIn   int64  `json:"max_buy_in" validate:"required,gt=0"`
	SmallBlind int64  `json:"small_blind" validate:"required,gt=0"`
	BigBlind   int64  `json:"big_blind" validate:"required,gt=0"`
	IsPrivate  bool   `json:"is_private"`
	Password   string `json:"password,omitempty"`
}

type UpdateTableRequest struct {
	Name       *string `json:"name,omitempty"`
	IsPrivate  *bool   `json:"is_private,omitempty"`
	Password   *string `json:"password,omitempty"`
	MaxBuyIn   *int64  `json:"max_buy_in,omitempty"`
	MinBuyIn   *int64  `json:"min_buy_in,omitempty"`
	SmallBlind *int64  `json:"small_blind,omitempty"`
	BigBlind   *int64  `json:"big_blind,omitempty"`
}

type JoinTableRequest struct {
	BuyInAmount int64  `json:"buy_in_amount" validate:"required,gt=0"`
	Password    string `json:"password,omitempty"`
}

// ListTables returns a list of available poker tables
func (h *TableHandler) ListTables(w http.ResponseWriter, r *http.Request) {
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

	tableType := r.URL.Query().Get("type") // cash or tournament
	status := r.URL.Query().Get("status")  // waiting, active, full, closed

	var tables []models.PokerTable
	query := h.db.Offset(offset).Limit(limit)

	// Apply filters
	if tableType != "" {
		query = query.Where("table_type = ?", tableType)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// Only show non-private tables unless user is authenticated
	userID, authenticated := auth.GetUserIDFromContext(r.Context())
	if !authenticated {
		query = query.Where("is_private = false")
	} else {
		// Show private tables created by user
		query = query.Where("is_private = false OR created_by = ?", userID)
	}

	if err := query.Find(&tables).Error; err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to fetch tables")
		return
	}

	// Get total count for pagination
	var total int64
	countQuery := h.db.Model(&models.PokerTable{})
	if tableType != "" {
		countQuery = countQuery.Where("table_type = ?", tableType)
	}
	if status != "" {
		countQuery = countQuery.Where("status = ?", status)
	}
	if !authenticated {
		countQuery = countQuery.Where("is_private = false")
	} else {
		countQuery = countQuery.Where("is_private = false OR created_by = ?", userID)
	}
	countQuery.Count(&total)

	response := map[string]interface{}{
		"tables": tables,
		"pagination": map[string]interface{}{
			"limit":  limit,
			"offset": offset,
			"total":  total,
		},
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// CreateTable creates a new poker table
func (h *TableHandler) CreateTable(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		writeErrorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req CreateTableRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Validation
	if req.Name == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Table name is required")
		return
	}

	if req.TableType == "" {
		req.TableType = "cash" // default
	}

	if req.GameType == "" {
		req.GameType = "texas_holdem" // default
	}

	if req.MaxPlayers == 0 {
		req.MaxPlayers = 9 // default
	}

	if req.MaxBuyIn <= req.MinBuyIn {
		writeErrorResponse(w, http.StatusBadRequest, "Max buy-in must be greater than min buy-in")
		return
	}

	if req.BigBlind <= req.SmallBlind {
		writeErrorResponse(w, http.StatusBadRequest, "Big blind must be greater than small blind")
		return
	}

	// Create table
	table := models.PokerTable{
		Name:       req.Name,
		TableType:  req.TableType,
		GameType:   req.GameType,
		MaxPlayers: req.MaxPlayers,
		MinBuyIn:   req.MinBuyIn,
		MaxBuyIn:   req.MaxBuyIn,
		SmallBlind: req.SmallBlind,
		BigBlind:   req.BigBlind,
		IsPrivate:  req.IsPrivate,
		Status:     "waiting",
		CreatedBy:  userID,
	}

	// Hash password if provided
	if req.Password != "" && req.IsPrivate {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to process password")
			return
		}
		hashedPasswordStr := string(hashedPassword)
		table.PasswordHash = &hashedPasswordStr
	}

	if err := h.db.Create(&table).Error; err != nil {
		if database.IsUniqueConstraintError(err) {
			writeErrorResponse(w, http.StatusConflict, "Table name already exists")
			return
		}
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to create table")
		return
	}

	writeJSONResponse(w, http.StatusCreated, table)
}

// GetTable returns details of a specific table
func (h *TableHandler) GetTable(w http.ResponseWriter, r *http.Request) {
	tableIDStr := chi.URLParam(r, "tableID")
	tableID, err := uuid.Parse(tableIDStr)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid table ID")
		return
	}

	var table models.PokerTable
	if err := h.db.First(&table, "id = ?", tableID).Error; err != nil {
		writeErrorResponse(w, http.StatusNotFound, "Table not found")
		return
	}

	// Hide password hash from response
	table.PasswordHash = nil

	writeJSONResponse(w, http.StatusOK, table)
}

// UpdateTable updates table settings (only by creator)
func (h *TableHandler) UpdateTable(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		writeErrorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	tableIDStr := chi.URLParam(r, "tableID")
	tableID, err := uuid.Parse(tableIDStr)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid table ID")
		return
	}

	var table models.PokerTable
	if err := h.db.First(&table, "id = ?", tableID).Error; err != nil {
		writeErrorResponse(w, http.StatusNotFound, "Table not found")
		return
	}

	// Check if user is the creator
	if table.CreatedBy != userID {
		writeErrorResponse(w, http.StatusForbidden, "Only table creator can update table settings")
		return
	}

	var req UpdateTableRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Update fields if provided
	updates := make(map[string]interface{})
	if req.Name != nil && *req.Name != "" {
		updates["name"] = *req.Name
	}
	if req.IsPrivate != nil {
		updates["is_private"] = *req.IsPrivate
	}
	if req.Password != nil {
		if *req.Password == "" {
			updates["password_hash"] = nil
		} else {
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
			if err != nil {
				writeErrorResponse(w, http.StatusInternalServerError, "Failed to process password")
				return
			}
			updates["password_hash"] = string(hashedPassword)
		}
	}
	if req.MaxBuyIn != nil && *req.MaxBuyIn > 0 {
		updates["max_buy_in"] = *req.MaxBuyIn
	}
	if req.MinBuyIn != nil && *req.MinBuyIn > 0 {
		updates["min_buy_in"] = *req.MinBuyIn
	}
	if req.SmallBlind != nil && *req.SmallBlind > 0 {
		updates["small_blind"] = *req.SmallBlind
	}
	if req.BigBlind != nil && *req.BigBlind > 0 {
		updates["big_blind"] = *req.BigBlind
	}

	if len(updates) == 0 {
		writeErrorResponse(w, http.StatusBadRequest, "No valid fields to update")
		return
	}

	if err := h.db.Model(&table).Updates(updates).Error; err != nil {
		if database.IsUniqueConstraintError(err) {
			writeErrorResponse(w, http.StatusConflict, "Table name already exists")
			return
		}
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to update table")
		return
	}

	// Fetch updated table
	h.db.First(&table, "id = ?", tableID)
	table.PasswordHash = nil // Hide password hash

	writeJSONResponse(w, http.StatusOK, table)
}

// DeleteTable deletes a table (only by creator)
func (h *TableHandler) DeleteTable(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		writeErrorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	tableIDStr := chi.URLParam(r, "tableID")
	tableID, err := uuid.Parse(tableIDStr)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid table ID")
		return
	}

	var table models.PokerTable
	if err := h.db.First(&table, "id = ?", tableID).Error; err != nil {
		writeErrorResponse(w, http.StatusNotFound, "Table not found")
		return
	}

	// Check if user is the creator
	if table.CreatedBy != userID {
		writeErrorResponse(w, http.StatusForbidden, "Only table creator can delete the table")
		return
	}

	// Check if table is active (has players)
	if table.CurrentPlayers > 0 {
		writeErrorResponse(w, http.StatusBadRequest, "Cannot delete table with active players")
		return
	}

	if err := h.db.Delete(&table).Error; err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to delete table")
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]string{
		"message": "Table deleted successfully",
	})
}

// JoinTable allows a user to join a poker table
func (h *TableHandler) JoinTable(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		writeErrorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	tableIDStr := chi.URLParam(r, "tableID")
	tableID, err := uuid.Parse(tableIDStr)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid table ID")
		return
	}

	var req JoinTableRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	var table models.PokerTable
	if err := h.db.First(&table, "id = ?", tableID).Error; err != nil {
		writeErrorResponse(w, http.StatusNotFound, "Table not found")
		return
	}

	// Check if table is full
	if table.CurrentPlayers >= table.MaxPlayers {
		writeErrorResponse(w, http.StatusBadRequest, "Table is full")
		return
	}

	// Check password for private tables
	if table.IsPrivate && table.PasswordHash != nil {
		err := bcrypt.CompareHashAndPassword([]byte(*table.PasswordHash), []byte(req.Password))
		if err != nil {
			writeErrorResponse(w, http.StatusUnauthorized, "Incorrect table password")
			return
		}
	}

	// Check buy-in amount
	if req.BuyInAmount < table.MinBuyIn || req.BuyInAmount > table.MaxBuyIn {
		writeErrorResponse(w, http.StatusBadRequest, "Buy-in amount must be between min and max buy-in")
		return
	}

	// Check if user has sufficient balance for buy-in
	balance, err := h.formanceService.GetUserBalance(r.Context(), userID, h.db.DB)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to check user balance")
		return
	}

	if balance.MainBalance < req.BuyInAmount {
		writeErrorResponse(w, http.StatusBadRequest, "Insufficient balance for buy-in")
		return
	}

	// Check if user already has an active session at this table
	var existingSession models.GameSession
	if err := h.db.Where("user_id = ? AND table_id = ? AND status = ?", userID, tableID, "active").First(&existingSession).Error; err == nil {
		writeErrorResponse(w, http.StatusBadRequest, "User already has an active session at this table")
		return
	}

	// Create game session
	session := models.GameSession{
		UserID:       userID,
		TableID:      tableID,
		BuyInAmount:  req.BuyInAmount,
		CurrentChips: req.BuyInAmount,
		Status:       "active",
	}

	if err := h.db.Create(&session).Error; err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to create game session")
		return
	}

	// Transfer funds from main account to game account using Formance
	transactionID, err := h.formanceService.TransferToGame(r.Context(), userID, req.BuyInAmount, session.ID)
	if err != nil {
		// Rollback session creation
		h.db.Delete(&session)
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to transfer funds: "+err.Error())
		return
	}

	// Update table player count (this should be done atomically in real implementation)
	if err := h.db.Model(&table).Update("current_players", table.CurrentPlayers+1).Error; err != nil {
		// Rollback fund transfer and session
		h.formanceService.TransferFromGame(r.Context(), userID, req.BuyInAmount, session.ID)
		h.db.Delete(&session)
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to join table")
		return
	}

	// Update table status if needed
	if table.CurrentPlayers+1 >= table.MaxPlayers {
		h.db.Model(&table).Update("status", "full")
	} else if table.Status == "waiting" {
		h.db.Model(&table).Update("status", "active")
	}

	response := map[string]interface{}{
		"message":        "Successfully joined table",
		"table_id":       tableID,
		"user_id":        userID,
		"buy_in_amount":  req.BuyInAmount,
		"session_id":     session.ID,
		"transaction_id": transactionID,
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// LeaveTable allows a user to leave a poker table
func (h *TableHandler) LeaveTable(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		writeErrorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	tableIDStr := chi.URLParam(r, "tableID")
	tableID, err := uuid.Parse(tableIDStr)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid table ID")
		return
	}

	// Find user's active session at this table
	var session models.GameSession
	if err := h.db.Where("user_id = ? AND table_id = ? AND status = ?", userID, tableID, "active").First(&session).Error; err != nil {
		if err.Error() == "record not found" {
			writeErrorResponse(w, http.StatusBadRequest, "User is not currently at this table")
		} else {
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to find user session")
		}
		return
	}

	var table models.PokerTable
	if err := h.db.First(&table, "id = ?", tableID).Error; err != nil {
		writeErrorResponse(w, http.StatusNotFound, "Table not found")
		return
	}

	if table.CurrentPlayers <= 0 {
		writeErrorResponse(w, http.StatusBadRequest, "No players at table")
		return
	}

	// Transfer remaining chips from game account back to main account
	var transactionID string
	if session.CurrentChips > 0 {
		tid, err := h.formanceService.TransferFromGame(r.Context(), userID, session.CurrentChips, session.ID)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to return funds: "+err.Error())
			return
		}
		transactionID = tid
	}

	// Mark session as finished
	session.Finish()
	if err := h.db.Save(&session).Error; err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to update session")
		return
	}

	// Update table player count
	newPlayerCount := table.CurrentPlayers - 1
	updates := map[string]interface{}{
		"current_players": newPlayerCount,
	}

	// Update status
	if newPlayerCount == 0 {
		updates["status"] = "waiting"
	} else if table.Status == "full" {
		updates["status"] = "active"
	}

	if err := h.db.Model(&table).Updates(updates).Error; err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to leave table")
		return
	}

	response := map[string]interface{}{
		"message":        "Successfully left table",
		"table_id":       tableID,
		"user_id":        userID,
		"session_id":     session.ID,
		"chips_returned": session.CurrentChips,
		"net_result":     session.GetNetResult(),
	}

	if transactionID != "" {
		response["transaction_id"] = transactionID
	}

	writeJSONResponse(w, http.StatusOK, response)
}
