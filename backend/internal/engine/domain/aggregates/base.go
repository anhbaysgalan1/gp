package aggregates

import (
	"github.com/anhbaysgalan1/gp/internal/engine/domain/events"
	"github.com/google/uuid"
)

// AggregateRoot represents the base aggregate root for event sourcing
type AggregateRoot struct {
	ID      uuid.UUID            `json:"id"`
	Version int64                `json:"version"`
	Changes []events.DomainEvent `json:"-"`
}

// GetID returns the aggregate ID
func (ar *AggregateRoot) GetID() uuid.UUID {
	return ar.ID
}

// GetVersion returns the current version
func (ar *AggregateRoot) GetVersion() int64 {
	return ar.Version
}

// GetUncommittedChanges returns the uncommitted events
func (ar *AggregateRoot) GetUncommittedChanges() []events.DomainEvent {
	return ar.Changes
}

// MarkChangesAsCommitted clears the uncommitted changes
func (ar *AggregateRoot) MarkChangesAsCommitted() {
	ar.Changes = []events.DomainEvent{}
}

// ApplyChange applies an event to the aggregate and adds it to uncommitted changes
func (ar *AggregateRoot) ApplyChange(event events.DomainEvent) {
	ar.Changes = append(ar.Changes, event)
	ar.Version++
}

// LoadFromHistory loads the aggregate state from a series of events
func (ar *AggregateRoot) LoadFromHistory(events []events.DomainEvent) {
	for _, event := range events {
		ar.Version = event.GetVersion()
	}
}

// Aggregate interface that all aggregates must implement
type Aggregate interface {
	GetID() uuid.UUID
	GetVersion() int64
	GetUncommittedChanges() []events.DomainEvent
	MarkChangesAsCommitted()
	ApplyChange(event events.DomainEvent)
	LoadFromHistory(events []events.DomainEvent)
}
