package executor

import (
	"testing"

	"github.com/andinger/vaultquery/internal/dql"
	"github.com/andinger/vaultquery/internal/eval"
)

func TestApplyFlatten(t *testing.T) {
	ev := eval.New()
	rows := []map[string]any{
		{"path": "a.md", "tags": []string{"linux", "docker"}},
		{"path": "b.md", "tags": "single"},
		{"path": "c.md"},
	}

	flattenDefs := []dql.FieldDef{
		{Expr: dql.FieldAccessExpr{Parts: []string{"tags"}}},
	}

	result := applyFlatten(rows, flattenDefs, ev)

	// Row 0 should expand into 2 rows (linux, docker)
	// Row 1 stays as-is (single string, not a list)
	// Row 2 stays as-is (no tags)
	if len(result) != 4 {
		t.Fatalf("expected 4 rows after flatten, got %d", len(result))
	}
	if result[0]["tags"] != "linux" {
		t.Errorf("row 0 tags: expected 'linux', got %v", result[0]["tags"])
	}
	if result[1]["tags"] != "docker" {
		t.Errorf("row 1 tags: expected 'docker', got %v", result[1]["tags"])
	}
}

func TestApplyGroupBy(t *testing.T) {
	ev := eval.New()
	_ = ev

	rows := []map[string]any{
		{"path": "a.md", "type": "Lead"},
		{"path": "b.md", "type": "Cluster"},
		{"path": "c.md", "type": "Lead"},
		{"path": "d.md", "type": "Cluster"},
		{"path": "e.md"},
	}

	groupByDefs := []dql.FieldDef{
		{Expr: dql.FieldAccessExpr{Parts: []string{"type"}}},
	}

	groups := applyGroupBy(rows, groupByDefs, eval.New())

	if len(groups) != 3 {
		t.Fatalf("expected 3 groups, got %d", len(groups))
	}

	// First group encountered: Lead
	if groups[0].Key != "Lead" {
		t.Errorf("group 0 key: expected 'Lead', got %v", groups[0].Key)
	}
	if len(groups[0].Rows) != 2 {
		t.Errorf("group 0 rows: expected 2, got %d", len(groups[0].Rows))
	}

	// Second group: Cluster
	if groups[1].Key != "Cluster" {
		t.Errorf("group 1 key: expected 'Cluster', got %v", groups[1].Key)
	}
	if len(groups[1].Rows) != 2 {
		t.Errorf("group 1 rows: expected 2, got %d", len(groups[1].Rows))
	}

	// Third group: nil (no type)
	if len(groups[2].Rows) != 1 {
		t.Errorf("group 2 rows: expected 1, got %d", len(groups[2].Rows))
	}
}

func TestApplyFlattenEmpty(t *testing.T) {
	ev := eval.New()
	rows := []map[string]any{{"path": "a.md"}}
	result := applyFlatten(rows, nil, ev)
	if len(result) != 1 {
		t.Errorf("expected 1 row, got %d", len(result))
	}
}
