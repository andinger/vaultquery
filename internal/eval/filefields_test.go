package eval

import (
	"testing"

	"github.com/andinger/vaultquery/internal/dql"
)

func TestFileFieldsBasic(t *testing.T) {
	ctx := NewEvalContextWithMeta(
		"Clients/Acme Corp/CLUSTER.md",
		"Acme Cluster",
		map[string]dql.Value{
			"type": dql.NewString("Kubernetes Cluster"),
		},
		&FileMetadata{
			Size:  1024,
			Mtime: 1700000000,
			Tags:  []string{"linux", "prod"},
		},
	)

	tests := []struct {
		field string
		check func(dql.Value) bool
	}{
		{"file.name", func(v dql.Value) bool { s, ok := v.AsString(); return ok && s == "CLUSTER" }},
		{"file.folder", func(v dql.Value) bool { s, ok := v.AsString(); return ok && s == "Clients/Acme Corp" }},
		{"file.path", func(v dql.Value) bool { s, ok := v.AsString(); return ok && s == "Clients/Acme Corp/CLUSTER.md" }},
		{"file.ext", func(v dql.Value) bool { s, ok := v.AsString(); return ok && s == ".md" }},
		{"file.size", func(v dql.Value) bool { n, ok := v.AsNumber(); return ok && n == 1024 }},
		{"file.link", func(v dql.Value) bool { l, ok := v.AsLink(); return ok && l == "CLUSTER" }},
		{"file.mtime", func(v dql.Value) bool { _, ok := v.AsDate(); return ok }},
		{"file.tags", func(v dql.Value) bool { l, ok := v.AsList(); return ok && len(l) == 2 }},
		{"file.inlinks", func(v dql.Value) bool { l, ok := v.AsList(); return ok && len(l) == 0 }},
		{"file.outlinks", func(v dql.Value) bool { l, ok := v.AsList(); return ok && len(l) == 0 }},
		{"file.frontmatter", func(v dql.Value) bool { _, ok := v.AsObject(); return ok }},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			v := ctx.Lookup(tt.field)
			if !tt.check(v) {
				t.Errorf("field %q: unexpected value %v (type=%v)", tt.field, v, v.Type)
			}
		})
	}
}

func TestFileFieldsDateFromFilename(t *testing.T) {
	ctx := NewEvalContextWithMeta(
		"Journal/2024/01/2024-01-15.md",
		"Daily Note",
		nil,
		&FileMetadata{Size: 100, Mtime: 1700000000},
	)

	v := ctx.Lookup("file.day")
	d, ok := v.AsDate()
	if !ok {
		t.Fatalf("expected date for file.day, got %v", v)
	}
	if d.Format("2006-01-02") != "2024-01-15" {
		t.Errorf("expected 2024-01-15, got %s", d.Format("2006-01-02"))
	}
}

func TestFileFieldsWithLinks(t *testing.T) {
	ctx := NewEvalContextWithMeta(
		"notes/test.md",
		"Test",
		nil,
		&FileMetadata{
			Size:     50,
			Mtime:    1700000000,
			InLinks:  []string{"Page A", "Page B"},
			OutLinks: []string{"Page C"},
		},
	)

	inlinks, ok := ctx.Lookup("file.inlinks").AsList()
	if !ok || len(inlinks) != 2 {
		t.Errorf("expected 2 inlinks, got %v", ctx.Lookup("file.inlinks"))
	}

	outlinks, ok := ctx.Lookup("file.outlinks").AsList()
	if !ok || len(outlinks) != 1 {
		t.Errorf("expected 1 outlink, got %v", ctx.Lookup("file.outlinks"))
	}
}

func TestFileFieldsNilMeta(t *testing.T) {
	ctx := NewEvalContextWithMeta("test.md", "Test", nil, nil)

	// file.* fields should not exist without metadata
	v := ctx.Lookup("file.name")
	if !v.IsNull() {
		t.Errorf("expected null for file.name without meta, got %v", v)
	}
}
