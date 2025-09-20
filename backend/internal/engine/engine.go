package engine

import (
	"context"

	"github.com/anhbaysgalan1/gp/internal/application/dto"
	"github.com/anhbaysgalan1/gp/internal/engine/domain/aggregates"
	"github.com/anhbaysgalan1/gp/internal/engine/domain/events"
	"github.com/anhbaysgalan1/gp/internal/engine/domain/table"
	"github.com/google/uuid"
)

// PokerEngine interface defines the main poker engine operations
type PokerEngine interface {
	// Table management
	CreateTable(ctx context.Context, cmd dto.CreateTableCommand) (*table.Table, error)
	GetTable(ctx context.Context, tableID uuid.UUID) (*table.Table, error)
	ListTables(ctx context.Context) ([]*table.TableInfo, error)
	DeleteTable(ctx context.Context, tableID uuid.UUID) error

	// Player management
	JoinTable(ctx context.Context, cmd dto.JoinTableCommand) error
	LeaveTable(ctx context.Context, cmd dto.LeaveTableCommand) error
	SeatPlayer(ctx context.Context, cmd dto.SeatPlayerCommand) error

	// Game actions
	StartHand(ctx context.Context, tableID uuid.UUID) error
	PlayerAction(ctx context.Context, cmd dto.PlayerActionCommand) error

	// Queries
	GetGameState(ctx context.Context, tableID uuid.UUID) (*dto.GameStateView, error)
	GetHandHistory(ctx context.Context, tableID uuid.UUID, limit, offset int) ([]*dto.HandHistory, error)
}

// EventStore interface for persisting events
type EventStore interface {
	SaveEvents(ctx context.Context, aggregateID uuid.UUID, events []events.DomainEvent, expectedVersion int64) error
	GetEvents(ctx context.Context, aggregateID uuid.UUID) ([]events.DomainEvent, error)
	GetEventsFromVersion(ctx context.Context, aggregateID uuid.UUID, version int64) ([]events.DomainEvent, error)
}

// Repository interface for aggregate persistence
type Repository interface {
	Save(ctx context.Context, aggregate aggregates.Aggregate) error
	GetByID(ctx context.Context, id uuid.UUID) (aggregates.Aggregate, error)
}
