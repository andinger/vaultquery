package indexer

import (
	"io/fs"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/andinger/vaultquery/internal/index"
)

type fsInfo struct {
	mtime int64
	size  int64
}

// Indexer performs change detection and index updates.
type Indexer struct {
	store *index.Store
	fs    FS
	log   *slog.Logger
}

// New creates a new Indexer.
func New(store *index.Store, fs FS, log *slog.Logger) *Indexer {
	return &Indexer{store: store, fs: fs, log: log}
}

// Update scans the vault at root and synchronises the index.
func (idx *Indexer) Update(root string) error {
	// 1. Walk all .md files
	fsFiles := make(map[string]fsInfo)
	err := idx.fs.Walk(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		info, err := idx.fs.Stat(path)
		if err != nil {
			return err
		}
		fsFiles[rel] = fsInfo{mtime: info.ModTime.Unix(), size: info.Size}
		return nil
	})
	if err != nil {
		return err
	}

	// 2. Get current index state
	indexed, err := idx.store.ListFiles()
	if err != nil {
		return err
	}
	indexMap := make(map[string]index.FileInfo, len(indexed))
	for _, f := range indexed {
		indexMap[f.Path] = f
	}

	idx.log.Info("scan complete", "files_found", len(fsFiles), "files_indexed", len(indexed))

	// 3. Compute delta
	var toDelete []string
	var toUpsert []string

	for path := range fsFiles {
		if existing, ok := indexMap[path]; ok {
			fi := fsFiles[path]
			if fi.mtime != existing.Mtime || fi.size != existing.Size {
				toUpsert = append(toUpsert, path)
			}
		} else {
			toUpsert = append(toUpsert, path)
		}
	}
	for path := range indexMap {
		if _, ok := fsFiles[path]; !ok {
			toDelete = append(toDelete, path)
		}
	}

	idx.log.Info("changes detected", "to_upsert", len(toUpsert), "to_delete", len(toDelete))

	// 4. Begin transaction
	tx, err := idx.store.BeginTx()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// 5. Delete removed files
	for _, path := range toDelete {
		if err := idx.store.DeleteFileTx(tx, path); err != nil {
			return err
		}
	}

	// 6. Upsert new/changed files
	var skipped int
	for _, rel := range toUpsert {
		absPath := filepath.Join(root, rel)
		data, err := idx.fs.ReadFile(absPath)
		if err != nil {
			return err
		}
		fields, title, err := ParseFrontmatter(data)
		if err != nil {
			idx.log.Warn("skipping file with invalid frontmatter", "file", rel, "error", err)
			skipped++
			continue
		}
		fi := fsFiles[rel]
		fileID, err := idx.store.UpsertFileTx(tx, rel, fi.mtime, fi.size, title)
		if err != nil {
			return err
		}
		if err := idx.store.SetFieldsTx(tx, fileID, fields); err != nil {
			return err
		}
	}
	if skipped > 0 {
		idx.log.Warn("some files were skipped due to invalid frontmatter", "skipped", skipped)
	}

	// 7. Set vault root metadata
	if _, err := tx.Exec(
		"INSERT INTO meta (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value=excluded.value",
		"vault_root", root,
	); err != nil {
		return err
	}

	// 8. Commit
	return tx.Commit()
}
