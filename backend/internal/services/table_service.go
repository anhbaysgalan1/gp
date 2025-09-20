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

// TableService provides simple GORM-based table operations
type TableService struct {
	db *database.DB
}

// NewTableService creates a new table service
func NewTableService(db *database.DB) *TableService {
	return &TableService{db: db}
}

// CreateTable creates a new poker table using direct GORM operations
func (ts *TableService) CreateTable(ctx context.Context, name string, createdBy uuid.UUID) (*models.PokerTable, error) {
	slog.Info("Creating table with direct GORM operations", "name", name, "created_by", createdBy)

	table := &models.PokerTable{
		Name:           name,
		TableType:      "cash",
		GameType:       "texas_holdem",
		MaxPlayers:     9,
		MinBuyIn:       100,   // 100 MNT
		MaxBuyIn:       10000, // 10000 MNT
		SmallBlind:     50,    // 50 MNT
		BigBlind:       100,   // 100 MNT
		IsPrivate:      false,
		Status:         "waiting",
		CurrentPlayers: 0,
		CreatedBy:      createdBy,
	}

	// Create table in database
	if err := ts.db.WithContext(ctx).Create(table).Error; err != nil {
		slog.Error("Failed to create table", "name", name, "error", err)
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	slog.Info("Table created successfully", "table_id", table.ID, "name", name)
	return table, nil
}

// GetTableByID retrieves a table by ID using direct GORM operations
func (ts *TableService) GetTableByID(ctx context.Context, id uuid.UUID) (*models.PokerTable, error) {
	var table models.PokerTable

	if err := ts.db.WithContext(ctx).First(&table, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("table not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get table: %w", err)
	}

	return &table, nil
}

// GetTableByName retrieves a table by name using direct GORM operations
func (ts *TableService) GetTableByName(ctx context.Context, name string) (*models.PokerTable, error) {
	var table models.PokerTable

	if err := ts.db.WithContext(ctx).First(&table, "name = ?", name).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("table not found: %s", name)
		}
		return nil, fmt.Errorf("failed to get table: %w", err)
	}

	return &table, nil
}

// UpdateTableStatus updates the table status
func (ts *TableService) UpdateTableStatus(ctx context.Context, id uuid.UUID, status string) error {
	result := ts.db.WithContext(ctx).Model(&models.PokerTable{}).Where("id = ?", id).Update("status", status)
	if result.Error != nil {
		return fmt.Errorf("failed to update table status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("table not found: %s", id)
	}
	return nil
}

// UpdatePlayerCount updates the current player count
func (ts *TableService) UpdatePlayerCount(ctx context.Context, id uuid.UUID, count int) error {
	result := ts.db.WithContext(ctx).Model(&models.PokerTable{}).Where("id = ?", id).Update("current_players", count)
	if result.Error != nil {
		return fmt.Errorf("failed to update player count: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("table not found: %s", id)
	}
	return nil
}

// ListTables returns all available tables
func (ts *TableService) ListTables(ctx context.Context) ([]*models.PokerTable, error) {
	var tables []*models.PokerTable

	if err := ts.db.WithContext(ctx).Find(&tables).Error; err != nil {
		return nil, fmt.Errorf("failed to list tables: %w", err)
	}

	return tables, nil
}

// TableExists checks if a table exists
func (ts *TableService) TableExists(ctx context.Context, id uuid.UUID) (bool, error) {
	var count int64
	if err := ts.db.WithContext(ctx).Model(&models.PokerTable{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check table existence: %w", err)
	}
	return count > 0, nil
}

// TableExistsByName checks if a table exists by name
func (ts *TableService) TableExistsByName(ctx context.Context, name string) (bool, error) {
	var count int64
	if err := ts.db.WithContext(ctx).Model(&models.PokerTable{}).Where("name = ?", name).Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check table existence: %w", err)
	}
	return count > 0, nil
}
