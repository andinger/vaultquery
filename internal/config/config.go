package config

import (
	"os"
	"path/filepath"
)

// DefaultDBPath returns the default database path following XDG conventions.
func DefaultDBPath() string {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			dataHome = filepath.Join(".", ".local", "share")
		} else {
			dataHome = filepath.Join(home, ".local", "share")
		}
	}
	return filepath.Join(dataHome, "vaultquery", "index.db")
}

// ResolveVaultRoot resolves the vault root directory.
// If explicit is non-empty, it is returned as-is.
// Otherwise, the current working directory is used.
func ResolveVaultRoot(explicit string) (string, error) {
	if explicit != "" {
		return filepath.Abs(explicit)
	}
	return os.Getwd()
}
