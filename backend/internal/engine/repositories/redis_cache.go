package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/anhbaysgalan1/gp/internal/application/dto"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

// RedisCache provides caching capabilities for the poker engine
type RedisCache struct {
	client *redis.Client
}

// NewRedisCache creates a new Redis cache instance
func NewRedisCache(client *redis.Client) *RedisCache {
	return &RedisCache{
		client: client,
	}
}

const (
	// Cache key prefixes
	gameStateCachePrefix = "game_state:"
	playerSessionPrefix  = "player_session:"
	tableInfoPrefix      = "table_info:"
	handHistoryPrefix    = "hand_history:"

	// Cache TTL durations
	gameStateTTL     = 1 * time.Hour
	playerSessionTTL = 24 * time.Hour
	tableInfoTTL     = 6 * time.Hour
	handHistoryTTL   = 48 * time.Hour
)

// Game State Caching

// SetGameState caches the current game state for a table
func (rc *RedisCache) SetGameState(ctx context.Context, tableID uuid.UUID, gameState *dto.GameStateView) error {
	key := gameStateCachePrefix + tableID.String()

	data, err := json.Marshal(gameState)
	if err != nil {
		return fmt.Errorf("failed to marshal game state: %w", err)
	}

	err = rc.client.Set(ctx, key, data, gameStateTTL).Err()
	if err != nil {
		return fmt.Errorf("failed to cache game state: %w", err)
	}

	return nil
}

// GetGameState retrieves cached game state for a table
func (rc *RedisCache) GetGameState(ctx context.Context, tableID uuid.UUID) (*dto.GameStateView, error) {
	key := gameStateCachePrefix + tableID.String()

	data, err := rc.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, fmt.Errorf("failed to get cached game state: %w", err)
	}

	var gameState dto.GameStateView
	err = json.Unmarshal([]byte(data), &gameState)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal game state: %w", err)
	}

	return &gameState, nil
}

// InvalidateGameState removes cached game state for a table
func (rc *RedisCache) InvalidateGameState(ctx context.Context, tableID uuid.UUID) error {
	key := gameStateCachePrefix + tableID.String()
	return rc.client.Del(ctx, key).Err()
}

// Player Session Caching

// PlayerSession represents a cached player session
type PlayerSession struct {
	PlayerID   uuid.UUID  `json:"player_id"`
	Username   string     `json:"username"`
	TableID    *uuid.UUID `json:"table_id,omitempty"`
	SeatNumber *int       `json:"seat_number,omitempty"`
	Chips      int64      `json:"chips"`
	LastSeen   time.Time  `json:"last_seen"`
	IsActive   bool       `json:"is_active"`
}

// SetPlayerSession caches player session information
func (rc *RedisCache) SetPlayerSession(ctx context.Context, playerID uuid.UUID, session *PlayerSession) error {
	key := playerSessionPrefix + playerID.String()

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal player session: %w", err)
	}

	err = rc.client.Set(ctx, key, data, playerSessionTTL).Err()
	if err != nil {
		return fmt.Errorf("failed to cache player session: %w", err)
	}

	return nil
}

// GetPlayerSession retrieves cached player session
func (rc *RedisCache) GetPlayerSession(ctx context.Context, playerID uuid.UUID) (*PlayerSession, error) {
	key := playerSessionPrefix + playerID.String()

	data, err := rc.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, fmt.Errorf("failed to get cached player session: %w", err)
	}

	var session PlayerSession
	err = json.Unmarshal([]byte(data), &session)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal player session: %w", err)
	}

	return &session, nil
}

// UpdatePlayerLastSeen updates the last seen timestamp for a player
func (rc *RedisCache) UpdatePlayerLastSeen(ctx context.Context, playerID uuid.UUID) error {
	session, err := rc.GetPlayerSession(ctx, playerID)
	if err != nil {
		return err
	}
	if session == nil {
		return nil // No session to update
	}

	session.LastSeen = time.Now()
	return rc.SetPlayerSession(ctx, playerID, session)
}

// InvalidatePlayerSession removes cached player session
func (rc *RedisCache) InvalidatePlayerSession(ctx context.Context, playerID uuid.UUID) error {
	key := playerSessionPrefix + playerID.String()
	return rc.client.Del(ctx, key).Err()
}

// Hand History Caching

// CacheHandHistory caches hand history for quick retrieval
func (rc *RedisCache) CacheHandHistory(ctx context.Context, tableID uuid.UUID, handHistories []*dto.HandHistory) error {
	key := handHistoryPrefix + tableID.String()

	data, err := json.Marshal(handHistories)
	if err != nil {
		return fmt.Errorf("failed to marshal hand history: %w", err)
	}

	err = rc.client.Set(ctx, key, data, handHistoryTTL).Err()
	if err != nil {
		return fmt.Errorf("failed to cache hand history: %w", err)
	}

	return nil
}

// GetCachedHandHistory retrieves cached hand history
func (rc *RedisCache) GetCachedHandHistory(ctx context.Context, tableID uuid.UUID) ([]*dto.HandHistory, error) {
	key := handHistoryPrefix + tableID.String()

	data, err := rc.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, fmt.Errorf("failed to get cached hand history: %w", err)
	}

	var handHistories []*dto.HandHistory
	err = json.Unmarshal([]byte(data), &handHistories)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal hand history: %w", err)
	}

	return handHistories, nil
}

// InvalidateHandHistory removes cached hand history for a table
func (rc *RedisCache) InvalidateHandHistory(ctx context.Context, tableID uuid.UUID) error {
	key := handHistoryPrefix + tableID.String()
	return rc.client.Del(ctx, key).Err()
}

// Table Info Caching

// CacheTableInfo caches table information
func (rc *RedisCache) CacheTableInfo(ctx context.Context, tableID uuid.UUID, tableInfo interface{}) error {
	key := tableInfoPrefix + tableID.String()

	data, err := json.Marshal(tableInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal table info: %w", err)
	}

	err = rc.client.Set(ctx, key, data, tableInfoTTL).Err()
	if err != nil {
		return fmt.Errorf("failed to cache table info: %w", err)
	}

	return nil
}

// GetCachedTableInfo retrieves cached table information
func (rc *RedisCache) GetCachedTableInfo(ctx context.Context, tableID uuid.UUID) (interface{}, error) {
	key := tableInfoPrefix + tableID.String()

	data, err := rc.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, fmt.Errorf("failed to get cached table info: %w", err)
	}

	var tableInfo interface{}
	err = json.Unmarshal([]byte(data), &tableInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal table info: %w", err)
	}

	return tableInfo, nil
}

// Utility Methods

// FlushTableCache removes all cached data for a specific table
func (rc *RedisCache) FlushTableCache(ctx context.Context, tableID uuid.UUID) error {
	keys := []string{
		gameStateCachePrefix + tableID.String(),
		tableInfoPrefix + tableID.String(),
		handHistoryPrefix + tableID.String(),
	}

	return rc.client.Del(ctx, keys...).Err()
}

// GetCacheStats returns basic cache statistics
func (rc *RedisCache) GetCacheStats(ctx context.Context) (map[string]interface{}, error) {
	info, err := rc.client.Info(ctx, "memory").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get Redis info: %w", err)
	}

	dbSize, err := rc.client.DBSize(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get Redis DB size: %w", err)
	}

	stats := map[string]interface{}{
		"db_size":     dbSize,
		"memory_info": info,
	}

	return stats, nil
}
