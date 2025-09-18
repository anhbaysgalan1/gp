package database

import (
	"log/slog"
)

// SetupIndexes creates additional indexes that GORM can't handle automatically
func (db *DB) SetupIndexes() error {
	slog.Info("Setting up additional database indexes")

	// Composite unique index for tournament registrations
	if err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_tournament_registrations_unique
		ON tournament_registrations(tournament_id, user_id)
		WHERE deleted_at IS NULL
	`).Error; err != nil {
		return err
	}

	// Composite unique index for user statistics
	if err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_user_statistics_unique
		ON user_statistics(user_id, stat_type)
		WHERE deleted_at IS NULL
	`).Error; err != nil {
		return err
	}

	// Composite unique index for leaderboard entries
	if err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_leaderboard_entries_unique
		ON leaderboard_entries(user_id, leaderboard_type, period_start)
		WHERE deleted_at IS NULL
	`).Error; err != nil {
		return err
	}

	// Performance indexes
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_game_sessions_performance
		ON game_sessions(user_id, joined_at DESC)
	`).Error; err != nil {
		return err
	}

	slog.Info("Additional database indexes created successfully")
	return nil
}