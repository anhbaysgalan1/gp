package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/evanofslack/go-poker/internal/auth"
	"github.com/evanofslack/go-poker/internal/database"
	"github.com/evanofslack/go-poker/internal/formance"
	"github.com/evanofslack/go-poker/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type AdminHandler struct {
	db              *database.DB
	formanceService *formance.Service
}

func NewAdminHandler(db *database.DB, formanceService *formance.Service) *AdminHandler {
	return &AdminHandler{
		db:              db,
		formanceService: formanceService,
	}
}

func (h *AdminHandler) Routes(roleMiddleware *auth.RoleMiddleware) chi.Router {
	r := chi.NewRouter()

	// All admin routes require admin role
	r.Use(roleMiddleware.RequireAdmin)

	r.Get("/users", h.ListUsers)
	r.Put("/users/{userID}/role", h.UpdateUserRole)
	r.Delete("/users/{userID}", h.DeleteUser)
	r.Get("/stats", h.GetSystemStats)

	// Development only - balance management endpoints
	r.Post("/users/{userID}/deposit", h.DepositMoney)
	r.Post("/users/{userID}/withdraw", h.WithdrawMoney)

	return r
}

// ListUsers returns paginated list of all users (admin only)
func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
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

	var users []models.User
	if err := h.db.Offset(offset).Limit(limit).Find(&users).Error; err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to fetch users")
		return
	}

	var total int64
	h.db.Model(&models.User{}).Count(&total)

	response := map[string]interface{}{
		"users": users,
		"pagination": map[string]interface{}{
			"limit":  limit,
			"offset": offset,
			"total":  total,
		},
	}

	writeJSONResponse(w, http.StatusOK, response)
}

type UpdateUserRoleRequest struct {
	Role string `json:"role" validate:"required,oneof=player moderator admin"`
}

// UpdateUserRole updates a user's role (admin only)
func (h *AdminHandler) UpdateUserRole(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "userID")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	var req UpdateUserRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Validate role
	var newRole models.UserRole
	switch req.Role {
	case "player":
		newRole = models.UserRolePlayer
	case "moderator":
		newRole = models.UserRoleMod
	case "admin":
		newRole = models.UserRoleAdmin
	default:
		writeErrorResponse(w, http.StatusBadRequest, "Invalid role")
		return
	}

	// Update user role
	result := h.db.Model(&models.User{}).Where("id = ?", userID).Update("role", newRole)
	if result.Error != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to update user role")
		return
	}

	if result.RowsAffected == 0 {
		writeErrorResponse(w, http.StatusNotFound, "User not found")
		return
	}

	response := map[string]interface{}{
		"message": "User role updated successfully",
		"user_id": userID,
		"role":    newRole,
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// DeleteUser soft deletes a user (admin only)
func (h *AdminHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "userID")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	// Prevent admin from deleting themselves
	adminUserID, ok := auth.GetUserIDFromContext(r.Context())
	if ok && adminUserID == userID {
		writeErrorResponse(w, http.StatusBadRequest, "Cannot delete your own account")
		return
	}

	// Soft delete user
	result := h.db.Delete(&models.User{}, "id = ?", userID)
	if result.Error != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to delete user")
		return
	}

	if result.RowsAffected == 0 {
		writeErrorResponse(w, http.StatusNotFound, "User not found")
		return
	}

	response := map[string]interface{}{
		"message": "User deleted successfully",
		"user_id": userID,
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// GetSystemStats returns system statistics (admin only)
func (h *AdminHandler) GetSystemStats(w http.ResponseWriter, r *http.Request) {
	var stats struct {
		TotalUsers       int64 `json:"total_users"`
		VerifiedUsers    int64 `json:"verified_users"`
		TotalTables      int64 `json:"total_tables"`
		ActiveTables     int64 `json:"active_tables"`
		TotalTournaments int64 `json:"total_tournaments"`
	}

	// Get user statistics
	h.db.Model(&models.User{}).Count(&stats.TotalUsers)
	h.db.Model(&models.User{}).Where("is_verified = ?", true).Count(&stats.VerifiedUsers)

	// Get table statistics
	h.db.Model(&models.PokerTable{}).Count(&stats.TotalTables)
	h.db.Model(&models.PokerTable{}).Where("status IN ?", []string{"active", "full"}).Count(&stats.ActiveTables)

	// Get tournament statistics
	h.db.Model(&models.Tournament{}).Count(&stats.TotalTournaments)

	writeJSONResponse(w, http.StatusOK, stats)
}

// DepositMoneyRequest represents the request to deposit money to a user account
type DepositMoneyRequest struct {
	Amount int64 `json:"amount" validate:"required,gt=0"`
}

// DepositMoney adds money to a user's main account (development only)
func (h *AdminHandler) DepositMoney(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "userID")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	var req DepositMoneyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.Amount <= 0 {
		writeErrorResponse(w, http.StatusBadRequest, "Amount must be positive")
		return
	}

	// Check if user exists
	var user models.User
	if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
		writeErrorResponse(w, http.StatusNotFound, "User not found")
		return
	}

	// Create deposit transaction using Formance
	transactionID, err := h.formanceService.DepositMoney(r.Context(), userID, req.Amount)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to deposit money: "+err.Error())
		return
	}

	response := map[string]interface{}{
		"message":        "Money deposited successfully",
		"user_id":        userID,
		"amount":         req.Amount,
		"transaction_id": transactionID,
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// WithdrawMoneyRequest represents the request to withdraw money from a user account
type WithdrawMoneyRequest struct {
	Amount int64 `json:"amount" validate:"required,gt=0"`
}

// WithdrawMoney removes money from a user's main account (development only)
func (h *AdminHandler) WithdrawMoney(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "userID")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	var req WithdrawMoneyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.Amount <= 0 {
		writeErrorResponse(w, http.StatusBadRequest, "Amount must be positive")
		return
	}

	// Check if user exists
	var user models.User
	if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
		writeErrorResponse(w, http.StatusNotFound, "User not found")
		return
	}

	// Check if user has sufficient balance
	balance, err := h.formanceService.GetUserBalance(r.Context(), userID, h.db.DB)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to check user balance")
		return
	}

	if balance.MainBalance < req.Amount {
		writeErrorResponse(w, http.StatusBadRequest, "Insufficient balance for withdrawal")
		return
	}

	// Create withdrawal transaction using Formance
	transactionID, err := h.formanceService.WithdrawMoney(r.Context(), userID, req.Amount)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to withdraw money: "+err.Error())
		return
	}

	response := map[string]interface{}{
		"message":        "Money withdrawn successfully",
		"user_id":        userID,
		"amount":         req.Amount,
		"transaction_id": transactionID,
	}

	writeJSONResponse(w, http.StatusOK, response)
}