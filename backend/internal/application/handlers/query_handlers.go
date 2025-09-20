package handlers

import (
	"context"
	"fmt"

	"github.com/anhbaysgalan1/gp/internal/application/dto"
	"github.com/anhbaysgalan1/gp/internal/engine/domain/aggregates"
	"github.com/anhbaysgalan1/gp/internal/engine/domain/events"
	"github.com/anhbaysgalan1/gp/internal/engine/domain/game"
	"github.com/anhbaysgalan1/gp/internal/engine/repositories"
	"github.com/google/uuid"
)

// QueryHandler handles all query operations for the poker engine
type QueryHandler struct {
	tableRepository *repositories.TableRepository
	eventStore      *repositories.PostgreSQLEventStore
}

// NewQueryHandler creates a new query handler
func NewQueryHandler(tableRepository *repositories.TableRepository, eventStore *repositories.PostgreSQLEventStore) *QueryHandler {
	return &QueryHandler{
		tableRepository: tableRepository,
		eventStore:      eventStore,
	}
}

// GetTable retrieves a table by ID
func (qh *QueryHandler) GetTable(ctx context.Context, tableID uuid.UUID) (*dto.GameStateView, error) {
	// Load table aggregate
	aggregate, err := qh.tableRepository.GetByID(ctx, tableID)
	if err != nil {
		return nil, fmt.Errorf("failed to get table: %w", err)
	}

	tableAggregate, ok := aggregate.(*aggregates.TableAggregate)
	if !ok {
		return nil, fmt.Errorf("expected TableAggregate, got %T", aggregate)
	}

	// Convert to view model
	return qh.convertToGameStateView(tableAggregate), nil
}

// GetGameState retrieves the current game state for a table
func (qh *QueryHandler) GetGameState(ctx context.Context, tableID uuid.UUID) (*dto.GameStateView, error) {
	return qh.GetTable(ctx, tableID)
}

// GetHandHistory retrieves hand history for a table
func (qh *QueryHandler) GetHandHistory(ctx context.Context, tableID uuid.UUID, limit, offset int) ([]*dto.HandHistory, error) {
	// Load events for the table
	events, err := qh.eventStore.GetEvents(ctx, tableID)
	if err != nil {
		return nil, fmt.Errorf("failed to load events: %w", err)
	}

	// Build hand history from events
	handHistories, err := qh.buildHandHistoryFromEvents(events)
	if err != nil {
		return nil, fmt.Errorf("failed to build hand history: %w", err)
	}

	// Apply pagination
	start := offset
	end := offset + limit
	if start >= len(handHistories) {
		return []*dto.HandHistory{}, nil
	}
	if end > len(handHistories) {
		end = len(handHistories)
	}

	return handHistories[start:end], nil
}

// convertToGameStateView converts a table aggregate to a game state view
func (qh *QueryHandler) convertToGameStateView(tableAggregate *aggregates.TableAggregate) *dto.GameStateView {
	table := tableAggregate.Table
	gameState := table.Game

	// Convert players
	players := make([]dto.PlayerView, len(gameState.Players))
	for i, player := range gameState.Players {
		players[i] = dto.PlayerView{
			ID:           player.ID,
			Username:     player.Username,
			SeatNumber:   player.SeatNumber,
			Chips:        player.Chips,
			CurrentBet:   player.CurrentBet,
			TotalBet:     player.TotalBet,
			IsActive:     player.IsActive,
			IsFolded:     player.IsFolded,
			IsAllIn:      player.IsAllIn,
			HasActed:     player.HasActed,
			IsDealer:     player.Position.IsDealer,
			IsSmallBlind: player.Position.IsSmallBlind,
			IsBigBlind:   player.Position.IsBigBlind,
			HoleCards:    qh.convertCards(player.HoleCards),
		}
	}

	// Convert community cards
	communityCards := qh.convertCards(gameState.CommunityCards)

	// Convert pots
	pots := make([]dto.PotView, len(gameState.Pots))
	for i, pot := range gameState.Pots {
		pots[i] = dto.PotView{
			ID:              pot.ID,
			Amount:          pot.Amount,
			EligiblePlayers: pot.EligiblePlayers,
			WinningPlayers:  pot.WinningPlayers,
			IsSidePot:       pot.IsSidePot,
		}
	}

	return &dto.GameStateView{
		TableID:        table.ID,
		TableName:      table.Name,
		Status:         table.Status,
		Stage:          gameState.Stage.String(),
		IsRunning:      gameState.IsRunning,
		HandID:         gameState.HandID,
		Players:        players,
		CommunityCards: communityCards,
		Pots:           pots,
		DealerSeat:     gameState.DealerSeat,
		ActionSeat:     gameState.ActionSeat,
		CurrentBet:     gameState.GetCurrentBet(),
		MinRaise:       gameState.MinRaise,
		SmallBlind:     gameState.SmallBlind,
		BigBlind:       gameState.BigBlind,
		HandNumber:     gameState.HandNumber,
	}
}

// convertCards converts domain cards to view cards
func (qh *QueryHandler) convertCards(cards []game.Card) []dto.CardView {
	if cards == nil {
		return nil
	}

	viewCards := make([]dto.CardView, len(cards))
	for i, card := range cards {
		viewCards[i] = dto.CardView{
			Suit:  card.Suit,
			Rank:  card.Rank,
			Value: card.Value,
		}
	}
	return viewCards
}

// buildHandHistoryFromEvents builds hand history records from a stream of domain events
func (qh *QueryHandler) buildHandHistoryFromEvents(domainEvents []events.DomainEvent) ([]*dto.HandHistory, error) {
	handHistories := make([]*dto.HandHistory, 0)
	currentHand := &dto.HandHistory{}
	handStarted := false

	for _, event := range domainEvents {
		switch e := event.(type) {
		case *events.HandStarted:
			// Start new hand
			if handStarted {
				// Complete previous hand if one was started
				handHistories = append(handHistories, currentHand)
			}
			currentHand = &dto.HandHistory{
				HandID:     e.HandID,
				TableID:    e.AggregateID,
				StartTime:  e.Timestamp.Format("2006-01-02T15:04:05Z"),
				DealerSeat: e.DealerSeat,
				Players:    make([]dto.HandHistoryPlayer, 0),
				Actions:    make([]dto.HandHistoryAction, 0),
				Pots:       make([]dto.HandHistoryPot, 0),
				Winners:    make([]dto.HandHistoryWinner, 0),
			}
			handStarted = true

		case *events.PlayerJoined:
			// Add player to current hand if one is in progress
			if handStarted {
				currentHand.Players = append(currentHand.Players, dto.HandHistoryPlayer{
					PlayerID:   e.PlayerID,
					Username:   e.Username,
					SeatNumber: 0, // TODO: Get from PlayerSeated event
					StartChips: 0, // TODO: Get from actual game state
				})
			}

		case *events.PlayerAction:
			// Record player action
			if handStarted {
				currentHand.Actions = append(currentHand.Actions, dto.HandHistoryAction{
					PlayerID:  e.PlayerID,
					Action:    e.Action,
					Amount:    e.Amount,
					Stage:     "unknown", // TODO: Get from game state
					Timestamp: e.Timestamp.Format("2006-01-02T15:04:05Z"),
				})
			}

		case *events.HandEnded:
			// Complete current hand
			if handStarted {
				currentHand.EndTime = e.Timestamp.Format("2006-01-02T15:04:05Z")
				handHistories = append(handHistories, currentHand)
				handStarted = false
			}
		}
	}

	// Add the last hand if it's still in progress
	if handStarted {
		handHistories = append(handHistories, currentHand)
	}

	return handHistories, nil
}
