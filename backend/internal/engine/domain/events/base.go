package events

import (
	"time"

	"github.com/google/uuid"
)

// EventType represents the type of domain event
type EventType string

// Domain event types
const (
	// Table events
	TableCreatedEvent     EventType = "table.created"
	PlayerJoinedEvent     EventType = "player.joined"
	PlayerLeftEvent       EventType = "player.left"
	PlayerSeatedEvent     EventType = "player.seated"

	// Game events
	GameStartedEvent      EventType = "game.started"
	GameEndedEvent        EventType = "game.ended"
	HandStartedEvent      EventType = "hand.started"
	HandEndedEvent        EventType = "hand.ended"
	CardsDealtEvent       EventType = "cards.dealt"
	CommunityCardsDealtEvent EventType = "community_cards.dealt"

	// Player action events
	PlayerBetEvent        EventType = "player.bet"
	PlayerCallEvent       EventType = "player.call"
	PlayerRaiseEvent      EventType = "player.raise"
	PlayerCheckEvent      EventType = "player.check"
	PlayerFoldEvent       EventType = "player.fold"
	PlayerAllInEvent      EventType = "player.all_in"

	// Pot events
	PotCreatedEvent       EventType = "pot.created"
	PotWonEvent           EventType = "pot.won"
	SidePotCreatedEvent   EventType = "side_pot.created"

	// Financial events
	BuyInEvent            EventType = "financial.buy_in"
	CashOutEvent          EventType = "financial.cash_out"
	WinningsDistributedEvent EventType = "financial.winnings_distributed"
)

// BaseEvent contains common fields for all domain events
type BaseEvent struct {
	ID          uuid.UUID `json:"id"`
	EventType   EventType `json:"event_type"`
	AggregateID uuid.UUID `json:"aggregate_id"`
	Version     int64     `json:"version"`
	Timestamp   time.Time `json:"timestamp"`
	UserID      *uuid.UUID `json:"user_id,omitempty"`
}

// DomainEvent interface that all events must implement
type DomainEvent interface {
	GetID() uuid.UUID
	GetEventType() EventType
	GetAggregateID() uuid.UUID
	GetVersion() int64
	GetTimestamp() time.Time
	GetUserID() *uuid.UUID
}

// Implement DomainEvent interface for BaseEvent
func (e BaseEvent) GetID() uuid.UUID {
	return e.ID
}

func (e BaseEvent) GetEventType() EventType {
	return e.EventType
}

func (e BaseEvent) GetAggregateID() uuid.UUID {
	return e.AggregateID
}

func (e BaseEvent) GetVersion() int64 {
	return e.Version
}

func (e BaseEvent) GetTimestamp() time.Time {
	return e.Timestamp
}

func (e BaseEvent) GetUserID() *uuid.UUID {
	return e.UserID
}

// NewBaseEvent creates a new base event
func NewBaseEvent(eventType EventType, aggregateID uuid.UUID, version int64, userID *uuid.UUID) BaseEvent {
	return BaseEvent{
		ID:          uuid.New(),
		EventType:   eventType,
		AggregateID: aggregateID,
		Version:     version,
		Timestamp:   time.Now(),
		UserID:      userID,
	}
}