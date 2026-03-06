package executor

import (
	"strings"
	"testing"

	"github.com/andinger/vaultquery/internal/dql"
	"github.com/andinger/vaultquery/internal/index"
)

// simpleFields creates []FieldDef from simple field names.
func simpleFields(names ...string) []dql.FieldDef {
	fds := make([]dql.FieldDef, len(names))
	for i, n := range names {
		fds[i] = dql.FieldDef{Expr: dql.FieldAccessExpr{Parts: []string{n}}}
	}
	return fds
}

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

	t.Cleanup(func() { _ = store.Close() })
	return store
}

// setupComplexTestStore creates a richer test store for complex expression tests.
func setupComplexTestStore(t *testing.T) *index.Store {
	t.Helper()
	store, err := index.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}

	// Initiative in project Alpha (active)
	id1, err := store.UpsertFile("Clients/Acme Corp/Alpha/Migrate DB/VORHABEN.md", 1709251200, 120, "Migrate DB")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SetFields(id1, map[string][]string{
		"type":    {"Initiative"},
		"project": {"Alpha"},
		"urgency": {"high"},
	}); err != nil {
		t.Fatal(err)
	}

	// Initiative in project Alpha (archived)
	id2, err := store.UpsertFile("Clients/Acme Corp/Alpha/_Archiv/Old Task/VORHABEN.md", 1709164800, 80, "Old Task")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SetFields(id2, map[string][]string{
		"type":    {"Initiative"},
		"project": {"Alpha"},
		"urgency": {"low"},
	}); err != nil {
		t.Fatal(err)
	}

	// Initiative in project Beta
	id3, err := store.UpsertFile("Clients/Acme Corp/Beta/Fix API/VORHABEN.md", 1709337600, 90, "Fix API")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SetFields(id3, map[string][]string{
		"type":    {"Initiative"},
		"project": {"Beta"},
		"urgency": {"normal"},
	}); err != nil {
		t.Fatal(err)
	}

	// Cluster with red status
	id4, err := store.UpsertFile("Clients/Acme Corp/Production/CLUSTER.md", 1709424000, 100, "Acme Prod Cluster")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SetFields(id4, map[string][]string{
		"type":              {"Kubernetes Cluster"},
		"kubectl_context":   {"acme-prod"},
		"cluster_status":    {"red"},
		"open_issues":       {"5"},
		"critical_issues":   {"2"},
		"last_health_check": {"2024-03-01"},
	}); err != nil {
		t.Fatal(err)
	}

	// Cluster with green status
	id5, err := store.UpsertFile("Clients/Globex Inc/Production/CLUSTER.md", 1709424000, 100, "Globex Prod Cluster")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SetFields(id5, map[string][]string{
		"type":              {"Kubernetes Cluster"},
		"kubectl_context":   {"globex-prod"},
		"cluster_status":    {"green"},
		"open_issues":       {"1"},
		"critical_issues":   {"0"},
		"last_health_check": {"2024-03-02"},
	}); err != nil {
		t.Fatal(err)
	}

	// Cluster with yellow status
	id6, err := store.UpsertFile("Clients/Initech/Staging/CLUSTER.md", 1709424000, 100, "Initech Staging Cluster")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SetFields(id6, map[string][]string{
		"type":              {"Kubernetes Cluster"},
		"kubectl_context":   {"initech-stg"},
		"cluster_status":    {"yellow"},
		"open_issues":       {"3"},
		"critical_issues":   {"0"},
		"last_health_check": {"2024-03-01"},
	}); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = store.Close() })
	return store
}

func TestExecute_TableWithSortAndWhere(t *testing.T) {
	store := setupTestStore(t)
	exec := New(store)

	q := &dql.Query{
		Mode:   "TABLE",
		Fields: simpleFields("customer", "kubectl_context"),
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
		Fields: simpleFields("customer"),
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
		Fields: simpleFields("customer"),
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

func TestExecute_TaskQuery(t *testing.T) {
	store := setupTestStore(t)
	exec := New(store)

	// Add tasks to a file
	if err := store.SetTasks(1, []index.TaskInfo{
		{Line: 10, Text: "Fix bug", Completed: false, Section: "TODO"},
		{Line: 11, Text: "Write tests", Completed: true, Section: "TODO"},
	}); err != nil {
		t.Fatal(err)
	}

	q := &dql.Query{Mode: "TASK"}
	result, err := exec.Execute(q)
	if err != nil {
		t.Fatal(err)
	}

	if result.Mode != "TASK" {
		t.Errorf("expected TASK mode, got %s", result.Mode)
	}
	if len(result.Results) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(result.Results))
	}
	if result.Results[0]["text"] != "Fix bug" {
		t.Errorf("expected 'Fix bug', got %v", result.Results[0]["text"])
	}
}

