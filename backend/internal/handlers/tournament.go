package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/anhbaysgalan1/gp/internal/auth"
	"github.com/anhbaysgalan1/gp/internal/database"
	"github.com/anhbaysgalan1/gp/internal/formance"
	"github.com/anhbaysgalan1/gp/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type TournamentHandler struct {
	db              *database.DB
	formanceService *formance.Service
}

func NewTournamentHandler(db *database.DB, formanceService *formance.Service) *TournamentHandler {
	return &TournamentHandler{
		db:              db,
		formanceService: formanceService,
	}
}

func (h *TournamentHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.ListTournaments)
	r.Post("/", h.CreateTournament)
	r.Get("/{tournamentID}", h.GetTournament)
	r.Post("/{tournamentID}/register", h.RegisterForTournament)
	r.Delete("/{tournamentID}/unregister", h.UnregisterFromTournament)
	r.Get("/{tournamentID}/registrations", h.GetTournamentRegistrations)
	r.Post("/{tournamentID}/start", h.StartTournament)
	r.Post("/{tournamentID}/finish", h.FinishTournament)

	return r
}

// ListTournaments returns a list of tournaments
func (h *TournamentHandler) ListTournaments(w http.ResponseWriter, r *http.Request) {
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

	tournamentType := r.URL.Query().Get("type") // scheduled or sitng
	status := r.URL.Query().Get("status")       // registering, running, finished

	var tournaments []models.Tournament
	query := h.db.Offset(offset).Limit(limit).Order("created_at DESC")

	// Apply filters
	if tournamentType != "" {
		query = query.Where("tournament_type = ?", tournamentType)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Find(&tournaments).Error; err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to fetch tournaments")
		return
	}

	// Get total count for pagination
	var total int64
	countQuery := h.db.Model(&models.Tournament{})
	if tournamentType != "" {
		countQuery = countQuery.Where("tournament_type = ?", tournamentType)
	}
	if status != "" {
		countQuery = countQuery.Where("status = ?", status)
	}
	countQuery.Count(&total)

	response := map[string]interface{}{
		"tournaments": tournaments,
		"pagination": map[string]interface{}{
			"limit":  limit,
			"offset": offset,
			"total":  total,
		},
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// CreateTournament creates a new tournament
func (h *TournamentHandler) CreateTournament(w http.ResponseWriter, r *http.Request) {
	_, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		writeErrorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req models.CreateTournamentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Validation
	if req.Name == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Tournament name is required")
		return
	}

	if req.TournamentType == "" {
		req.TournamentType = "sitng" // default to sit-n-go
	}

	if req.BuyIn <= 0 {
		writeErrorResponse(w, http.StatusBadRequest, "Buy-in amount must be positive")
		return
	}

	if req.MaxPlayers < 2 {
		writeErrorResponse(w, http.StatusBadRequest, "Tournament must allow at least 2 players")
		return
	}

	// For scheduled tournaments, start time is required and must be in the future
	if req.TournamentType == "scheduled" {
		if req.StartTime == nil {
			writeErrorResponse(w, http.StatusBadRequest, "Start time is required for scheduled tournaments")
			return
		}
		if req.StartTime.Before(time.Now()) {
			writeErrorResponse(w, http.StatusBadRequest, "Start time must be in the future")
			return
		}
	}

	// Default blind structure for sit-n-go tournaments
	defaultBlindStructure := `[
		{"level": 1, "small_blind": 25, "big_blind": 50, "duration": 300},
		{"level": 2, "small_blind": 50, "big_blind": 100, "duration": 300},
		{"level": 3, "small_blind": 75, "big_blind": 150, "duration": 300},
		{"level": 4, "small_blind": 100, "big_blind": 200, "duration": 300},
		{"level": 5, "small_blind": 150, "big_blind": 300, "duration": 300}
	]`

	// Default payout structure (top 3 get paid)
	defaultPayoutStructure := `[
		{"position": 1, "percentage": 60},
		{"position": 2, "percentage": 30},
		{"position": 3, "percentage": 10}
	]`

	// Use provided structures or defaults
	blindStructure := req.BlindStructure
	if len(blindStructure) == 0 {
		blindStructure = json.RawMessage(defaultBlindStructure)
	}

	payoutStructure := req.PayoutStructure
	if len(payoutStructure) == 0 {
		payoutStructure = json.RawMessage(defaultPayoutStructure)
	}

	// Create tournament
	tournament := models.Tournament{
		Name:            req.Name,
		TournamentType:  req.TournamentType,
		BuyIn:           req.BuyIn,
		MaxPlayers:      req.MaxPlayers,
		StartTime:       req.StartTime,
		BlindStructure:  blindStructure,
		PayoutStructure: payoutStructure,
		Status:          "registering",
	}

	if err := h.db.Create(&tournament).Error; err != nil {
		if database.IsUniqueConstraintError(err) {
			writeErrorResponse(w, http.StatusConflict, "Tournament name already exists")
			return
		}
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to create tournament")
		return
	}

	writeJSONResponse(w, http.StatusCreated, tournament)
}

// GetTournament returns details of a specific tournament
func (h *TournamentHandler) GetTournament(w http.ResponseWriter, r *http.Request) {
	tournamentIDStr := chi.URLParam(r, "tournamentID")
	tournamentID, err := uuid.Parse(tournamentIDStr)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid tournament ID")
		return
	}

	var tournament models.Tournament
	if err := h.db.First(&tournament, "id = ?", tournamentID).Error; err != nil {
		if database.IsNotFoundError(err) {
			writeErrorResponse(w, http.StatusNotFound, "Tournament not found")
		} else {
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to fetch tournament")
		}
		return
	}

	writeJSONResponse(w, http.StatusOK, tournament)
}

