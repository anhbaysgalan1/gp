package services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/anhbaysgalan1/gp/internal/database"
	"github.com/anhbaysgalan1/gp/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GameSessionService provides secure game session operations for real money games
type GameSessionService struct {
	db *database.DB
}

// NewGameSessionService creates a new game session service
func NewGameSessionService(db *database.DB) *GameSessionService {
	return &GameSessionService{db: db}
}

// CreateSession creates a new game session for a real money table
func (gs *GameSessionService) CreateSession(ctx context.Context, userID, tableID uuid.UUID, buyInAmount int64, seatNumber int) (*models.GameSession, error) {
	slog.Info("Creating game session", "user_id", userID, "table_id", tableID, "buy_in", buyInAmount, "seat", seatNumber)

	session := &models.GameSession{
		UserID:       userID,
		TableID:      tableID,
		BuyInAmount:  buyInAmount,
		CurrentChips: buyInAmount, // Start with full buy-in amount
		Status:       models.GameSessionStatusActive,
		SeatNumber:   &seatNumber,
	}

	if err := gs.db.WithContext(ctx).Create(session).Error; err != nil {
		slog.Error("Failed to create game session", "user_id", userID, "table_id", tableID, "error", err)
		return nil, fmt.Errorf("failed to create game session: %w", err)
	}

	slog.Info("Game session created successfully", "session_id", session.ID, "user_id", userID, "table_id", tableID)
	return session, nil
}

// GetActiveSessionByUserAndTable retrieves an active session for a user at a specific table
func (gs *GameSessionService) GetActiveSessionByUserAndTable(ctx context.Context, userID, tableID uuid.UUID) (*models.GameSession, error) {
	var session models.GameSession

	err := gs.db.WithContext(ctx).Where("user_id = ? AND table_id = ? AND status = ?",
		userID, tableID, models.GameSessionStatusActive).First(&session).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // No active session found
		}
		return nil, fmt.Errorf("failed to get active session: %w", err)
	}

	return &session, nil
}

// GetSessionByID retrieves a session by ID
func (gs *GameSessionService) GetSessionByID(ctx context.Context, sessionID uuid.UUID) (*models.GameSession, error) {
	var session models.GameSession

	err := gs.db.WithContext(ctx).First(&session, "id = ?", sessionID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("session not found: %s", sessionID)
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return &session, nil
}

// UpdateChips updates the current chip count for a session
func (gs *GameSessionService) UpdateChips(ctx context.Context, sessionID uuid.UUID, newChipAmount int64) error {
	slog.Info("Updating session chips", "session_id", sessionID, "new_chips", newChipAmount)

	result := gs.db.WithContext(ctx).Model(&models.GameSession{}).
		Where("id = ? AND status = ?", sessionID, models.GameSessionStatusActive).
		Update("current_chips", newChipAmount)

	if result.Error != nil {
		return fmt.Errorf("failed to update session chips: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("no active session found to update: %s", sessionID)
	}

	slog.Info("Session chips updated successfully", "session_id", sessionID, "new_chips", newChipAmount)
	return nil
}

// FinishSession marks a session as finished and records final chip count
func (gs *GameSessionService) FinishSession(ctx context.Context, sessionID uuid.UUID, finalChips int64) error {
	slog.Info("Finishing game session", "session_id", sessionID, "final_chips", finalChips)

	// Update session with final status and chip count
	result := gs.db.WithContext(ctx).Model(&models.GameSession{}).
		Where("id = ? AND status = ?", sessionID, models.GameSessionStatusActive).
		Updates(map[string]interface{}{
			"current_chips": finalChips,
			"status":        models.GameSessionStatusFinished,
			"left_at":       gorm.Expr("CURRENT_TIMESTAMP"),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to finish session: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("no active session found to finish: %s", sessionID)
	}

	slog.Info("Session finished successfully", "session_id", sessionID, "final_chips", finalChips)
	return nil
}

// AbandonSession marks a session as abandoned (for unexpected disconnections)
func (gs *GameSessionService) AbandonSession(ctx context.Context, sessionID uuid.UUID) error {
	slog.Info("Abandoning game session", "session_id", sessionID)

	result := gs.db.WithContext(ctx).Model(&models.GameSession{}).
		Where("id = ? AND status = ?", sessionID, models.GameSessionStatusActive).
		Updates(map[string]interface{}{
			"status":  models.GameSessionStatusAbandoned,
			"left_at": gorm.Expr("CURRENT_TIMESTAMP"),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to abandon session: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("no active session found to abandon: %s", sessionID)
	}

	slog.Info("Session abandoned successfully", "session_id", sessionID)
	return nil
}

// GetSessionNetResult calculates the net result for a session
func (gs *GameSessionService) GetSessionNetResult(ctx context.Context, sessionID uuid.UUID) (int64, error) {
	var session models.GameSession

	err := gs.db.WithContext(ctx).Select("buy_in_amount", "current_chips").
		First(&session, "id = ?", sessionID).Error

	if err != nil {
		return 0, fmt.Errorf("failed to get session for net result: %w", err)
	}

	return session.CurrentChips - session.BuyInAmount, nil
}

// IsRealMoneySession checks if a session ID represents a real money session
func (gs *GameSessionService) IsRealMoneySession(ctx context.Context, sessionID uuid.UUID) (bool, error) {
	if sessionID == uuid.Nil {
		return false, nil // Nil session ID means practice/virtual game
	}

	var count int64
	err := gs.db.WithContext(ctx).Model(&models.GameSession{}).
		Where("id = ?", sessionID).Count(&count).Error

	if err != nil {
		return false, fmt.Errorf("failed to check session existence: %w", err)
	}

	return count > 0, nil
}
