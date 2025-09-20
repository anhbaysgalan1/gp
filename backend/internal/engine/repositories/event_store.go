package repositories

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/anhbaysgalan1/gp/internal/engine/domain/events"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// EventModel represents an event record in the database
type EventModel struct {
	ID          uuid.UUID     `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	EventType   string        `gorm:"not null;index"`
	AggregateID uuid.UUID     `gorm:"type:uuid;not null;index"`
	Version     int64         `gorm:"not null"`
	UserID      *uuid.UUID    `gorm:"type:uuid;index"`
	Data        EventData     `gorm:"type:jsonb;not null"`
	Metadata    EventMetadata `gorm:"type:jsonb"`
	Timestamp   time.Time     `gorm:"not null;default:CURRENT_TIMESTAMP"`
	CreatedAt   time.Time     `gorm:"autoCreateTime"`
}

// EventData is a custom type for JSON data
type EventData map[string]interface{}

// EventMetadata is a custom type for JSON metadata
type EventMetadata map[string]interface{}

// Implement sql.Scanner and driver.Valuer for EventData
func (ed *EventData) Scan(value interface{}) error {
	if value == nil {
		*ed = EventData{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to scan EventData")
	}

	return json.Unmarshal(bytes, ed)
}

func (ed EventData) Value() (driver.Value, error) {
	if ed == nil {
		return nil, nil
	}
	return json.Marshal(ed)
}

// Implement sql.Scanner and driver.Valuer for EventMetadata
func (em *EventMetadata) Scan(value interface{}) error {
	if value == nil {
		*em = EventMetadata{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to scan EventMetadata")
	}

	return json.Unmarshal(bytes, em)
}

func (em EventMetadata) Value() (driver.Value, error) {
	if em == nil {
		return nil, nil
	}
	return json.Marshal(em)
}

// PostgreSQLEventStore implements the EventStore interface using PostgreSQL
type PostgreSQLEventStore struct {
	db *gorm.DB
}

// NewPostgreSQLEventStore creates a new PostgreSQL event store
func NewPostgreSQLEventStore(db *gorm.DB) *PostgreSQLEventStore {
	return &PostgreSQLEventStore{db: db}
}

// Migrate creates the necessary database tables
func (es *PostgreSQLEventStore) Migrate() error {
	return es.db.AutoMigrate(&EventModel{})
}

// SaveEvents saves a batch of events to the database
func (es *PostgreSQLEventStore) SaveEvents(ctx context.Context, aggregateID uuid.UUID, domainEvents []events.DomainEvent, expectedVersion int64) error {
	if len(domainEvents) == 0 {
		return nil
	}

	return es.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Check for concurrency conflicts
		var currentVersion int64
		result := tx.Model(&EventModel{}).
			Where("aggregate_id = ?", aggregateID).
			Select("COALESCE(MAX(version), 0)").
			Scan(&currentVersion)

		if result.Error != nil {
			return fmt.Errorf("failed to get current version: %w", result.Error)
		}

		if currentVersion != expectedVersion {
			return fmt.Errorf("concurrency conflict: expected version %d, but current version is %d", expectedVersion, currentVersion)
		}

		// Convert domain events to event models
		eventModels := make([]*EventModel, len(domainEvents))
		for i, domainEvent := range domainEvents {
			eventData, err := es.serializeEvent(domainEvent)
			if err != nil {
				return fmt.Errorf("failed to serialize event: %w", err)
			}

			eventModels[i] = &EventModel{
				EventType:   string(domainEvent.GetEventType()),
				AggregateID: domainEvent.GetAggregateID(),
				Version:     domainEvent.GetVersion(),
				UserID:      domainEvent.GetUserID(),
				Data:        eventData,
				Metadata:    EventMetadata{},
				Timestamp:   domainEvent.GetTimestamp(),
			}
		}

		// Save all events in a single batch
		if err := tx.Create(eventModels).Error; err != nil {
			return fmt.Errorf("failed to save events: %w", err)
		}

		return nil
	})
}

// GetEvents retrieves all events for an aggregate
func (es *PostgreSQLEventStore) GetEvents(ctx context.Context, aggregateID uuid.UUID) ([]events.DomainEvent, error) {
	var eventModels []*EventModel
	err := es.db.WithContext(ctx).
		Where("aggregate_id = ?", aggregateID).
		Order("version ASC").
		Find(&eventModels).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}

	domainEvents := make([]events.DomainEvent, len(eventModels))
	for i, eventModel := range eventModels {
		domainEvent, err := es.deserializeEvent(eventModel)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize event: %w", err)
		}
		domainEvents[i] = domainEvent
	}

	return domainEvents, nil
}

// GetEventsFromVersion retrieves events for an aggregate starting from a specific version
func (es *PostgreSQLEventStore) GetEventsFromVersion(ctx context.Context, aggregateID uuid.UUID, version int64) ([]events.DomainEvent, error) {
	var eventModels []*EventModel
	err := es.db.WithContext(ctx).
		Where("aggregate_id = ? AND version > ?", aggregateID, version).
		Order("version ASC").
		Find(&eventModels).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}

	domainEvents := make([]events.DomainEvent, len(eventModels))
	for i, eventModel := range eventModels {
		domainEvent, err := es.deserializeEvent(eventModel)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize event: %w", err)
		}
		domainEvents[i] = domainEvent
	}

	return domainEvents, nil
}

// serializeEvent converts a domain event to event data
func (es *PostgreSQLEventStore) serializeEvent(domainEvent events.DomainEvent) (EventData, error) {
	bytes, err := json.Marshal(domainEvent)
	if err != nil {
		return nil, err
	}

	var eventData EventData
	err = json.Unmarshal(bytes, &eventData)
	return eventData, err
}

// deserializeEvent converts event data back to a domain event
func (es *PostgreSQLEventStore) deserializeEvent(eventModel *EventModel) (events.DomainEvent, error) {
	eventType := events.EventType(eventModel.EventType)

	// Convert EventData back to JSON bytes
	bytes, err := json.Marshal(eventModel.Data)
	if err != nil {
		return nil, err
	}

	// Create the appropriate event type based on the event type
	switch eventType {
	case events.TableCreatedEvent:
		var event events.TableCreated
		err = json.Unmarshal(bytes, &event)
		return &event, err

	case events.PlayerJoinedEvent:
		var event events.PlayerJoined
		err = json.Unmarshal(bytes, &event)
		return &event, err

	case events.PlayerLeftEvent:
		var event events.PlayerLeft
		err = json.Unmarshal(bytes, &event)
		return &event, err

	case events.PlayerSeatedEvent:
		var event events.PlayerSeated
		err = json.Unmarshal(bytes, &event)
		return &event, err

	case events.HandStartedEvent:
		var event events.HandStarted
		err = json.Unmarshal(bytes, &event)
		return &event, err

	case events.CardsDealtEvent:
		var event events.CardsDealt
		err = json.Unmarshal(bytes, &event)
		return &event, err

	case events.CommunityCardsDealtEvent:
		var event events.CommunityCardsDealt
		err = json.Unmarshal(bytes, &event)
		return &event, err

	case events.PlayerBetEvent, events.PlayerCallEvent, events.PlayerRaiseEvent,
		events.PlayerCheckEvent, events.PlayerFoldEvent, events.PlayerAllInEvent:
		var event events.PlayerAction
		err = json.Unmarshal(bytes, &event)
		return &event, err

	case events.HandEndedEvent:
		var event events.HandEnded
		err = json.Unmarshal(bytes, &event)
		return &event, err

	case events.BuyInEvent:
		var event events.BuyIn
		err = json.Unmarshal(bytes, &event)
		return &event, err

	case events.CashOutEvent:
		var event events.CashOut
		err = json.Unmarshal(bytes, &event)
		return &event, err

	case events.WinningsDistributedEvent:
		var event events.WinningsDistributed
		err = json.Unmarshal(bytes, &event)
		return &event, err

	default:
		return nil, fmt.Errorf("unknown event type: %s", eventType)
	}
}
