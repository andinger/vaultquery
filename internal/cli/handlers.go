package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/andinger/vaultquery/internal/config"
	"github.com/andinger/vaultquery/internal/dql"
	"github.com/andinger/vaultquery/internal/executor"
	"github.com/andinger/vaultquery/internal/index"
	"github.com/andinger/vaultquery/internal/indexer"
	"github.com/spf13/cobra"
)

func getFlags(cmd *cobra.Command) (vaultRoot, dbPath string, err error) {
	vaultFlag, _ := cmd.Flags().GetString("vault")
	dbFlag, _ := cmd.Flags().GetString("db")

	vaultRoot, err = config.ResolveVaultRoot(vaultFlag)
	if err != nil {
		return "", "", fmt.Errorf("resolving vault root: %w", err)
	}

	dbPath = dbFlag
	if dbPath == "" {
		dbPath = config.DefaultDBPath()
	}
	return vaultRoot, dbPath, nil
}

func ensureIndex(cmd *cobra.Command) (*index.Store, error) {
	vaultRoot, dbPath, err := getFlags(cmd)
	if err != nil {
		return nil, err
	}

	store, err := index.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening index: %w", err)
	}

	fs := indexer.NewRealFS()
	idx := indexer.New(store, fs)
	if err := idx.Update(vaultRoot); err != nil {
		_ = store.Close()
		return nil, fmt.Errorf("updating index: %w", err)
	}

	return store, nil
}

func runQuery(cmd *cobra.Command, args []string) error {
	query, err := dql.Parse(args[0])
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	store, err := ensureIndex(cmd)
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
	vaultRoot, dbPath, err := getFlags(cmd)
	if err != nil {
		return err
	}

	store, err := index.Open(dbPath)
	if err != nil {
		return fmt.Errorf("opening index: %w", err)
	}
	defer func() { _ = store.Close() }()

	if err := store.DropAll(); err != nil {
		return fmt.Errorf("dropping index: %w", err)
	}

	fs := indexer.NewRealFS()
	idx := indexer.New(store, fs)
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
	_, dbPath, err := getFlags(cmd)
	if err != nil {
		return err
	}

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

	vaultRoot, _ := store.GetMeta("vault_root")

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(map[string]any{
		"indexed":    true,
		"db_path":    dbPath,
		"vault_root": vaultRoot,
		"files":      stats.FileCount,
	})
}
