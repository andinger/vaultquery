package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/andinger/vaultquery/internal/config"
	"github.com/andinger/vaultquery/internal/dql"
	"github.com/andinger/vaultquery/internal/executor"
	"github.com/andinger/vaultquery/internal/index"
	"github.com/andinger/vaultquery/internal/indexer"
)

func getVaultRoot(cmd *cobra.Command) (string, error) {
	vaultFlag, _ := cmd.Flags().GetString("vault")
	return config.ResolveVaultRoot(vaultFlag)
}

func ensureIndex(cmd *cobra.Command) (*index.Store, error) {
	vaultRoot, err := getVaultRoot(cmd)
	if err != nil {
		return nil, err
	}

	if err := config.EnsureVaultDir(vaultRoot); err != nil {
		return nil, fmt.Errorf("creating .vaultquery directory: %w", err)
	}

	cfg, err := config.LoadConfig(vaultRoot)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	dbPath := config.VaultDBPath(vaultRoot)
	store, err := index.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening index: %w", err)
	}

	log := newLogger(cmd)
	fs := indexer.NewRealFS()
	idx := indexer.New(store, fs, log, cfg.Exclude)
	if err := idx.Update(vaultRoot); err != nil {
		_ = store.Close()
		return nil, fmt.Errorf("updating index: %w", err)
	}

	return store, nil
}

func openIndex(cmd *cobra.Command) (*index.Store, error) {
	vaultRoot, err := getVaultRoot(cmd)
	if err != nil {
		return nil, err
	}

	dbPath := config.VaultDBPath(vaultRoot)

	// If DB doesn't exist yet, do a full sync (first-run)
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return ensureIndex(cmd)
	}

	store, err := index.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening index: %w", err)
	}

	// Check if index is empty (first-run after DB created but no data)
	stats, err := store.Stats()
	if err != nil {
		_ = store.Close()
		return nil, err
	}
	if stats.FileCount == 0 {
		_ = store.Close()
		return ensureIndex(cmd)
	}

	return store, nil
}

func runQuery(cmd *cobra.Command, args []string) error {
	query, err := dql.Parse(args[0])
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	syncFlag, _ := cmd.Flags().GetBool("sync")

	var store *index.Store
	if syncFlag {
		store, err = ensureIndex(cmd)
	} else {
		store, err = openIndex(cmd)
	}
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	exec := executor.New(store)
	result, err := exec.Execute(query)
	if err != nil {
		return fmt.Errorf("query error: %w", err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

func runIndex(cmd *cobra.Command, _ []string) error {
	start := time.Now()
	store, err := ensureIndex(cmd)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	stats, err := store.Stats()
	if err != nil {
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(map[string]any{
		"files":    stats.FileCount,
		"duration": time.Since(start).String(),
	})
}

func runReindex(cmd *cobra.Command, _ []string) error {
	vaultRoot, err := getVaultRoot(cmd)
	if err != nil {
		return err
	}

	if err := config.EnsureVaultDir(vaultRoot); err != nil {
		return fmt.Errorf("creating .vaultquery directory: %w", err)
	}

	cfg, err := config.LoadConfig(vaultRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	dbPath := config.VaultDBPath(vaultRoot)
	store, err := index.Open(dbPath)
	if err != nil {
		return fmt.Errorf("opening index: %w", err)
	}
	defer func() { _ = store.Close() }()

	if err := store.DropAll(); err != nil {
		return fmt.Errorf("dropping index: %w", err)
	}

	log := newLogger(cmd)
	fs := indexer.NewRealFS()
	idx := indexer.New(store, fs, log, cfg.Exclude)
	start := time.Now()
	if err := idx.Update(vaultRoot); err != nil {
		return fmt.Errorf("reindexing: %w", err)
	}

	stats, err := store.Stats()
	if err != nil {
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(map[string]any{
		"files":    stats.FileCount,
		"duration": time.Since(start).String(),
	})
}

func runStatus(cmd *cobra.Command, _ []string) error {
	vaultRoot, err := getVaultRoot(cmd)
	if err != nil {
		return err
	}

	dbPath := config.VaultDBPath(vaultRoot)

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]any{
			"indexed": false,
			"db_path": dbPath,
		})
	}

	store, err := index.Open(dbPath)
	if err != nil {
		return fmt.Errorf("opening index: %w", err)
	}
	defer func() { _ = store.Close() }()

	stats, err := store.Stats()
	if err != nil {
		return err
	}

	vaultMeta, _ := store.GetMeta("vault_root")

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(map[string]any{
		"indexed":    true,
		"db_path":    dbPath,
		"vault_root": vaultMeta,
		"files":      stats.FileCount,
	})
}
