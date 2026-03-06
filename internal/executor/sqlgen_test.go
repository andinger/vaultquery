package executor

import (
	"testing"

	"github.com/andinger/vaultquery/internal/dql"
)

func TestGenerateSQL_SimpleList(t *testing.T) {
	q := &dql.Query{Mode: "LIST"}
	sql, args, err := GenerateSQL(q)
	if err != nil {
		t.Fatal(err)
	}
	expected := "SELECT DISTINCT f.id, f.path, f.title FROM files f"
	if sql != expected {
		t.Errorf("got %q, want %q", sql, expected)
	}
	if len(args) != 0 {
		t.Errorf("expected no args, got %v", args)
	}
}

func TestGenerateSQL_ListWithFrom(t *testing.T) {
	q := &dql.Query{Mode: "LIST", From: "Clients"}
	sql, args, err := GenerateSQL(q)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := sql, "SELECT DISTINCT f.id, f.path, f.title FROM files f WHERE f.path LIKE ? || '%'"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if len(args) != 1 || args[0] != "Clients/" {
		t.Errorf("expected args [Clients/], got %v", args)
	}
}

func TestGenerateSQL_WhereEquals(t *testing.T) {
	q := &dql.Query{
		Mode:  "LIST",
		Where: dql.ComparisonExpr{Field: "type", Op: "=", Value: "Cluster"},
	}
	sql, args, err := GenerateSQL(q)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := sql, "SELECT DISTINCT f.id, f.path, f.title FROM files f WHERE f.id IN (SELECT file_id FROM fields WHERE key = ? AND value = ?)"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if len(args) != 2 || args[0] != "type" || args[1] != "Cluster" {
		t.Errorf("unexpected args: %v", args)
	}
}

func TestGenerateSQL_WhereNotEquals(t *testing.T) {
	q := &dql.Query{
		Mode:  "LIST",
		Where: dql.ComparisonExpr{Field: "status", Op: "!=", Value: "lost"},
	}
	sql, args, err := GenerateSQL(q)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := sql, "SELECT DISTINCT f.id, f.path, f.title FROM files f WHERE f.id NOT IN (SELECT file_id FROM fields WHERE key = ? AND value = ?)"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if len(args) != 2 || args[0] != "status" || args[1] != "lost" {
		t.Errorf("unexpected args: %v", args)
	}
}

func TestGenerateSQL_WhereContains(t *testing.T) {
	q := &dql.Query{
		Mode:  "LIST",
		Where: dql.ComparisonExpr{Field: "tags", Op: "contains", Value: "linux"},
	}
	sql, args, err := GenerateSQL(q)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := sql, "SELECT DISTINCT f.id, f.path, f.title FROM files f WHERE f.id IN (SELECT file_id FROM fields WHERE key = ? AND value = ?)"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if len(args) != 2 || args[0] != "tags" || args[1] != "linux" {
		t.Errorf("unexpected args: %v", args)
	}
}

func TestGenerateSQL_WhereExists(t *testing.T) {
	q := &dql.Query{
		Mode:  "LIST",
		Where: dql.ExistsExpr{Field: "status", Negated: false},
	}
	sql, args, err := GenerateSQL(q)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := sql, "SELECT DISTINCT f.id, f.path, f.title FROM files f WHERE f.id IN (SELECT file_id FROM fields WHERE key = ?)"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if len(args) != 1 || args[0] != "status" {
		t.Errorf("unexpected args: %v", args)
	}
}

func TestGenerateSQL_WhereNotExists(t *testing.T) {
	q := &dql.Query{
		Mode:  "LIST",
		Where: dql.ExistsExpr{Field: "status", Negated: true},
	}
	sql, args, err := GenerateSQL(q)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := sql, "SELECT DISTINCT f.id, f.path, f.title FROM files f WHERE f.id NOT IN (SELECT file_id FROM fields WHERE key = ?)"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if len(args) != 1 || args[0] != "status" {
		t.Errorf("unexpected args: %v", args)
	}
}

