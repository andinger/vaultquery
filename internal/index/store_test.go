package index

import (
	"testing"
)

func mustOpen(t *testing.T) *Store {
	t.Helper()
	s, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestOpenClose(t *testing.T) {
	s := mustOpen(t)
	// Verify schema exists by querying each table.
	for _, table := range []string{"files", "fields", "meta"} {
		_, err := s.DB().Exec("SELECT 1 FROM " + table + " LIMIT 1")
		if err != nil {
			t.Errorf("table %s not found: %v", table, err)
		}
	}
}

func TestUpsertAndList(t *testing.T) {
	s := mustOpen(t)

	id1, err := s.UpsertFile("a.md", 100, 200, "Title A")
	if err != nil {
		t.Fatalf("UpsertFile: %v", err)
	}
	id2, err := s.UpsertFile("b.md", 300, 400, "Title B")
	if err != nil {
		t.Fatalf("UpsertFile: %v", err)
	}
	if id1 == id2 {
		t.Fatalf("expected different IDs, got %d and %d", id1, id2)
	}

	files, err := s.ListFiles()
	if err != nil {
		t.Fatalf("ListFiles: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
}

func TestUpsertUpdate(t *testing.T) {
	s := mustOpen(t)

	id1, err := s.UpsertFile("a.md", 100, 200, "Title A")
	if err != nil {
		t.Fatalf("UpsertFile: %v", err)
	}
	id2, err := s.UpsertFile("a.md", 999, 888, "Updated A")
	if err != nil {
		t.Fatalf("UpsertFile: %v", err)
	}
	if id1 != id2 {
		t.Fatalf("expected same ID after update, got %d and %d", id1, id2)
	}

	files, err := s.ListFiles()
	if err != nil {
		t.Fatalf("ListFiles: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Mtime != 999 || files[0].Size != 888 {
		t.Fatalf("expected updated values, got mtime=%d size=%d", files[0].Mtime, files[0].Size)
	}
}

func TestDeleteFile(t *testing.T) {
	s := mustOpen(t)

	id, err := s.UpsertFile("a.md", 100, 200, "Title A")
	if err != nil {
		t.Fatalf("UpsertFile: %v", err)
	}
	if err := s.SetFields(id, map[string][]string{"tag": {"x", "y"}}); err != nil {
		t.Fatalf("SetFields: %v", err)
	}

	if err := s.DeleteFile("a.md"); err != nil {
		t.Fatalf("DeleteFile: %v", err)
	}

	files, err := s.ListFiles()
	if err != nil {
		t.Fatalf("ListFiles: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("expected 0 files after delete, got %d", len(files))
	}

	// Verify fields are also gone (CASCADE).
	var count int
	if err := s.DB().QueryRow("SELECT COUNT(*) FROM fields WHERE file_id=?", id).Scan(&count); err != nil {
		t.Fatalf("query fields: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 field rows after cascade delete, got %d", count)
	}
}

func TestSetFields(t *testing.T) {
	s := mustOpen(t)

	id, err := s.UpsertFile("a.md", 100, 200, "Title A")
	if err != nil {
		t.Fatalf("UpsertFile: %v", err)
	}

	fields := map[string][]string{
		"tag":    {"go", "sqlite"},
		"status": {"draft"},
	}
	if err := s.SetFields(id, fields); err != nil {
		t.Fatalf("SetFields: %v", err)
	}

	var count int
	if err := s.DB().QueryRow("SELECT COUNT(*) FROM fields WHERE file_id=?", id).Scan(&count); err != nil {
		t.Fatalf("query fields: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 field rows, got %d", count)
	}
}

func TestSetFieldsReplace(t *testing.T) {
	s := mustOpen(t)

	id, err := s.UpsertFile("a.md", 100, 200, "Title A")
	if err != nil {
		t.Fatalf("UpsertFile: %v", err)
	}

	if err := s.SetFields(id, map[string][]string{"tag": {"old1", "old2"}}); err != nil {
		t.Fatalf("SetFields: %v", err)
	}
	if err := s.SetFields(id, map[string][]string{"tag": {"new1"}}); err != nil {
		t.Fatalf("SetFields: %v", err)
	}

	var count int
	if err := s.DB().QueryRow("SELECT COUNT(*) FROM fields WHERE file_id=?", id).Scan(&count); err != nil {
		t.Fatalf("query fields: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 field row after replace, got %d", count)
	}

	var val string
	if err := s.DB().QueryRow("SELECT value FROM fields WHERE file_id=? AND key='tag'", id).Scan(&val); err != nil {
		t.Fatalf("query field value: %v", err)
	}
	if val != "new1" {
		t.Fatalf("expected 'new1', got %q", val)
	}
}

func TestMeta(t *testing.T) {
	s := mustOpen(t)

	// Nonexistent key returns "".
	val, err := s.GetMeta("version")
	if err != nil {
		t.Fatalf("GetMeta: %v", err)
	}
	if val != "" {
		t.Fatalf("expected empty string for nonexistent key, got %q", val)
	}

	if err := s.SetMeta("version", "1"); err != nil {
		t.Fatalf("SetMeta: %v", err)
	}
	val, err = s.GetMeta("version")
	if err != nil {
		t.Fatalf("GetMeta: %v", err)
	}
	if val != "1" {
		t.Fatalf("expected '1', got %q", val)
	}

	// Update existing key.
	if err := s.SetMeta("version", "2"); err != nil {
		t.Fatalf("SetMeta: %v", err)
	}
	val, err = s.GetMeta("version")
	if err != nil {
		t.Fatalf("GetMeta: %v", err)
	}
	if val != "2" {
		t.Fatalf("expected '2', got %q", val)
	}
}

func TestStats(t *testing.T) {
	s := mustOpen(t)

	stats, err := s.Stats()
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.FileCount != 0 {
		t.Fatalf("expected 0 files, got %d", stats.FileCount)
	}

	s.UpsertFile("a.md", 1, 1, "")
	s.UpsertFile("b.md", 1, 1, "")

	stats, err = s.Stats()
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.FileCount != 2 {
		t.Fatalf("expected 2 files, got %d", stats.FileCount)
	}
}

func TestDropAll(t *testing.T) {
	s := mustOpen(t)

	s.UpsertFile("a.md", 1, 1, "")
	s.SetMeta("k", "v")

	if err := s.DropAll(); err != nil {
		t.Fatalf("DropAll: %v", err)
	}

	files, err := s.ListFiles()
	if err != nil {
		t.Fatalf("ListFiles: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("expected 0 files after drop, got %d", len(files))
	}

	val, err := s.GetMeta("k")
	if err != nil {
		t.Fatalf("GetMeta: %v", err)
	}
	if val != "" {
		t.Fatalf("expected empty meta after drop, got %q", val)
	}
}
