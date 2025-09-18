package database

import (
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// IsUniqueConstraintError checks if the error is a unique constraint violation
func IsUniqueConstraintError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		// PostgreSQL unique constraint violation error code
		return pgErr.Code == "23505"
	}
	return false
}

// IsForeignKeyConstraintError checks if the error is a foreign key constraint violation
func IsForeignKeyConstraintError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		// PostgreSQL foreign key constraint violation error code
		return pgErr.Code == "23503"
	}
	return false
}

// IsNotFoundError checks if the error is a record not found error
func IsNotFoundError(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

// GetConstraintName extracts the constraint name from a PostgreSQL error
func GetConstraintName(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.ConstraintName
	}
	return ""
}

// GetErrorMessage returns a user-friendly error message based on the database error
func GetErrorMessage(err error) string {
	if IsNotFoundError(err) {
		return "Record not found"
	}

	if IsUniqueConstraintError(err) {
		constraintName := GetConstraintName(err)
		switch {
		case strings.Contains(constraintName, "email"):
			return "Email address already exists"
		case strings.Contains(constraintName, "username"):
			return "Username already exists"
		case strings.Contains(constraintName, "name"):
			return "Name already exists"
		default:
			return "Record already exists"
		}
	}

	if IsForeignKeyConstraintError(err) {
		return "Referenced record not found"
	}

	// Default error message
	return "Database operation failed"
}