func TestGenerateSQL_AndOr(t *testing.T) {
	q := &dql.Query{
		Mode: "LIST",
		Where: dql.LogicalExpr{
			Op:    "AND",
			Left:  dql.ComparisonExpr{Field: "type", Op: "=", Value: "Lead"},
			Right: dql.ComparisonExpr{Field: "status", Op: "!=", Value: "lost"},
		},
	}
	sql, args, err := GenerateSQL(q)
	if err != nil {
		t.Fatal(err)
	}
	wantSQL := "SELECT DISTINCT f.id, f.path, f.title FROM files f WHERE (f.id IN (SELECT file_id FROM fields WHERE key = ? AND value = ?) AND f.id NOT IN (SELECT file_id FROM fields WHERE key = ? AND value = ?))"
	if sql != wantSQL {
		t.Errorf("got %q, want %q", sql, wantSQL)
	}
	if len(args) != 4 {
		t.Errorf("expected 4 args, got %v", args)
	}
}

func TestGenerateSQL_Sort(t *testing.T) {
	q := &dql.Query{
		Mode: "TABLE",
		Fields: []string{"customer"},
		Sort: []dql.SortField{{Field: "customer", Desc: false}},
	}
	sql, args, err := GenerateSQL(q)
	if err != nil {
		t.Fatal(err)
	}
	wantSQL := "SELECT DISTINCT f.id, f.path, f.title FROM files f LEFT JOIN fields sort0 ON sort0.file_id = f.id AND sort0.key = ? ORDER BY sort0.value ASC"
	if sql != wantSQL {
		t.Errorf("got %q, want %q", sql, wantSQL)
	}
	if len(args) != 1 || args[0] != "customer" {
		t.Errorf("unexpected args: %v", args)
	}
}

func TestGenerateSQL_SortDesc(t *testing.T) {
	q := &dql.Query{
		Mode:   "LIST",
		Sort:   []dql.SortField{{Field: "name", Desc: true}},
	}
	sql, _, err := GenerateSQL(q)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := sql, "SELECT DISTINCT f.id, f.path, f.title FROM files f LEFT JOIN fields sort0 ON sort0.file_id = f.id AND sort0.key = ? ORDER BY sort0.value DESC"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestGenerateSQL_Limit(t *testing.T) {
	q := &dql.Query{Mode: "LIST", Limit: 5}
	sql, args, err := GenerateSQL(q)
	if err != nil {
		t.Fatal(err)
	}
	wantSQL := "SELECT DISTINCT f.id, f.path, f.title FROM files f LIMIT ?"
	if sql != wantSQL {
		t.Errorf("got %q, want %q", sql, wantSQL)
	}
	if len(args) != 1 || args[0] != 5 {
		t.Errorf("unexpected args: %v", args)
	}
}

func TestGenerateSQL_NumericComparison(t *testing.T) {
	q := &dql.Query{
		Mode:  "LIST",
		Where: dql.ComparisonExpr{Field: "priority", Op: ">", Value: "3"},
	}
	sql, args, err := GenerateSQL(q)
	if err != nil {
		t.Fatal(err)
	}
	wantSQL := "SELECT DISTINCT f.id, f.path, f.title FROM files f WHERE f.id IN (SELECT file_id FROM fields WHERE key = ? AND CAST(value AS REAL) > CAST(? AS REAL))"
	if sql != wantSQL {
		t.Errorf("got %q, want %q", sql, wantSQL)
	}
	if len(args) != 2 || args[0] != "priority" || args[1] != "3" {
		t.Errorf("unexpected args: %v", args)
	}
}

func TestGenerateSQL_OrExpression(t *testing.T) {
	q := &dql.Query{
		Mode: "LIST",
		Where: dql.LogicalExpr{
			Op:    "OR",
			Left:  dql.ComparisonExpr{Field: "status", Op: "=", Value: "active"},
			Right: dql.ComparisonExpr{Field: "status", Op: "=", Value: "pending"},
		},
	}
	sql, args, err := GenerateSQL(q)
	if err != nil {
		t.Fatal(err)
	}
	wantSQL := "SELECT DISTINCT f.id, f.path, f.title FROM files f WHERE (f.id IN (SELECT file_id FROM fields WHERE key = ? AND value = ?) OR f.id IN (SELECT file_id FROM fields WHERE key = ? AND value = ?))"
	if sql != wantSQL {
		t.Errorf("got %q, want %q", sql, wantSQL)
	}
	if len(args) != 4 {
		t.Errorf("expected 4 args, got %v", args)
	}
}
