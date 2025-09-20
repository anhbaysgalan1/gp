package handlers

import (
	"context"
	"fmt"

	"github.com/anhbaysgalan1/gp/internal/application/dto"
	"github.com/anhbaysgalan1/gp/internal/engine/domain/aggregates"
	"github.com/anhbaysgalan1/gp/internal/engine/domain/table"
	"github.com/anhbaysgalan1/gp/internal/engine/repositories"
	"github.com/google/uuid"
)

// CommandHandler handles all command operations for the poker engine
type CommandHandler struct {
	tableRepository *repositories.TableRepository
}

// NewCommandHandler creates a new command handler
func NewCommandHandler(tableRepository *repositories.TableRepository) *CommandHandler {
	return &CommandHandler{
		tableRepository: tableRepository,
	}
}

// CreateTable handles the create table command
func (ch *CommandHandler) CreateTable(ctx context.Context, cmd dto.CreateTableCommand) (*table.Table, error) {
	// Validate command
	if cmd.Name == "" {
		return nil, fmt.Errorf("table name is required")
	}
	if cmd.MaxPlayers < 2 || cmd.MaxPlayers > 10 {
		return nil, fmt.Errorf("max players must be between 2 and 10")
	}
	if cmd.SmallBlind <= 0 || cmd.BigBlind <= cmd.SmallBlind {
		return nil, fmt.Errorf("invalid blind structure")
	}

	// Create new table aggregate
	tableID := uuid.New()
	tableAggregate := aggregates.NewTableAggregate(
		tableID,
		cmd.Name,
		cmd.Type,
		cmd.MaxPlayers,
		cmd.SmallBlind,
		cmd.BigBlind,
		cmd.Config,
	)

	// Save the aggregate
	if err := ch.tableRepository.Save(ctx, tableAggregate); err != nil {
		return nil, fmt.Errorf("failed to save table: %w", err)
	}

	return tableAggregate.Table, nil
}

// JoinTable handles the join table command
func (ch *CommandHandler) JoinTable(ctx context.Context, cmd dto.JoinTableCommand) error {
	// Validate command
	if cmd.PlayerID == uuid.Nil {
		return fmt.Errorf("player ID is required")
	}
	if cmd.Username == "" {
		return fmt.Errorf("username is required")
	}

	// Load table aggregate
	aggregate, err := ch.tableRepository.GetByID(ctx, cmd.TableID)
	if err != nil {
		return fmt.Errorf("failed to get table: %w", err)
	}

	tableAggregate, ok := aggregate.(*aggregates.TableAggregate)
	if !ok {
		return fmt.Errorf("expected TableAggregate, got %T", aggregate)
	}

	// Execute command
	if err := tableAggregate.AddPlayer(cmd.PlayerID, cmd.Username, cmd.Avatar); err != nil {
		return fmt.Errorf("failed to add player: %w", err)
	}

	// Save changes
	if err := ch.tableRepository.Save(ctx, tableAggregate); err != nil {
		return fmt.Errorf("failed to save table: %w", err)
	}

	return nil
}

// LeaveTable handles the leave table command
func (ch *CommandHandler) LeaveTable(ctx context.Context, cmd dto.LeaveTableCommand) error {
	// Validate command
	if cmd.PlayerID == uuid.Nil {
		return fmt.Errorf("player ID is required")
	}

	// Load table aggregate
	aggregate, err := ch.tableRepository.GetByID(ctx, cmd.TableID)
	if err != nil {
		return fmt.Errorf("failed to get table: %w", err)
	}

	tableAggregate, ok := aggregate.(*aggregates.TableAggregate)
	if !ok {
		return fmt.Errorf("expected TableAggregate, got %T", aggregate)
	}

	// Execute command
	if err := tableAggregate.RemovePlayer(cmd.PlayerID, cmd.Reason); err != nil {
		return fmt.Errorf("failed to remove player: %w", err)
	}

	// Save changes
	if err := ch.tableRepository.Save(ctx, tableAggregate); err != nil {
		return fmt.Errorf("failed to save table: %w", err)
	}

	return nil
}

// SeatPlayer handles the seat player command
func (ch *CommandHandler) SeatPlayer(ctx context.Context, cmd dto.SeatPlayerCommand) error {
	// Validate command
	if cmd.PlayerID == uuid.Nil {
		return fmt.Errorf("player ID is required")
	}
	if cmd.SessionID == uuid.Nil {
		return fmt.Errorf("session ID is required")
	}
	if cmd.SeatNumber < 1 {
		return fmt.Errorf("invalid seat number")
	}
	if cmd.BuyInAmount <= 0 {
		return fmt.Errorf("buy-in amount must be positive")
	}

	// Load table aggregate
	aggregate, err := ch.tableRepository.GetByID(ctx, cmd.TableID)
	if err != nil {
		return fmt.Errorf("failed to get table: %w", err)
	}

	tableAggregate, ok := aggregate.(*aggregates.TableAggregate)
	if !ok {
		return fmt.Errorf("expected TableAggregate, got %T", aggregate)
	}

	// Execute command
	if err := tableAggregate.SeatPlayer(cmd.PlayerID, cmd.SessionID, cmd.SeatNumber, cmd.BuyInAmount); err != nil {
		return fmt.Errorf("failed to seat player: %w", err)
	}

	// Save changes
	if err := ch.tableRepository.Save(ctx, tableAggregate); err != nil {
		return fmt.Errorf("failed to save table: %w", err)
	}

	return nil
}

// StartHand handles the start hand command
func (ch *CommandHandler) StartHand(ctx context.Context, tableID uuid.UUID) error {
	// Load table aggregate
	aggregate, err := ch.tableRepository.GetByID(ctx, tableID)
	if err != nil {
		return fmt.Errorf("failed to get table: %w", err)
	}

	tableAggregate, ok := aggregate.(*aggregates.TableAggregate)
	if !ok {
		return fmt.Errorf("expected TableAggregate, got %T", aggregate)
	}

	// Execute command
	if err := tableAggregate.StartHand(); err != nil {
		return fmt.Errorf("failed to start hand: %w", err)
	}

	// Save changes
	if err := ch.tableRepository.Save(ctx, tableAggregate); err != nil {
		return fmt.Errorf("failed to save table: %w", err)
	}

	return nil
}

// PlayerAction handles the player action command
func (ch *CommandHandler) PlayerAction(ctx context.Context, cmd dto.PlayerActionCommand) error {
	// Validate command
	if cmd.PlayerID == uuid.Nil {
		return fmt.Errorf("player ID is required")
	}
	if cmd.Action == "" {
		return fmt.Errorf("action is required")
	}

	// Load table aggregate
	aggregate, err := ch.tableRepository.GetByID(ctx, cmd.TableID)
	if err != nil {
		return fmt.Errorf("failed to get table: %w", err)
	}

	tableAggregate, ok := aggregate.(*aggregates.TableAggregate)
	if !ok {
		return fmt.Errorf("expected TableAggregate, got %T", aggregate)
	}

	// Execute command
	if err := tableAggregate.PlayerAction(cmd.PlayerID, cmd.Action, cmd.Amount); err != nil {
		return fmt.Errorf("failed to execute player action: %w", err)
	}

	// Save changes
	if err := ch.tableRepository.Save(ctx, tableAggregate); err != nil {
		return fmt.Errorf("failed to save table: %w", err)
	}

	return nil
}
