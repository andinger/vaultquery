package internal_test

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/andinger/vaultquery/internal/dql"
	"github.com/andinger/vaultquery/internal/executor"
	"github.com/andinger/vaultquery/internal/index"
	"github.com/andinger/vaultquery/internal/indexer"
)

type testVault struct {
	root string
}

func newTestVault(t *testing.T) *testVault {
	t.Helper()
	return &testVault{root: t.TempDir()}
}

func (v *testVault) addFile(t *testing.T, relPath string, frontmatter map[string]any, body string) {
	t.Helper()
	absPath := filepath.Join(v.root, relPath)
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		t.Fatal(err)
	}
	var content string
	if len(frontmatter) > 0 {
		content = "---\n"
		for k, val := range frontmatter {
			switch v := val.(type) {
			case []string:
				content += fmt.Sprintf("%s:\n", k)
				for _, item := range v {
					content += fmt.Sprintf("  - %s\n", item)
				}
			default:
				content += fmt.Sprintf("%s: %v\n", k, v)
			}
		}
		content += "---\n"
	}
	content += body
	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func setupTestVault(t *testing.T) (*testVault, *index.Store) {
	t.Helper()
	vault := newTestVault(t)

	vault.addFile(t, "Clients/Acme Corp/Production/CLUSTER.md", map[string]any{
		"type":            "Kubernetes Cluster",
		"customer":        "Acme Corp",
		"kubectl_context": "acme-prod",
		"status":          "active",
	}, "# Acme Production Cluster\n")

	vault.addFile(t, "Clients/Globex Inc/Staging/CLUSTER.md", map[string]any{
		"type":            "Kubernetes Cluster",
		"customer":        "Globex Inc",
		"kubectl_context": "globex-stg",
		"status":          "active",
	}, "# Globex Staging Cluster\n")

	vault.addFile(t, "Clients/Acme Corp/Webserver/SYSTEM.md", map[string]any{
		"type":     "System",
		"customer": "Acme Corp",
		"os":       "Ubuntu 24.04",
		"tags":     []string{"linux", "webserver", "nginx"},
	}, "# Acme Webserver\n")

	vault.addFile(t, "Sales/Initech/LEAD.md", map[string]any{
		"type":   "Lead",
		"status": "active",
		"value":  "5000",
	}, "# Initech Opportunity\n")

	vault.addFile(t, "Sales/Umbrella/LEAD.md", map[string]any{
		"type":   "Lead",
		"status": "lost",
		"value":  "3000",
	}, "# Umbrella Corp Deal\n")

	store, err := index.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	fs := indexer.NewRealFS()
	idx := indexer.New(store, fs, slog.New(slog.DiscardHandler))
	if err := idx.Update(vault.root); err != nil {
		t.Fatal(err)
	}

	return vault, store
}

func TestIntegration_TableKubernetesClusters(t *testing.T) {
	_, store := setupTestVault(t)

	query, err := dql.Parse(`TABLE customer, kubectl_context FROM "Clients" WHERE type = 'Kubernetes Cluster' SORT customer ASC`)
	if err != nil {
		t.Fatal(err)
	}

	exec := executor.New(store)
	result, err := exec.Execute(query)
	if err != nil {
		t.Fatal(err)
	}

	if result.Mode != "TABLE" {
		t.Errorf("expected TABLE mode, got %s", result.Mode)
	}
	if len(result.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result.Results))
	}

	first := result.Results[0]
	if first["customer"] != "Acme Corp" {
		t.Errorf("expected first customer 'Acme Corp', got %v", first["customer"])
	}
	if first["kubectl_context"] != "acme-prod" {
		t.Errorf("expected kubectl_context 'acme-prod', got %v", first["kubectl_context"])
	}

	second := result.Results[1]
	if second["customer"] != "Globex Inc" {
		t.Errorf("expected second customer 'Globex Inc', got %v", second["customer"])
	}
}

func TestIntegration_ListActiveLeads(t *testing.T) {
	_, store := setupTestVault(t)

	query, err := dql.Parse(`LIST FROM "Sales" WHERE type = 'Lead' AND status != 'lost'`)
	if err != nil {
		t.Fatal(err)
	}

	exec := executor.New(store)
	result, err := exec.Execute(query)
	if err != nil {
		t.Fatal(err)
	}

	if result.Mode != "LIST" {
		t.Errorf("expected LIST mode, got %s", result.Mode)
	}
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	if result.Results[0]["title"] != "Initech Opportunity" {
		t.Errorf("expected title 'Initech Opportunity', got %v", result.Results[0]["title"])
	}
}

func TestIntegration_ContainsTag(t *testing.T) {
	_, store := setupTestVault(t)

	query, err := dql.Parse(`TABLE customer FROM "Clients" WHERE tags contains 'linux'`)
	if err != nil {
		t.Fatal(err)
	}

	exec := executor.New(store)
	result, err := exec.Execute(query)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	if result.Results[0]["customer"] != "Acme Corp" {
		t.Errorf("expected customer 'Acme Corp', got %v", result.Results[0]["customer"])
	}
}

func TestIntegration_ListAll(t *testing.T) {
	_, store := setupTestVault(t)

	query, err := dql.Parse(`LIST`)
	if err != nil {
		t.Fatal(err)
	}

	exec := executor.New(store)
	result, err := exec.Execute(query)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Results) != 5 {
		t.Errorf("expected 5 results, got %d", len(result.Results))
	}
}

func TestIntegration_Limit(t *testing.T) {
	_, store := setupTestVault(t)

	query, err := dql.Parse(`LIST LIMIT 2`)
	if err != nil {
		t.Fatal(err)
	}

	exec := executor.New(store)
	result, err := exec.Execute(query)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(result.Results))
	}
}

func TestIntegration_ExistsOperator(t *testing.T) {
	_, store := setupTestVault(t)

	query, err := dql.Parse(`LIST WHERE kubectl_context exists`)
	if err != nil {
		t.Fatal(err)
	}

	exec := executor.New(store)
	result, err := exec.Execute(query)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Results) != 2 {
		t.Errorf("expected 2 results (clusters with kubectl_context), got %d", len(result.Results))
	}
}

func TestIntegration_OrOperator(t *testing.T) {
	_, store := setupTestVault(t)

	query, err := dql.Parse(`LIST WHERE type = 'Lead' OR type = 'System'`)
	if err != nil {
		t.Fatal(err)
	}

	exec := executor.New(store)
	result, err := exec.Execute(query)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Results) != 3 {
		t.Errorf("expected 3 results (2 leads + 1 system), got %d", len(result.Results))
	}
}

func TestIntegration_IncrementalUpdate(t *testing.T) {
	vault, store := setupTestVault(t)

	vault.addFile(t, "Clients/Initech/CLUSTER.md", map[string]any{
		"type":            "Kubernetes Cluster",
		"customer":        "Initech",
		"kubectl_context": "initech-prod",
	}, "# Initech Cluster\n")

	fs := indexer.NewRealFS()
	idx := indexer.New(store, fs, slog.New(slog.DiscardHandler))
	if err := idx.Update(vault.root); err != nil {
		t.Fatal(err)
	}

	query, err := dql.Parse(`LIST WHERE type = 'Kubernetes Cluster'`)
	if err != nil {
		t.Fatal(err)
	}

	exec := executor.New(store)
	result, err := exec.Execute(query)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Results) != 3 {
		t.Errorf("expected 3 clusters after adding one, got %d", len(result.Results))
	}
}
