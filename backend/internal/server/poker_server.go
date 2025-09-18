package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/evanofslack/go-poker/internal/auth"
	"github.com/evanofslack/go-poker/internal/config"
	"github.com/evanofslack/go-poker/internal/database"
	"github.com/evanofslack/go-poker/internal/formance"
	"github.com/evanofslack/go-poker/internal/handlers"
	custommiddleware "github.com/evanofslack/go-poker/internal/middleware"
	"github.com/evanofslack/go-poker/internal/services"
	"github.com/evanofslack/go-poker/server"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type PokerServer struct {
	config          *config.Config
	db              *database.DB
	formanceService *formance.Service
	jwtManager      *auth.JWTManager
	authMiddleware  *auth.AuthMiddleware
	roleMiddleware  *auth.RoleMiddleware
	authService     *services.AuthService
	apiRateLimiter  *custommiddleware.RateLimiter
	authRateLimiter *custommiddleware.RateLimiter
	server          *http.Server
	hub             *server.Hub
}

func NewPokerServer() (*PokerServer, error) {
	// Load configuration
	cfg := config.Load()

	// Setup database
	db, err := database.NewConnection(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Run migrations
	if err := db.AutoMigrate(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Setup Formance service
	formanceService := formance.NewService(cfg)

	// Initialize Formance (create ledger, etc.)
	if err := formanceService.Initialize(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to initialize Formance: %w", err)
	}

	// Setup JWT manager
	jwtManager := auth.NewJWTManager(cfg.JWTSecret, "poker-platform")
	authMiddleware := auth.NewAuthMiddleware(jwtManager)
	roleMiddleware := auth.NewRoleMiddleware(db)

	// Setup services
	emailService := services.NewEmailService(cfg)
	authService := services.NewAuthService(db, jwtManager, emailService, formanceService)

	// Setup rate limiters
	apiRateLimiter := custommiddleware.NewAPIRateLimiter()
	authRateLimiter := custommiddleware.NewAuthRateLimiter()

	// Setup WebSocket hub
	hub, err := server.NewHub()
	if err != nil {
		return nil, fmt.Errorf("failed to create WebSocket hub: %w", err)
	}

	return &PokerServer{
		config:          cfg,
		db:              db,
		formanceService: formanceService,
		jwtManager:      jwtManager,
		authMiddleware:  authMiddleware,
		roleMiddleware:  roleMiddleware,
		authService:     authService,
		apiRateLimiter:  apiRateLimiter,
		authRateLimiter: authRateLimiter,
		hub:             hub,
	}, nil
}

func (s *PokerServer) Start() error {
	// Setup router
	router := s.setupRouter()

	// Create HTTP server
	s.server = &http.Server{
		Addr:    ":" + s.config.Port,
		Handler: router,
	}

	// Start WebSocket hub
	go s.hub.Run()

	// Start server in goroutine
	go func() {
		slog.Info("Starting poker server", "port", s.config.Port)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server failed to start", "error", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down server...")
	return s.Shutdown()
}

func (s *PokerServer) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := s.server.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
	}

	// Close database connection
	if err := s.db.Close(); err != nil {
		slog.Error("Failed to close database connection", "error", err)
	}

	// Close rate limiters
	s.apiRateLimiter.Close()
	s.authRateLimiter.Close()

	slog.Info("Server shutdown complete")
	return nil
}

func (s *PokerServer) setupRouter() chi.Router {
	r := chi.NewRouter()

	// Basic middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(auth.SecurityHeaders)
	r.Use(s.apiRateLimiter.RateLimit) // Apply global rate limiting

	// CORS middleware
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"}, // Configure properly for production
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// WebSocket endpoint
	r.Get("/ws", s.serveWebSocket)

	// TODO: Add Swagger documentation

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Create auth handler
		authHandler := handlers.NewAuthHandler(s.authService)

		// Public auth routes with stricter rate limiting
		r.Group(func(r chi.Router) {
			r.Use(s.authRateLimiter.RateLimit) // Apply auth-specific rate limiting
			r.Mount("/auth", authHandler.Routes())
		})

		// Protected routes group
		r.Group(func(r chi.Router) {
			r.Use(s.authMiddleware.RequireAuth)

			// Protected auth routes under /user (different path to avoid conflicts)
			r.Mount("/user", authHandler.ProtectedRoutes())

			// Balance management routes
			balanceHandler := handlers.NewBalanceHandler(s.formanceService, s.db.DB)
			r.Mount("/balance", balanceHandler.Routes())

			// Table management routes
			tableHandler := handlers.NewTableHandler(s.db, s.formanceService)
			r.Mount("/tables", tableHandler.Routes())

			// Tournament management routes
			tournamentHandler := handlers.NewTournamentHandler(s.db, s.formanceService)
			r.Mount("/tournaments", tournamentHandler.Routes())

			// Admin routes (role-based authorization)
			adminHandler := handlers.NewAdminHandler(s.db, s.formanceService)
			r.Mount("/admin", adminHandler.Routes(s.roleMiddleware))

			// TODO: Add leaderboard routes
		})

		// Optional auth routes (can be accessed with or without auth)
		r.Group(func(r chi.Router) {
			r.Use(s.authMiddleware.OptionalAuth)

			// TODO: Add public table listing
			// TODO: Add public leaderboards
		})
	})

	return r
}

// serveWebSocket handles WebSocket upgrade with authentication
func (s *PokerServer) serveWebSocket(w http.ResponseWriter, r *http.Request) {
	// Extract JWT token from query parameter or Authorization header
	var token string

	// First try Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		token = s.jwtManager.ExtractTokenFromBearer(authHeader)
	}

	// If no header, try query parameter (for WebSocket clients that can't set headers)
	if token == "" {
		token = r.URL.Query().Get("token")
	}

	if token == "" {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Validate JWT token
	claims, err := s.jwtManager.ValidateToken(token)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// Create WebSocket connection with authenticated user info
	server.ServeWsWithAuth(s.hub, w, r, claims.UserID, claims.Username, s.formanceService, s.db.DB)
}