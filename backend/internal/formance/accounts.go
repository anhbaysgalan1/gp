package formance

import (
	"fmt"
	"github.com/google/uuid"
)

// Account naming constants and utility functions to ensure consistency
// across the entire Formance integration

const (
	// Account prefixes
	PlayerAccountPrefix     = "player"
	SessionAccountPrefix    = "session"
	SystemAccountPrefix     = "system"
	WorldAccount           = "world"

	// System account types
	SystemHouseAccount = "system:house"

	// Account suffixes
	WalletSuffix = "wallet"
)

// PlayerWalletAccount returns the main wallet account name for a user
func PlayerWalletAccount(userID uuid.UUID) string {
	return fmt.Sprintf("%s:%s:%s", PlayerAccountPrefix, userID.String(), WalletSuffix)
}

// SessionAccount returns the game session account name for a user
func SessionAccount(userID, sessionID uuid.UUID) string {
	return fmt.Sprintf("%s:%s:%s", SessionAccountPrefix, userID.String(), sessionID.String())
}

// TournamentPoolAccount returns the tournament pool account name
func TournamentPoolAccount(tournamentID uuid.UUID) string {
	return fmt.Sprintf("%s:tournament_pool:%s", SystemAccountPrefix, tournamentID.String())
}

// SessionPrefix returns the prefix for filtering user session accounts
func SessionPrefix(userID uuid.UUID) string {
	return fmt.Sprintf("%s:%s:", SessionAccountPrefix, userID.String())
}

// ValidateAccountFormat validates that an account follows the expected naming convention
func ValidateAccountFormat(account string) bool {
	// Basic validation - could be enhanced with regex
	return len(account) > 0 && (
		account == WorldAccount ||
		account == SystemHouseAccount ||
		len(account) > 7) // Minimum length for formatted accounts
}