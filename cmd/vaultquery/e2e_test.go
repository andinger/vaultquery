package main_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

var binaryPath string

func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "vaultquery-e2e-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = os.RemoveAll(tmp) }()

	binaryPath = filepath.Join(tmp, "vaultquery")
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = "."
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build binary: %v\n%s\n", err, out)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func createTestVault(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	files := map[string]string{
		"Clients/Acme Corp/CLUSTER.md": `---
type: Kubernetes Cluster
customer: Acme Corp
kubectl_context: acme-prod
status: active
---
# Acme Production Cluster
`,
		"Clients/Globex Inc/CLUSTER.md": `---
type: Kubernetes Cluster
customer: Globex Inc
kubectl_context: globex-stg
status: active
---
# Globex Staging Cluster
`,
		"Clients/Acme Corp/SYSTEM.md": `---
type: System
customer: Acme Corp
os: Ubuntu 24.04
tags:
  - linux
  - webserver
---
# Acme Webserver
`,
		"Sales/Initech/LEAD.md": `---
type: Lead
status: active
---
# Initech Opportunity
`,
	}

	for relPath, content := range files {
		absPath := filepath.Join(root, relPath)
		if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func runVaultquery(t *testing.T, dbPath, vaultRoot string, args ...string) ([]byte, error) {
	t.Helper()
	fullArgs := append([]string{"--db", dbPath, "--vault", vaultRoot}, args...)
	cmd := exec.Command(binaryPath, fullArgs...)
	return cmd.CombinedOutput()
}

func TestE2E_IndexCommand(t *testing.T) {
	vault := createTestVault(t)
	dbPath := filepath.Join(t.TempDir(), "test.db")

	out, err := runVaultquery(t, dbPath, vault, "index")
	if err != nil {
		t.Fatalf("index command failed: %v\n%s", err, out)
	}

	var result map[string]any
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, out)
	}

	files, ok := result["files"].(float64)
	if !ok || files != 4 {
		t.Errorf("expected 4 files, got %v", result["files"])
	}
}

func TestE2E_QueryCommand(t *testing.T) {
	vault := createTestVault(t)
	dbPath := filepath.Join(t.TempDir(), "test.db")

	out, err := runVaultquery(t, dbPath, vault, "query", `LIST WHERE type = 'Kubernetes Cluster'`)
	if err != nil {
		t.Fatalf("query command failed: %v\n%s", err, out)
	}

	var result struct {
		Mode    string           `json:"mode"`
		Results []map[string]any `json:"results"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, out)
	}

	if result.Mode != "LIST" {
		t.Errorf("expected LIST mode, got %s", result.Mode)
	}
	if len(result.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(result.Results))
	}
}

func TestE2E_TableQuery(t *testing.T) {
	vault := createTestVault(t)
	dbPath := filepath.Join(t.TempDir(), "test.db")

	out, err := runVaultquery(t, dbPath, vault, "query",
		`TABLE customer, kubectl_context WHERE type = 'Kubernetes Cluster' SORT customer ASC`)
	if err != nil {
		t.Fatalf("query command failed: %v\n%s", err, out)
	}

	var result struct {
		Mode    string           `json:"mode"`
		Fields  []string         `json:"fields"`
		Results []map[string]any `json:"results"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, out)
	}

	if result.Mode != "TABLE" {
		t.Errorf("expected TABLE mode, got %s", result.Mode)
	}
	if len(result.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result.Results))
	}
	if result.Results[0]["customer"] != "Acme Corp" {
		t.Errorf("expected first customer 'Acme Corp', got %v", result.Results[0]["customer"])
	}
	if result.Results[1]["customer"] != "Globex Inc" {
		t.Errorf("expected second customer 'Globex Inc', got %v", result.Results[1]["customer"])
	}
}

func TestE2E_ReindexCommand(t *testing.T) {
	vault := createTestVault(t)
	dbPath := filepath.Join(t.TempDir(), "test.db")

	// First index
	if _, err := runVaultquery(t, dbPath, vault, "index"); err != nil {
		t.Fatalf("index failed: %v", err)
	}

	// Reindex
	out, err := runVaultquery(t, dbPath, vault, "reindex")
	if err != nil {
		t.Fatalf("reindex command failed: %v\n%s", err, out)
	}

	var result map[string]any
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, out)
	}

	files, ok := result["files"].(float64)
	if !ok || files != 4 {
		t.Errorf("expected 4 files, got %v", result["files"])
	}
}

func TestE2E_StatusCommand(t *testing.T) {
	vault := createTestVault(t)
	dbPath := filepath.Join(t.TempDir(), "test.db")

	// Status before index
	out, err := runVaultquery(t, dbPath, vault, "status")
	if err != nil {
		t.Fatalf("status command failed: %v\n%s", err, out)
	}

	var result map[string]any
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if result["indexed"] != false {
		t.Errorf("expected indexed=false before indexing")
	}

	// Index, then status
	if _, err := runVaultquery(t, dbPath, vault, "index"); err != nil {
		t.Fatalf("index failed: %v", err)
	}

	out, err = runVaultquery(t, dbPath, vault, "status")
	if err != nil {
		t.Fatalf("status command failed: %v\n%s", err, out)
	}

	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if result["indexed"] != true {
		t.Errorf("expected indexed=true after indexing")
	}
	if result["files"].(float64) != 4 {
		t.Errorf("expected 4 files, got %v", result["files"])
	}
}

func TestE2E_InvalidQuery(t *testing.T) {
	vault := createTestVault(t)
	dbPath := filepath.Join(t.TempDir(), "test.db")

	_, err := runVaultquery(t, dbPath, vault, "query", "INVALID QUERY SYNTAX")
	if err == nil {
		t.Error("expected error for invalid query, got nil")
	}
}

func TestE2E_MissingQueryArg(t *testing.T) {
	vault := createTestVault(t)
	dbPath := filepath.Join(t.TempDir(), "test.db")

	_, err := runVaultquery(t, dbPath, vault, "query")
	if err == nil {
		t.Error("expected error for missing query arg, got nil")
	}
}

func TestE2E_ContainsQuery(t *testing.T) {
	vault := createTestVault(t)
	dbPath := filepath.Join(t.TempDir(), "test.db")

	out, err := runVaultquery(t, dbPath, vault, "query", `TABLE customer WHERE tags contains 'linux'`)
	if err != nil {
		t.Fatalf("query failed: %v\n%s", err, out)
	}

	var result struct {
		Results []map[string]any `json:"results"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}

	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	if result.Results[0]["customer"] != "Acme Corp" {
		t.Errorf("expected 'Acme Corp', got %v", result.Results[0]["customer"])
	}
}