// RegisterForTournament allows a user to register for a tournament
func (h *TournamentHandler) RegisterForTournament(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		writeErrorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	tournamentIDStr := chi.URLParam(r, "tournamentID")
	tournamentID, err := uuid.Parse(tournamentIDStr)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid tournament ID")
		return
	}

	// Get tournament details
	var tournament models.Tournament
	if err := h.db.First(&tournament, "id = ?", tournamentID).Error; err != nil {
		if database.IsNotFoundError(err) {
			writeErrorResponse(w, http.StatusNotFound, "Tournament not found")
		} else {
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to fetch tournament")
		}
		return
	}

	// Check tournament status
	if tournament.Status != "registering" {
		writeErrorResponse(w, http.StatusBadRequest, "Tournament registration is closed")
		return
	}

	// Check if tournament is full
	if tournament.RegisteredPlayers >= tournament.MaxPlayers {
		writeErrorResponse(w, http.StatusBadRequest, "Tournament is full")
		return
	}

	// Check if user is already registered
	var existingRegistration models.TournamentRegistration
	err = h.db.Where("tournament_id = ? AND user_id = ?", tournamentID, userID).First(&existingRegistration).Error
	if err == nil {
		writeErrorResponse(w, http.StatusBadRequest, "User is already registered for this tournament")
		return
	}
	if !database.IsNotFoundError(err) {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to check registration status")
		return
	}

	// Process buy-in payment
	transactionID, err := h.formanceService.ProcessTournamentBuyIn(r.Context(), userID, tournamentID, tournament.BuyIn)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Create registration record
	registration := models.TournamentRegistration{
		TournamentID:       tournamentID,
		UserID:             userID,
		BuyInTransactionID: &transactionID,
	}

	// Begin transaction
	tx := h.db.Begin()
	if tx.Error != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to start transaction")
		return
	}

	// Create registration
	if err := tx.Create(&registration).Error; err != nil {
		tx.Rollback()
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to create registration")
		return
	}

	// Update tournament registered players count and prize pool
	updates := map[string]interface{}{
		"registered_players": tournament.RegisteredPlayers + 1,
		"prize_pool":         tournament.PrizePool + tournament.BuyIn,
	}

	// For sit-n-go tournaments, start when full
	if tournament.TournamentType == "sitng" && tournament.RegisteredPlayers+1 >= tournament.MaxPlayers {
		updates["status"] = "running"
		updates["start_time"] = time.Now()
	}

	if err := tx.Model(&tournament).Updates(updates).Error; err != nil {
		tx.Rollback()
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to update tournament")
		return
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to complete registration")
		return
	}

	// Fetch updated registration with user details
	h.db.Preload("User").Preload("Tournament").First(&registration, "id = ?", registration.ID)

	response := map[string]interface{}{
		"message":        "Successfully registered for tournament",
		"registration":   registration,
		"transaction_id": transactionID,
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// UnregisterFromTournament allows a user to unregister from a tournament
func (h *TournamentHandler) UnregisterFromTournament(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		writeErrorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	tournamentIDStr := chi.URLParam(r, "tournamentID")
	tournamentID, err := uuid.Parse(tournamentIDStr)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid tournament ID")
		return
	}

	// Get tournament details
	var tournament models.Tournament
	if err := h.db.First(&tournament, "id = ?", tournamentID).Error; err != nil {
		if database.IsNotFoundError(err) {
			writeErrorResponse(w, http.StatusNotFound, "Tournament not found")
		} else {
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to fetch tournament")
		}
		return
	}

	// Check tournament status
	if tournament.Status != "registering" {
		writeErrorResponse(w, http.StatusBadRequest, "Cannot unregister from tournament that has started")
		return
	}

	// Find user's registration
	var registration models.TournamentRegistration
	err = h.db.Where("tournament_id = ? AND user_id = ?", tournamentID, userID).First(&registration).Error
	if database.IsNotFoundError(err) {
		writeErrorResponse(w, http.StatusNotFound, "User is not registered for this tournament")
		return
	}
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to find registration")
		return
	}

	// Begin transaction
	tx := h.db.Begin()
	if tx.Error != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to start transaction")
		return
	}

	// Delete registration
	if err := tx.Delete(&registration).Error; err != nil {
		tx.Rollback()
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to delete registration")
		return
	}

	// Update tournament counts
	updates := map[string]interface{}{
		"registered_players": tournament.RegisteredPlayers - 1,
		"prize_pool":         tournament.PrizePool - tournament.BuyIn,
	}

	if err := tx.Model(&tournament).Updates(updates).Error; err != nil {
		tx.Rollback()
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to update tournament")
		return
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to complete unregistration")
		return
	}

	// TODO: Process refund through Formance service

	response := map[string]interface{}{
		"message":       "Successfully unregistered from tournament",
		"tournament_id": tournamentID,
		"user_id":       userID,
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// GetTournamentRegistrations returns the list of registered players for a tournament
func (h *TournamentHandler) GetTournamentRegistrations(w http.ResponseWriter, r *http.Request) {
	tournamentIDStr := chi.URLParam(r, "tournamentID")
	tournamentID, err := uuid.Parse(tournamentIDStr)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid tournament ID")
		return
	}

	// Check if tournament exists
	var tournament models.Tournament
	if err := h.db.First(&tournament, "id = ?", tournamentID).Error; err != nil {
		if database.IsNotFoundError(err) {
			writeErrorResponse(w, http.StatusNotFound, "Tournament not found")
		} else {
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to fetch tournament")
		}
		return
	}

	// Get registrations with user details
	var registrations []models.TournamentRegistration
	if err := h.db.Preload("User").Where("tournament_id = ?", tournamentID).Order("registered_at ASC").Find(&registrations).Error; err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to fetch registrations")
		return
	}

	response := map[string]interface{}{
		"tournament":    tournament,
		"registrations": registrations,
		"count":         len(registrations),
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// StartTournament manually starts a scheduled tournament
func (h *TournamentHandler) StartTournament(w http.ResponseWriter, r *http.Request) {
	_, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		writeErrorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	tournamentIDStr := chi.URLParam(r, "tournamentID")
	tournamentID, err := uuid.Parse(tournamentIDStr)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid tournament ID")
		return
	}

	var tournament models.Tournament
	if err := h.db.First(&tournament, "id = ?", tournamentID).Error; err != nil {
		if database.IsNotFoundError(err) {
			writeErrorResponse(w, http.StatusNotFound, "Tournament not found")
		} else {
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to fetch tournament")
		}
		return
	}

	// Check if tournament can be started
	if tournament.Status != "registering" {
		writeErrorResponse(w, http.StatusBadRequest, "Tournament is not in registering state")
		return
	}

	if tournament.RegisteredPlayers < 2 {
		writeErrorResponse(w, http.StatusBadRequest, "Tournament needs at least 2 players to start")
		return
	}

	// TODO: Add authorization check - only tournament organizers or admins should be able to start tournaments

	// Start the tournament
	now := time.Now()
	updates := map[string]interface{}{
		"status":     "running",
		"start_time": now,
	}

	if err := h.db.Model(&tournament).Updates(updates).Error; err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to start tournament")
		return
	}

	// Fetch updated tournament
	h.db.First(&tournament, "id = ?", tournamentID)

	response := map[string]interface{}{
		"message":    "Tournament started successfully",
		"tournament": tournament,
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// FinishTournament finishes a tournament and distributes prizes
func (h *TournamentHandler) FinishTournament(w http.ResponseWriter, r *http.Request) {
	_, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		writeErrorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	tournamentIDStr := chi.URLParam(r, "tournamentID")
	tournamentID, err := uuid.Parse(tournamentIDStr)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid tournament ID")
		return
	}

	type FinishTournamentRequest struct {
		Results []struct {
			UserID      uuid.UUID `json:"user_id" validate:"required"`
			Position    int       `json:"position" validate:"required,gt=0"`
			PrizeAmount int64     `json:"prize_amount" validate:"gte=0"`
		} `json:"results" validate:"required,min=1"`
	}

	var req FinishTournamentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	var tournament models.Tournament
	if err := h.db.First(&tournament, "id = ?", tournamentID).Error; err != nil {
		if database.IsNotFoundError(err) {
			writeErrorResponse(w, http.StatusNotFound, "Tournament not found")
		} else {
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to fetch tournament")
		}
		return
	}

	// Check if tournament can be finished
	if tournament.Status != "running" {
		writeErrorResponse(w, http.StatusBadRequest, "Tournament is not running")
		return
	}

	// TODO: Add authorization check - only tournament organizers or game server should finish tournaments

	// Begin transaction
	tx := h.db.Begin()
	if tx.Error != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to start transaction")
		return
	}

	// Update registrations with final positions and prize amounts
	for _, result := range req.Results {
		updates := map[string]interface{}{
			"final_position": result.Position,
			"prize_amount":   result.PrizeAmount,
		}

		if err := tx.Model(&models.TournamentRegistration{}).
			Where("tournament_id = ? AND user_id = ?", tournamentID, result.UserID).
			Updates(updates).Error; err != nil {
			tx.Rollback()
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to update player results")
			return
		}

		// Distribute prizes if amount > 0
		if result.PrizeAmount > 0 {
			if _, err := h.formanceService.DistributeTournamentPrize(r.Context(), result.UserID, tournamentID, result.PrizeAmount); err != nil {
				tx.Rollback()
				writeErrorResponse(w, http.StatusInternalServerError, "Failed to distribute prize")
				return
			}
		}
	}

	// Update tournament status
	now := time.Now()
	tournamentUpdates := map[string]interface{}{
		"status":   "finished",
		"end_time": now,
	}

	if err := tx.Model(&tournament).Updates(tournamentUpdates).Error; err != nil {
		tx.Rollback()
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to finish tournament")
		return
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to complete tournament finish")
		return
	}

	// Fetch updated tournament with registrations
	h.db.Preload("TournamentRegistrations.User").First(&tournament, "id = ?", tournamentID)

	response := map[string]interface{}{
		"message":    "Tournament finished successfully",
		"tournament": tournament,
	}

	writeJSONResponse(w, http.StatusOK, response)
}
