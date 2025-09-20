package repositories

import (
	"context"
	"fmt"

	"github.com/anhbaysgalan1/gp/internal/engine/domain/aggregates"
	"github.com/anhbaysgalan1/gp/internal/engine/domain/events"
	"github.com/anhbaysgalan1/gp/internal/engine/domain/table"
	"github.com/google/uuid"
)

// TableRepository handles persistence of table aggregates
type TableRepository struct {
	eventStore *PostgreSQLEventStore
}

// NewTableRepository creates a new table repository
func NewTableRepository(eventStore *PostgreSQLEventStore) *TableRepository {
	return &TableRepository{eventStore: eventStore}
}

// Save persists a table aggregate
func (tr *TableRepository) Save(ctx context.Context, aggregate aggregates.Aggregate) error {
	tableAggregate, ok := aggregate.(*aggregates.TableAggregate)
	if !ok {
		return fmt.Errorf("expected TableAggregate, got %T", aggregate)
	}

	uncommittedChanges := tableAggregate.GetUncommittedChanges()
	if len(uncommittedChanges) == 0 {
		return nil // No changes to save
	}

	// Save events to event store
	expectedVersion := tableAggregate.GetVersion() - int64(len(uncommittedChanges))
	err := tr.eventStore.SaveEvents(ctx, tableAggregate.GetID(), uncommittedChanges, expectedVersion)
	if err != nil {
		return fmt.Errorf("failed to save events: %w", err)
	}

	// Mark changes as committed
	tableAggregate.MarkChangesAsCommitted()

	return nil
}

// GetByID retrieves a table aggregate by ID
func (tr *TableRepository) GetByID(ctx context.Context, id uuid.UUID) (aggregates.Aggregate, error) {
	// Get all events for the aggregate
	domainEvents, err := tr.eventStore.GetEvents(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}

	if len(domainEvents) == 0 {
		return nil, fmt.Errorf("table not found: %s", id)
	}

	// Create a new table aggregate
	tableAggregate := &aggregates.TableAggregate{
		AggregateRoot: aggregates.AggregateRoot{
			ID:      id,
			Version: 0,
			Changes: make([]events.DomainEvent, 0),
		},
		Table: &table.Table{}, // Will be populated by events
	}

	// Rebuild state from events
	tableAggregate.LoadFromHistory(domainEvents)

	return tableAggregate, nil
}

// GetTableByName retrieves a table aggregate by name
func (tr *TableRepository) GetTableByName(ctx context.Context, name string) (*aggregates.TableAggregate, error) {
	// This is a simplified implementation
	// In a real system, you might want to maintain an index of table names to IDs
	// For now, we'll return an error indicating this needs to be implemented differently
	return nil, fmt.Errorf("GetTableByName not implemented - consider maintaining a name-to-ID index")
}

// ListTables returns a list of all table aggregates
func (tr *TableRepository) ListTables(ctx context.Context) ([]*aggregates.TableAggregate, error) {
	// This is a complex query that would require either:
	// 1. A separate read model/projection
	// 2. Querying all table creation events
	// For now, we'll return an error indicating this needs a different approach
	return nil, fmt.Errorf("ListTables not implemented - consider using read model projections")
}

// Exists checks if a table aggregate exists
func (tr *TableRepository) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	domainEvents, err := tr.eventStore.GetEvents(ctx, id)
	if err != nil {
		return false, fmt.Errorf("failed to check if table exists: %w", err)
	}

	return len(domainEvents) > 0, nil
}
