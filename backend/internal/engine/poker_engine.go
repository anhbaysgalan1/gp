package engine

import (
	"context"
	"fmt"

	"github.com/anhbaysgalan1/gp/internal/application/dto"
	"github.com/anhbaysgalan1/gp/internal/application/handlers"
	"github.com/anhbaysgalan1/gp/internal/engine/domain/table"
	"github.com/anhbaysgalan1/gp/internal/engine/repositories"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// pokerEngineImpl implements the PokerEngine interface
type pokerEngineImpl struct {
	commandHandler *handlers.CommandHandler
	queryHandler   *handlers.QueryHandler
	eventStore     *repositories.PostgreSQLEventStore
	tableRepo      *repositories.TableRepository
	cache          *repositories.RedisCache
}

// NewPokerEngineImpl creates a new poker engine implementation without Redis
func NewPokerEngineImpl(db *gorm.DB) (PokerEngine, error) {
	return NewPokerEngineWithRedis(db, nil)
}

// NewPokerEngineWithRedis creates a new poker engine implementation with Redis caching
func NewPokerEngineWithRedis(db *gorm.DB, redisClient *redis.Client) (PokerEngine, error) {
	// Initialize event store
	eventStore := repositories.NewPostgreSQLEventStore(db)

	// Migrate event store tables
	if err := eventStore.Migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate event store: %w", err)
	}

	// Initialize repositories
	tableRepo := repositories.NewTableRepository(eventStore)

	// Initialize Redis cache if client is provided
	var cache *repositories.RedisCache
	if redisClient != nil {
		cache = repositories.NewRedisCache(redisClient)
	}

	// Initialize handlers
	commandHandler := handlers.NewCommandHandler(tableRepo)
	queryHandler := handlers.NewQueryHandler(tableRepo, eventStore)

	return &pokerEngineImpl{
		commandHandler: commandHandler,
		queryHandler:   queryHandler,
		eventStore:     eventStore,
		tableRepo:      tableRepo,
		cache:          cache,
	}, nil
}

// CreateTable creates a new poker table
func (pe *pokerEngineImpl) CreateTable(ctx context.Context, cmd dto.CreateTableCommand) (*table.Table, error) {
	return pe.commandHandler.CreateTable(ctx, cmd)
}

// GetTable retrieves a table by ID
func (pe *pokerEngineImpl) GetTable(ctx context.Context, tableID uuid.UUID) (*table.Table, error) {
	// Get the game state view and extract table info
	gameState, err := pe.queryHandler.GetTable(ctx, tableID)
	if err != nil {
		return nil, err
	}

	// Convert back to table (this is a simplified approach)
	// In a real implementation, you might want separate queries for table vs game state
	tableInfo := &table.Table{
		ID:         gameState.TableID,
		Name:       gameState.TableName,
		Status:     gameState.Status,
		SmallBlind: gameState.SmallBlind,
		BigBlind:   gameState.BigBlind,
	}

	return tableInfo, nil
}

// ListTables returns a list of all tables
func (pe *pokerEngineImpl) ListTables(ctx context.Context) ([]*table.TableInfo, error) {
	// This would require a read model/projection in a real implementation
	// For now, return empty list
	return []*table.TableInfo{}, nil
}

// DeleteTable removes a table
func (pe *pokerEngineImpl) DeleteTable(ctx context.Context, tableID uuid.UUID) error {
	// This would require implementing a DeleteTable command
	// For now, return not implemented error
	return fmt.Errorf("DeleteTable not yet implemented")
}

// JoinTable adds a player to a table
func (pe *pokerEngineImpl) JoinTable(ctx context.Context, cmd dto.JoinTableCommand) error {
	return pe.commandHandler.JoinTable(ctx, cmd)
}

// LeaveTable removes a player from a table
func (pe *pokerEngineImpl) LeaveTable(ctx context.Context, cmd dto.LeaveTableCommand) error {
	return pe.commandHandler.LeaveTable(ctx, cmd)
}

// SeatPlayer seats a player at a specific seat
func (pe *pokerEngineImpl) SeatPlayer(ctx context.Context, cmd dto.SeatPlayerCommand) error {
	return pe.commandHandler.SeatPlayer(ctx, cmd)
}

// StartHand starts a new hand at the table
func (pe *pokerEngineImpl) StartHand(ctx context.Context, tableID uuid.UUID) error {
	err := pe.commandHandler.StartHand(ctx, tableID)
	if err != nil {
		return err
	}

	// Invalidate cache for the table
	if pe.cache != nil {
		go func() {
			pe.cache.InvalidateGameState(context.Background(), tableID)
			pe.cache.InvalidateHandHistory(context.Background(), tableID)
		}()
	}

	return nil
}

// PlayerAction processes a player action
func (pe *pokerEngineImpl) PlayerAction(ctx context.Context, cmd dto.PlayerActionCommand) error {
	err := pe.commandHandler.PlayerAction(ctx, cmd)
	if err != nil {
		return err
	}

	// Invalidate cache for the table
	if pe.cache != nil {
		go func() {
			pe.cache.InvalidateGameState(context.Background(), cmd.TableID)
		}()
	}

	return nil
}

// GetGameState retrieves the current game state with caching
func (pe *pokerEngineImpl) GetGameState(ctx context.Context, tableID uuid.UUID) (*dto.GameStateView, error) {
	// Try cache first if available
	if pe.cache != nil {
		cached, err := pe.cache.GetGameState(ctx, tableID)
		if err == nil && cached != nil {
			return cached, nil
		}
		// Cache miss or error, continue to query handler
	}

	// Get from query handler
	gameState, err := pe.queryHandler.GetGameState(ctx, tableID)
	if err != nil {
		return nil, err
	}

	// Cache the result if cache is available
	if pe.cache != nil && gameState != nil {
		// Fire and forget caching
		go func() {
			if cacheErr := pe.cache.SetGameState(context.Background(), tableID, gameState); cacheErr != nil {
				// Log cache error but don't fail the operation
				fmt.Printf("Failed to cache game state: %v\n", cacheErr)
			}
		}()
	}

	return gameState, nil
}

// GetHandHistory retrieves hand history for a table with caching
func (pe *pokerEngineImpl) GetHandHistory(ctx context.Context, tableID uuid.UUID, limit, offset int) ([]*dto.HandHistory, error) {
	// For hand history, we only cache if limit/offset are default (first page)
	// This avoids cache complexity with different pagination parameters
	if pe.cache != nil && limit == 10 && offset == 0 {
		cached, err := pe.cache.GetCachedHandHistory(ctx, tableID)
		if err == nil && cached != nil {
			return cached, nil
		}
	}

	// Get from query handler
	handHistory, err := pe.queryHandler.GetHandHistory(ctx, tableID, limit, offset)
	if err != nil {
		return nil, err
	}

	// Cache the result if it's the first page and cache is available
	if pe.cache != nil && handHistory != nil && limit == 10 && offset == 0 {
		// Fire and forget caching
		go func() {
			if cacheErr := pe.cache.CacheHandHistory(context.Background(), tableID, handHistory); cacheErr != nil {
				// Log cache error but don't fail the operation
				fmt.Printf("Failed to cache hand history: %v\n", cacheErr)
			}
		}()
	}

	return handHistory, nil
}
