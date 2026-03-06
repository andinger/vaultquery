package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const vaultDirName = ".vaultquery"

// Config holds vault-local configuration.
type Config struct {
	Exclude []string `yaml:"exclude"`
}

// VaultDir returns the path to the .vaultquery directory inside a vault.
func VaultDir(vaultRoot string) string {
	return filepath.Join(vaultRoot, vaultDirName)
}

// VaultDBPath returns the path to the SQLite database inside a vault.
func VaultDBPath(vaultRoot string) string {
	return filepath.Join(vaultRoot, vaultDirName, "index.db")
}

// LoadConfig reads .vaultquery/config.yaml from the vault root.
// Returns an empty config if the file does not exist.
func LoadConfig(vaultRoot string) (*Config, error) {
	path := filepath.Join(vaultRoot, vaultDirName, "config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// EnsureVaultDir creates the .vaultquery directory and a .gitignore file inside it.
func EnsureVaultDir(vaultRoot string) error {
	dir := VaultDir(vaultRoot)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	gitignore := filepath.Join(dir, ".gitignore")
	if _, err := os.Stat(gitignore); os.IsNotExist(err) {
		return os.WriteFile(gitignore, []byte("*\n"), 0o644)
	}
	return nil
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
