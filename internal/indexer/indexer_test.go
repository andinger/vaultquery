package indexer

import (
	"log/slog"
	"testing"
	"time"

	"github.com/andinger/vaultquery/internal/index"
)

var nopLogger = slog.New(slog.DiscardHandler)

const testRoot = "/vault"

var testTime = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

func testContent() []byte {
	return []byte(`---
type: Kubernetes Cluster
customer: Acme Corp
kubectl_context: acme-prod
tags:
  - linux
  - production
---
# Acme Production Cluster
`)
}

func setupStore(t *testing.T) *index.Store {
	t.Helper()
	store, err := index.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func TestInitialIndex(t *testing.T) {
	store := setupStore(t)
	mfs := NewMemFS()
	mfs.AddFile(testRoot+"/clusters/acme.md", testContent(), testTime)
	mfs.AddFile(testRoot+"/clusters/beta.md", []byte("# Beta Cluster\n"), testTime.Add(time.Hour))

	idx := New(store, mfs, nopLogger)
	if err := idx.Update(testRoot); err != nil {
		t.Fatal(err)
	}

	files, err := store.ListFiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}

	fileMap := make(map[string]index.FileInfo)
	for _, f := range files {
		fileMap[f.Path] = f
	}

	acme, ok := fileMap["clusters/acme.md"]
	if !ok {
		t.Fatal("clusters/acme.md not found")
	}
	if acme.Mtime != testTime.Unix() {
		t.Errorf("mtime = %d, want %d", acme.Mtime, testTime.Unix())
	}

	root, err := store.GetMeta("vault_root")
	if err != nil {
		t.Fatal(err)
	}
	if root != testRoot {
		t.Errorf("vault_root = %q, want %q", root, testRoot)
	}
}

func TestIncrementalAdd(t *testing.T) {
	store := setupStore(t)
	mfs := NewMemFS()
	mfs.AddFile(testRoot+"/a.md", []byte("# A\n"), testTime)

	idx := New(store, mfs, nopLogger)
	if err := idx.Update(testRoot); err != nil {
		t.Fatal(err)
	}

	// Add new file
	mfs.AddFile(testRoot+"/b.md", []byte("# B\n"), testTime.Add(time.Hour))
	if err := idx.Update(testRoot); err != nil {
		t.Fatal(err)
	}

	files, _ := store.ListFiles()
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
}

func TestIncrementalChange(t *testing.T) {
	store := setupStore(t)
	mfs := NewMemFS()
	mfs.AddFile(testRoot+"/a.md", []byte("# Original\n"), testTime)

	idx := New(store, mfs, nopLogger)
	if err := idx.Update(testRoot); err != nil {
		t.Fatal(err)
	}

	// Modify file
	newContent := []byte("# Updated\nMore content.\n")
	mfs.AddFile(testRoot+"/a.md", newContent, testTime.Add(time.Hour))
	if err := idx.Update(testRoot); err != nil {
		t.Fatal(err)
	}

	files, _ := store.ListFiles()
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Mtime != testTime.Add(time.Hour).Unix() {
		t.Errorf("mtime not updated")
	}
	if files[0].Size != int64(len(newContent)) {
		t.Errorf("size = %d, want %d", files[0].Size, len(newContent))
	}
}

func TestIncrementalDelete(t *testing.T) {
	store := setupStore(t)
	mfs := NewMemFS()
	mfs.AddFile(testRoot+"/a.md", []byte("# A\n"), testTime)
	mfs.AddFile(testRoot+"/b.md", []byte("# B\n"), testTime)

	idx := New(store, mfs, nopLogger)
	if err := idx.Update(testRoot); err != nil {
		t.Fatal(err)
	}

	// Remove file
	delete(mfs.Files, testRoot+"/b.md")
	if err := idx.Update(testRoot); err != nil {
		t.Fatal(err)
	}

	files, _ := store.ListFiles()
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Path != "a.md" {
		t.Errorf("remaining file = %q, want %q", files[0].Path, "a.md")
	}
}

func TestNoChanges(t *testing.T) {
	store := setupStore(t)
	mfs := NewMemFS()
	mfs.AddFile(testRoot+"/a.md", []byte("# A\n"), testTime)

	idx := New(store, mfs, nopLogger)
	if err := idx.Update(testRoot); err != nil {
		t.Fatal(err)
	}

	// Run again without changes
	if err := idx.Update(testRoot); err != nil {
		t.Fatal(err)
	}

	files, _ := store.ListFiles()
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
}
