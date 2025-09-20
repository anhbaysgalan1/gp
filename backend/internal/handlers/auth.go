package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/anhbaysgalan1/gp/internal/auth"
	"github.com/anhbaysgalan1/gp/internal/models"
	"github.com/anhbaysgalan1/gp/internal/services"
	"github.com/anhbaysgalan1/gp/internal/validation"
	"github.com/go-chi/chi/v5"
)

type AuthHandler struct {
	authService *services.AuthService
}

func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

func (h *AuthHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Public routes (no auth required)
	r.Post("/register", h.Register)
	r.Post("/login", h.Login)
	r.Post("/verify-email", h.VerifyEmail)

	return r
}

func (h *AuthHandler) ProtectedRoutes() chi.Router {
	r := chi.NewRouter()

	// Protected routes (auth required)
	r.Get("/me", h.GetCurrentUser)
	r.Put("/profile", h.UpdateProfile)

	return r
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Validate request
	if err := validation.Validate(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	user, err := h.authService.RegisterUser(req)
	if err != nil {
		writeErrorResponse(w, http.StatusConflict, err.Error())
		return
	}

	// Create email verification token
	verification, err := h.authService.CreateEmailVerification(user.ID)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to create email verification")
		return
	}

	response := map[string]interface{}{
		"message":            "User registered successfully",
		"user":               user,
		"verification_token": verification.Token, // In production, send via email
	}

	writeJSONResponse(w, http.StatusCreated, response)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Validate request
	if err := validation.Validate(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	loginResponse, err := h.authService.LoginUser(req)
	if err != nil {
		writeErrorResponse(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	writeJSONResponse(w, http.StatusOK, loginResponse)
}

func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.Token == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Token is required")
		return
	}

	if err := h.authService.VerifyEmail(req.Token); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]string{
		"message": "Email verified successfully",
	})
}

func (h *AuthHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		writeErrorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	user, err := h.authService.GetUserByID(userID)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, "User not found")
		return
	}

	writeJSONResponse(w, http.StatusOK, user)
}

func (h *AuthHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		writeErrorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if err := h.authService.UpdateUserProfile(userID, updates); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]string{
		"message": "Profile updated successfully",
	})
}

// Helper functions
func writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	writeJSONResponse(w, statusCode, map[string]string{
		"error": message,
	})
}
