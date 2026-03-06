package executor

import (
	"testing"

	"github.com/andinger/vaultquery/internal/dql"
	"github.com/andinger/vaultquery/internal/index"
)

func setupTestStore(t *testing.T) *index.Store {
	t.Helper()
	store, err := index.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}

	// Acme Production Cluster
	id1, err := store.UpsertFile("Clients/Acme Corp/Production/CLUSTER.md", 1000, 100, "Acme Production Cluster")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SetFields(id1, map[string][]string{
		"type":            {"Kubernetes Cluster"},
		"customer":        {"Acme Corp"},
		"kubectl_context": {"acme-prod"},
		"status":          {"active"},
	}); err != nil {
		t.Fatal(err)
	}

	// Globex Staging Cluster
	id2, err := store.UpsertFile("Clients/Globex Inc/Staging/CLUSTER.md", 1001, 200, "Globex Staging Cluster")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SetFields(id2, map[string][]string{
		"type":            {"Kubernetes Cluster"},
		"customer":        {"Globex Inc"},
		"kubectl_context": {"globex-staging"},
		"status":          {"active"},
	}); err != nil {
		t.Fatal(err)
	}

	// Initech Lead
	id3, err := store.UpsertFile("Sales/Leads/Initech.md", 1002, 50, "Initech Lead")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SetFields(id3, map[string][]string{
		"type":   {"Lead"},
		"status": {"qualified"},
	}); err != nil {
		t.Fatal(err)
	}

	// Acme Webserver with tags
	id4, err := store.UpsertFile("Clients/Acme Corp/Production/webserver.md", 1003, 80, "Acme Webserver")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SetFields(id4, map[string][]string{
		"type":     {"VM"},
		"customer": {"Acme Corp"},
		"tags":     {"linux", "nginx", "production"},
		"status":   {"active"},
	}); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { store.Close() })
	return store
}

func TestExecute_TableWithSortAndWhere(t *testing.T) {
	store := setupTestStore(t)
	exec := New(store)

	q := &dql.Query{
		Mode:   "TABLE",
		Fields: []string{"customer", "kubectl_context"},
		Where:  dql.ComparisonExpr{Field: "type", Op: "=", Value: "Kubernetes Cluster"},
		Sort:   []dql.SortField{{Field: "customer", Desc: false}},
	}

	result, err := exec.Execute(q)
	if err != nil {
		t.Fatal(err)
	}
	if result.Mode != "TABLE" {
		t.Errorf("expected TABLE mode, got %s", result.Mode)
	}
	if len(result.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result.Results))
	}
	// Sorted by customer ASC: Acme Corp, Globex Inc
	if got := result.Results[0]["customer"]; got != "Acme Corp" {
		t.Errorf("expected Acme Corp first, got %v", got)
	}
	if got := result.Results[1]["customer"]; got != "Globex Inc" {
		t.Errorf("expected Globex Inc second, got %v", got)
	}
	if got := result.Results[0]["kubectl_context"]; got != "acme-prod" {
		t.Errorf("expected acme-prod, got %v", got)
	}
}

func TestExecute_ListFromWithAndCondition(t *testing.T) {
	store := setupTestStore(t)
	exec := New(store)

	q := &dql.Query{
		Mode: "LIST",
		From: "Sales",
		Where: dql.LogicalExpr{
			Op:    "AND",
			Left:  dql.ComparisonExpr{Field: "type", Op: "=", Value: "Lead"},
			Right: dql.ComparisonExpr{Field: "status", Op: "!=", Value: "lost"},
		},
	}

	result, err := exec.Execute(q)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	if got := result.Results[0]["path"]; got != "Sales/Leads/Initech.md" {
		t.Errorf("expected Initech path, got %v", got)
	}
}

func TestExecute_TableContains(t *testing.T) {
	store := setupTestStore(t)
	exec := New(store)

	q := &dql.Query{
		Mode:   "TABLE",
		Fields: []string{"customer"},
		Where:  dql.ComparisonExpr{Field: "tags", Op: "contains", Value: "linux"},
	}

	result, err := exec.Execute(q)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	if got := result.Results[0]["customer"]; got != "Acme Corp" {
		t.Errorf("expected Acme Corp, got %v", got)
	}
}

func TestExecute_ListWhereExists(t *testing.T) {
	store := setupTestStore(t)
	exec := New(store)

	q := &dql.Query{
		Mode:  "LIST",
		Where: dql.ExistsExpr{Field: "status", Negated: false},
	}

	result, err := exec.Execute(q)
	if err != nil {
		t.Fatal(err)
	}
	// All 4 files have status
	if len(result.Results) != 4 {
		t.Errorf("expected 4 results, got %d", len(result.Results))
	}
}

func TestExecute_TableWithLimit(t *testing.T) {
	store := setupTestStore(t)
	exec := New(store)

	q := &dql.Query{
		Mode:   "TABLE",
		Fields: []string{"customer"},
		Where:  dql.ComparisonExpr{Field: "type", Op: "=", Value: "Kubernetes Cluster"},
		Limit:  1,
	}

	result, err := exec.Execute(q)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(result.Results))
	}
}

func TestExecute_BareList(t *testing.T) {
	store := setupTestStore(t)
	exec := New(store)

	q := &dql.Query{Mode: "LIST"}

	result, err := exec.Execute(q)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Results) != 4 {
		t.Errorf("expected 4 results, got %d", len(result.Results))
	}
}

func TestExecute_WhereOr(t *testing.T) {
	store := setupTestStore(t)
	exec := New(store)

	q := &dql.Query{
		Mode: "LIST",
		Where: dql.LogicalExpr{
			Op:    "OR",
			Left:  dql.ComparisonExpr{Field: "type", Op: "=", Value: "Lead"},
			Right: dql.ComparisonExpr{Field: "type", Op: "=", Value: "VM"},
		},
	}

	result, err := exec.Execute(q)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Results) != 2 {
		t.Errorf("expected 2 results (Lead + VM), got %d", len(result.Results))
	}
}

func TestExecute_EmptyResult(t *testing.T) {
	store := setupTestStore(t)
	exec := New(store)

	q := &dql.Query{
		Mode:  "LIST",
		Where: dql.ComparisonExpr{Field: "type", Op: "=", Value: "nonexistent"},
	}

	result, err := exec.Execute(q)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Results) != 0 {
		t.Errorf("expected 0 results, got %d", len(result.Results))
	}
}

func TestExecute_MultiValueField(t *testing.T) {
	store := setupTestStore(t)
	exec := New(store)

	q := &dql.Query{
		Mode:   "TABLE",
		Fields: []string{"tags"},
		Where:  dql.ComparisonExpr{Field: "tags", Op: "contains", Value: "linux"},
	}

	result, err := exec.Execute(q)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	tags, ok := result.Results[0]["tags"].([]string)
	if !ok {
		t.Fatalf("expected []string for tags, got %T", result.Results[0]["tags"])
	}
	if len(tags) != 3 {
		t.Errorf("expected 3 tags, got %d", len(tags))
	}
}
