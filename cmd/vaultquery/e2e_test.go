package main_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func runVaultquery(t *testing.T, vaultRoot string, args ...string) ([]byte, error) {
	t.Helper()
	fullArgs := append([]string{"--vault", vaultRoot}, args...)
	cmd := exec.Command(binaryPath, fullArgs...)
	return cmd.CombinedOutput()
}

func TestE2E_IndexCommand(t *testing.T) {
	vault := createTestVault(t)

	out, err := runVaultquery(t, vault, "index")
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

	// Verify DB was created inside vault
	dbPath := filepath.Join(vault, ".vaultquery", "index.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("expected .vaultquery/index.db to be created inside vault")
	}

	// Verify .gitignore was created
	gitignore := filepath.Join(vault, ".vaultquery", ".gitignore")
	if _, err := os.Stat(gitignore); os.IsNotExist(err) {
		t.Error("expected .vaultquery/.gitignore to be created")
	}
}

func TestE2E_QueryCommand(t *testing.T) {
	vault := createTestVault(t)

	// Index first, then query (query no longer auto-syncs)
	if _, err := runVaultquery(t, vault, "index"); err != nil {
		t.Fatalf("index failed: %v", err)
	}

	out, err := runVaultquery(t, vault, "query", `LIST WHERE type = 'Kubernetes Cluster'`)
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

	if _, err := runVaultquery(t, vault, "index"); err != nil {
		t.Fatalf("index failed: %v", err)
	}

	out, err := runVaultquery(t, vault, "query",
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

	// First index
	if _, err := runVaultquery(t, vault, "index"); err != nil {
		t.Fatalf("index failed: %v", err)
	}

	// Reindex
	out, err := runVaultquery(t, vault, "reindex")
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

	// Status before index
	out, err := runVaultquery(t, vault, "status")
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
	if _, err := runVaultquery(t, vault, "index"); err != nil {
		t.Fatalf("index failed: %v", err)
	}

	out, err = runVaultquery(t, vault, "status")
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
	// Verify db_path points to vault-local location
	dbPath, _ := result["db_path"].(string)
	expected := filepath.Join(vault, ".vaultquery", "index.db")
	if dbPath != expected {
		t.Errorf("db_path = %q, want %q", dbPath, expected)
	}
}

func TestE2E_InvalidQuery(t *testing.T) {
	vault := createTestVault(t)

	_, err := runVaultquery(t, vault, "query", "INVALID QUERY SYNTAX")
	if err == nil {
		t.Error("expected error for invalid query, got nil")
	}
}

func TestE2E_MissingQueryArg(t *testing.T) {
	vault := createTestVault(t)

	_, err := runVaultquery(t, vault, "query")
	if err == nil {
		t.Error("expected error for missing query arg, got nil")
	}
}

func TestE2E_QuerySyncFlag(t *testing.T) {
	vault := createTestVault(t)

	// --sync should index and query in one step (no prior index needed)
	out, err := runVaultquery(t, vault, "query", "--sync", `LIST WHERE type = 'Kubernetes Cluster'`)
	if err != nil {
		t.Fatalf("query --sync failed: %v\n%s", err, out)
	}

	var result struct {
		Results []map[string]any `json:"results"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if len(result.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(result.Results))
	}
}

func TestE2E_QueryAutoSyncsFirstRun(t *testing.T) {
	vault := createTestVault(t)

	// No prior index — query should auto-sync on first run (DB doesn't exist)
	out, err := runVaultquery(t, vault, "query", `LIST WHERE type = 'Kubernetes Cluster'`)
	if err != nil {
		t.Fatalf("query (first run) failed: %v\n%s", err, out)
	}

	var result struct {
		Results []map[string]any `json:"results"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if len(result.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(result.Results))
	}
}

func TestE2E_QuerySkipsSyncOnExistingIndex(t *testing.T) {
	vault := createTestVault(t)

	// Index first
	if _, err := runVaultquery(t, vault, "index"); err != nil {
		t.Fatalf("index failed: %v", err)
	}

	// Add a new file after indexing
	newFile := filepath.Join(vault, "Sales/NewLead/LEAD.md")
	if err := os.MkdirAll(filepath.Dir(newFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newFile, []byte("---\ntype: Lead\nstatus: new\n---\n# New Lead\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Query without --sync should NOT see the new file
	out, err := runVaultquery(t, vault, "query", `LIST WHERE type = 'Lead'`)
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
		t.Errorf("expected 1 result (no sync), got %d", len(result.Results))
	}

	// Query with --sync SHOULD see the new file
	out, err = runVaultquery(t, vault, "query", "--sync", `LIST WHERE type = 'Lead'`)
	if err != nil {
		t.Fatalf("query --sync failed: %v\n%s", err, out)
	}

	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if len(result.Results) != 2 {
		t.Errorf("expected 2 results (after sync), got %d", len(result.Results))
	}
}

func TestE2E_ContainsQuery(t *testing.T) {
	vault := createTestVault(t)

	if _, err := runVaultquery(t, vault, "index"); err != nil {
		t.Fatalf("index failed: %v", err)
	}

	out, err := runVaultquery(t, vault, "query", `TABLE customer WHERE tags contains 'linux'`)
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

func TestE2E_ExcludeFolders(t *testing.T) {
	vault := createTestVault(t)

	// Create config that excludes Sales folder
	configDir := filepath.Join(vault, ".vaultquery")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	configFile := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte("exclude:\n  - Sales\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runVaultquery(t, vault, "index")
	if err != nil {
		t.Fatalf("index failed: %v\n%s", err, out)
	}

	var result map[string]any
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}

	// 4 total files minus 1 in Sales = 3
	files, ok := result["files"].(float64)
	if !ok || files != 3 {
		t.Errorf("expected 3 files (Sales excluded), got %v", result["files"])
	}
}

func TestE2E_QueryFormatTOON(t *testing.T) {
	vault := createTestVault(t)

	if _, err := runVaultquery(t, vault, "index"); err != nil {
		t.Fatalf("index failed: %v", err)
	}

	out, err := runVaultquery(t, vault, "query", "--format", "toon", `LIST WHERE type = 'Kubernetes Cluster'`)
	if err != nil {
		t.Fatalf("query --format toon failed: %v\n%s", err, out)
	}

	// Output should NOT be valid JSON
	var jsonCheck map[string]any
	if json.Unmarshal(out, &jsonCheck) == nil {
		t.Error("expected non-JSON output for TOON format, but got valid JSON")
	}

	// Output should contain the mode
	if !contains(string(out), "LIST") {
		t.Errorf("TOON output should contain 'LIST', got: %s", out)
	}
}

func TestE2E_QueryFormatFromConfig(t *testing.T) {
	vault := createTestVault(t)

	// Set format: toon in config
	configDir := filepath.Join(vault, ".vaultquery")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte("format: toon\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := runVaultquery(t, vault, "index"); err != nil {
		t.Fatalf("index failed: %v", err)
	}

	// Query without --format flag should use config (toon)
	out, err := runVaultquery(t, vault, "query", `LIST WHERE type = 'Lead'`)
	if err != nil {
		t.Fatalf("query failed: %v\n%s", err, out)
	}

	var jsonCheck map[string]any
	if json.Unmarshal(out, &jsonCheck) == nil {
		t.Error("expected non-JSON output when config sets toon, but got valid JSON")
	}
}

func TestE2E_QueryFormatFlagOverridesConfig(t *testing.T) {
	vault := createTestVault(t)

	// Set format: toon in config
	configDir := filepath.Join(vault, ".vaultquery")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte("format: toon\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := runVaultquery(t, vault, "index"); err != nil {
		t.Fatalf("index failed: %v", err)
	}

	// --format json should override config
	out, err := runVaultquery(t, vault, "query", "--format", "json", `LIST WHERE type = 'Lead'`)
	if err != nil {
		t.Fatalf("query --format json failed: %v\n%s", err, out)
	}

	var result struct {
		Mode    string           `json:"mode"`
		Results []map[string]any `json:"results"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("expected valid JSON when --format json overrides config: %v\n%s", err, out)
	}
	if result.Mode != "LIST" {
		t.Errorf("expected LIST mode, got %s", result.Mode)
	}
}

func TestE2E_QueryFormatInvalid(t *testing.T) {
	vault := createTestVault(t)

	_, err := runVaultquery(t, vault, "query", "--format", "xml", `LIST`)
	if err == nil {
		t.Error("expected error for invalid format, got nil")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && strings.Contains(s, substr)
}

func TestE2E_VaultqueryDirNotIndexed(t *testing.T) {
	vault := createTestVault(t)

	// Put a .md file in .vaultquery to ensure it's not indexed
	configDir := filepath.Join(vault, ".vaultquery")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "notes.md"), []byte("---\ntype: Internal\n---\n# Notes\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runVaultquery(t, vault, "index")
	if err != nil {
		t.Fatalf("index failed: %v\n%s", err, out)
	}

	var result map[string]any
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}

	// Should be 4 files (the .vaultquery/notes.md should NOT be counted)
	files, ok := result["files"].(float64)
	if !ok || files != 4 {
		t.Errorf("expected 4 files (.vaultquery excluded), got %v", result["files"])
	}
}