func TestExecute_TaskQueryWithWhere(t *testing.T) {
	store := setupTestStore(t)
	exec := New(store)

	if err := store.SetTasks(1, []index.TaskInfo{
		{Line: 10, Text: "Fix bug", Completed: false, Section: "TODO"},
		{Line: 11, Text: "Write tests", Completed: true, Section: "TODO"},
	}); err != nil {
		t.Fatal(err)
	}

	q := &dql.Query{
		Mode:  "TASK",
		Where: dql.ComparisonExpr{Field: "completed", Op: "=", Value: "false"},
	}
	result, err := exec.Execute(q)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Results) != 1 {
		t.Fatalf("expected 1 uncompleted task, got %d", len(result.Results))
	}
	if result.Results[0]["text"] != "Fix bug" {
		t.Errorf("expected 'Fix bug', got %v", result.Results[0]["text"])
	}
}

func TestExecute_MultiValueField(t *testing.T) {
	store := setupTestStore(t)
	exec := New(store)

	q := &dql.Query{
		Mode:   "TABLE",
		Fields: simpleFields("tags"),
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

// --- Complex expression integration tests (parse → execute) ---

func TestExecute_DateformatFileMtime(t *testing.T) {
	store := setupComplexTestStore(t)
	exec := New(store)

	q, err := dql.Parse(`TABLE dateformat(file.mtime, "dd.MM.yyyy") AS "Modified" FROM "Clients/Acme Corp/Alpha" WHERE type = "Initiative"`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	result, err := exec.Execute(q)
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	if len(result.Results) < 1 {
		t.Fatal("expected at least 1 result")
	}
	for _, row := range result.Results {
		mod, ok := row["Modified"].(string)
		if !ok {
			t.Fatalf("expected string for Modified, got %T: %v", row["Modified"], row["Modified"])
		}
		// Should be in dd.MM.yyyy format
		if len(mod) != 10 || mod[2] != '.' || mod[5] != '.' {
			t.Errorf("expected dd.MM.yyyy format, got %q", mod)
		}
	}
}

func TestExecute_RegexreplaceFileFolder(t *testing.T) {
	store := setupComplexTestStore(t)
	exec := New(store)

	q, err := dql.Parse(`TABLE WITHOUT ID regexreplace(file.folder, ".*/", "") AS "Titel", urgency AS "Priority" FROM "Clients/Acme Corp" WHERE type = "Initiative" AND project = "Alpha"`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	result, err := exec.Execute(q)
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	if len(result.Results) != 2 {
		t.Fatalf("expected 2 results (both Alpha initiatives), got %d", len(result.Results))
	}
	// Check that regexreplace extracted the last folder segment
	for _, row := range result.Results {
		titel, ok := row["Titel"].(string)
		if !ok {
			t.Errorf("expected string for Titel, got %T: %v", row["Titel"], row["Titel"])
			continue
		}
		if titel == "" {
			t.Error("expected non-empty Titel from regexreplace")
		}
		// Should be folder names like "Migrate DB" or "Old Task", not full paths
		if strings.Contains(titel, "/") {
			t.Errorf("expected folder basename, got path: %q", titel)
		}
	}
}

func TestExecute_NegatedContainsInWhere(t *testing.T) {
	store := setupComplexTestStore(t)
	exec := New(store)

	q, err := dql.Parse(`LIST FROM "Clients/Acme Corp" WHERE type = "Initiative" AND project = "Alpha" AND !contains(file.path, "_Archiv")`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	result, err := exec.Execute(q)
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	// Should filter out the archived initiative
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result (non-archived), got %d", len(result.Results))
	}
	path := result.Results[0]["path"].(string)
	if strings.Contains(path, "_Archiv") {
		t.Errorf("archived file should be excluded, got %q", path)
	}
}

func TestExecute_ChoiceWithComparison(t *testing.T) {
	store := setupComplexTestStore(t)
	exec := New(store)

	q, err := dql.Parse(`TABLE WITHOUT ID kubectl_context AS "Cluster", choice(cluster_status = "red", "Critical", choice(cluster_status = "yellow", "Warning", "Healthy")) AS "Status" WHERE type = "Kubernetes Cluster"`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	result, err := exec.Execute(q)
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	if len(result.Results) != 3 {
		t.Fatalf("expected 3 clusters, got %d", len(result.Results))
	}

	statusMap := map[string]string{}
	for _, row := range result.Results {
		cluster := row["Cluster"].(string)
		status := row["Status"].(string)
		statusMap[cluster] = status
	}

	if statusMap["acme-prod"] != "Critical" {
		t.Errorf("expected acme-prod=Critical, got %q", statusMap["acme-prod"])
	}
	if statusMap["initech-stg"] != "Warning" {
		t.Errorf("expected initech-stg=Warning, got %q", statusMap["initech-stg"])
	}
	if statusMap["globex-prod"] != "Healthy" {
		t.Errorf("expected globex-prod=Healthy, got %q", statusMap["globex-prod"])
	}
}

func TestExecute_ExpressionBasedSort(t *testing.T) {
	store := setupComplexTestStore(t)
	exec := New(store)

	q, err := dql.Parse(`TABLE WITHOUT ID kubectl_context AS "Cluster", cluster_status AS "Status" WHERE type = "Kubernetes Cluster" SORT choice(cluster_status = "red", 1, choice(cluster_status = "yellow", 2, 3)) ASC`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	result, err := exec.Execute(q)
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	if len(result.Results) != 3 {
		t.Fatalf("expected 3 clusters, got %d", len(result.Results))
	}

	// Should be sorted: red (1) first, yellow (2) second, green (3) last
	first := result.Results[0]["Status"].(string)
	second := result.Results[1]["Status"].(string)
	third := result.Results[2]["Status"].(string)

	if first != "red" {
		t.Errorf("expected red first, got %q", first)
	}
	if second != "yellow" {
		t.Errorf("expected yellow second, got %q", second)
	}
	if third != "green" {
		t.Errorf("expected green third, got %q", third)
	}
}

func TestExecute_ContainsFunctionInWhere(t *testing.T) {
	store := setupComplexTestStore(t)
	exec := New(store)

	q, err := dql.Parse(`LIST WHERE type = "Kubernetes Cluster" AND contains(file.folder, "Acme")`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	result, err := exec.Execute(q)
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result (Acme cluster), got %d", len(result.Results))
	}
	path := result.Results[0]["path"].(string)
	if !strings.Contains(path, "Acme") {
		t.Errorf("expected Acme path, got %q", path)
	}
}

func TestExecute_FileLinkField(t *testing.T) {
	store := setupComplexTestStore(t)
	exec := New(store)

	q, err := dql.Parse(`TABLE WITHOUT ID file.link AS "Link", file.name AS "Name" FROM "Clients/Acme Corp/Alpha/Migrate DB"`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	result, err := exec.Execute(q)
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	link := result.Results[0]["Link"]
	if link == nil {
		t.Fatal("expected non-nil Link")
	}
	name := result.Results[0]["Name"]
	if name == nil {
		t.Fatal("expected non-nil Name")
	}
	if name != "VORHABEN" {
		t.Errorf("expected file.name='VORHABEN', got %v", name)
	}
}

func TestExecute_MixedSimpleAndComplexFields(t *testing.T) {
	store := setupComplexTestStore(t)
	exec := New(store)

	// Mix of simple EAV field (urgency) and complex expression (dateformat)
	q, err := dql.Parse(`TABLE urgency AS "Priority", dateformat(file.mtime, "yyyy-MM-dd") AS "Modified" FROM "Clients/Acme Corp/Alpha/Migrate DB"`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	result, err := exec.Execute(q)
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	row := result.Results[0]
	if row["Priority"] != "high" {
		t.Errorf("expected Priority='high', got %v", row["Priority"])
	}
	mod, ok := row["Modified"].(string)
	if !ok || mod == "" {
		t.Errorf("expected non-empty Modified string, got %v", row["Modified"])
	}
}
