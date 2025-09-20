package database

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/anhbaysgalan1/gp/internal/config"
	"github.com/anhbaysgalan1/gp/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DB struct {
	*gorm.DB
}

func NewConnection(cfg *config.Config) (*DB, error) {
	slog.Info("Connecting to database with GORM")

	// Configure GORM logger
	gormLogger := logger.Default.LogMode(logger.Info)
	if cfg.Environment == "production" {
		gormLogger = logger.Default.LogMode(logger.Error)
	}

	// Open connection
	db, err := gorm.Open(postgres.Open(cfg.GetDatabaseURL()), &gorm.Config{
		Logger: gormLogger,
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying sql.DB to configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Test connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	slog.Info("Successfully connected to database")
	return &DB{DB: db}, nil
}

func (db *DB) AutoMigrate() error {
	slog.Info("Running GORM auto-migrations")

	err := db.DB.AutoMigrate(
		&models.User{},
		&models.EmailVerification{},
		&models.PokerTable{},
		&models.Tournament{},
		&models.TournamentRegistration{},
		&models.GameSession{},
		&models.LeaderboardEntry{},
		&models.UserStatistics{},
	)

	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Create additional indexes
	if err := db.SetupIndexes(); err != nil {
		return fmt.Errorf("failed to setup additional indexes: %w", err)
	}

	slog.Info("GORM auto-migrations completed successfully")
	return nil
}

func (db *DB) Close() error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return err
	}

	if err := sqlDB.Close(); err != nil {
		return err
	}

	slog.Info("Database connection closed")
	return nil
}
