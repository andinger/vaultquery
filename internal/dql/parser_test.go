package dql

import (
	"testing"
)

func TestParseFullTableQuery(t *testing.T) {
	q, err := Parse(`TABLE customer, kubectl_context FROM "Clients" WHERE type = 'Kubernetes Cluster' SORT customer ASC`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.Mode != "TABLE" {
		t.Errorf("expected TABLE, got %s", q.Mode)
	}
	if len(q.Fields) != 2 || q.Fields[0] != "customer" || q.Fields[1] != "kubectl_context" {
		t.Errorf("unexpected fields: %v", q.Fields)
	}
	if q.From != "Clients" {
		t.Errorf("expected From 'Clients', got %q", q.From)
	}
	cmp, ok := q.Where.(ComparisonExpr)
	if !ok {
		t.Fatalf("expected ComparisonExpr, got %T", q.Where)
	}
	if cmp.Field != "type" || cmp.Op != "=" || cmp.Value != "Kubernetes Cluster" {
		t.Errorf("unexpected comparison: %+v", cmp)
	}
	if len(q.Sort) != 1 || q.Sort[0].Field != "customer" || q.Sort[0].Desc {
		t.Errorf("unexpected sort: %v", q.Sort)
	}
}

func TestParseListWithAndWhere(t *testing.T) {
	q, err := Parse(`LIST FROM "Sales" WHERE type = 'Lead' AND status != 'lost'`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.Mode != "LIST" {
		t.Errorf("expected LIST, got %s", q.Mode)
	}
	if q.From != "Sales" {
		t.Errorf("expected From 'Sales', got %q", q.From)
	}
	logical, ok := q.Where.(LogicalExpr)
	if !ok {
		t.Fatalf("expected LogicalExpr, got %T", q.Where)
	}
	if logical.Op != "AND" {
		t.Errorf("expected AND, got %s", logical.Op)
	}
}

func TestParseContains(t *testing.T) {
	q, err := Parse(`TABLE customer FROM "Clients" WHERE tags contains 'linux'`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cmp, ok := q.Where.(ComparisonExpr)
	if !ok {
		t.Fatalf("expected ComparisonExpr, got %T", q.Where)
	}
	if cmp.Op != "contains" {
		t.Errorf("expected op 'contains', got %q", cmp.Op)
	}
	if cmp.Value != "linux" {
		t.Errorf("expected value 'linux', got %q", cmp.Value)
	}
}

func TestParseBareList(t *testing.T) {
	q, err := Parse("LIST")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.Mode != "LIST" {
		t.Errorf("expected LIST, got %s", q.Mode)
	}
	if q.From != "" || q.Where != nil || q.Sort != nil {
		t.Errorf("expected empty clauses, got From=%q Where=%v Sort=%v", q.From, q.Where, q.Sort)
	}
}

func TestParseTableNoFrom(t *testing.T) {
	q, err := Parse("TABLE name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.Mode != "TABLE" || len(q.Fields) != 1 || q.Fields[0] != "name" {
		t.Errorf("unexpected result: %+v", q)
	}
	if q.From != "" {
		t.Errorf("expected empty From, got %q", q.From)
	}
}

func TestParseListFromOnly(t *testing.T) {
	q, err := Parse(`LIST FROM "path"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.Mode != "LIST" || q.From != "path" {
		t.Errorf("unexpected result: %+v", q)
	}
}

func TestParseListWhereOnly(t *testing.T) {
	q, err := Parse("LIST WHERE status = 'active'")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.Mode != "LIST" || q.From != "" {
		t.Errorf("unexpected result: %+v", q)
	}
	if q.Where == nil {
		t.Fatal("expected WHERE clause")
	}
}

func TestParseOrExpression(t *testing.T) {
	q, err := Parse("LIST WHERE status = 'active' OR status = 'pending'")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	logical, ok := q.Where.(LogicalExpr)
	if !ok {
		t.Fatalf("expected LogicalExpr, got %T", q.Where)
	}
	if logical.Op != "OR" {
		t.Errorf("expected OR, got %s", logical.Op)
	}
}

func TestParseParensWithAndOr(t *testing.T) {
	q, err := Parse("LIST WHERE (type = 'A' OR type = 'B') AND status = 'active'")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	logical, ok := q.Where.(LogicalExpr)
	if !ok {
		t.Fatalf("expected LogicalExpr, got %T", q.Where)
	}
	if logical.Op != "AND" {
		t.Errorf("expected AND at top level, got %s", logical.Op)
	}
	paren, ok := logical.Left.(ParenExpr)
	if !ok {
		t.Fatalf("expected ParenExpr on left, got %T", logical.Left)
	}
	inner, ok := paren.Inner.(LogicalExpr)
	if !ok {
		t.Fatalf("expected LogicalExpr inside parens, got %T", paren.Inner)
	}
	if inner.Op != "OR" {
		t.Errorf("expected OR inside parens, got %s", inner.Op)
	}
}

func TestParseExists(t *testing.T) {
	q, err := Parse("LIST WHERE status exists")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ex, ok := q.Where.(ExistsExpr)
	if !ok {
		t.Fatalf("expected ExistsExpr, got %T", q.Where)
	}
	if ex.Field != "status" || ex.Negated {
		t.Errorf("unexpected exists: %+v", ex)
	}
}

func TestParseNotExists(t *testing.T) {
	q, err := Parse("LIST WHERE status !exists")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ex, ok := q.Where.(ExistsExpr)
	if !ok {
		t.Fatalf("expected ExistsExpr, got %T", q.Where)
	}
	if !ex.Negated {
		t.Error("expected Negated=true")
	}
}

func TestParseNotContains(t *testing.T) {
	q, err := Parse("LIST WHERE tags !contains 'linux'")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cmp, ok := q.Where.(ComparisonExpr)
	if !ok {
		t.Fatalf("expected ComparisonExpr, got %T", q.Where)
	}
	if cmp.Op != "!contains" {
		t.Errorf("expected op '!contains', got %q", cmp.Op)
	}
}

func TestParseAllComparisonOperators(t *testing.T) {
	ops := []struct {
		sym string
		op  string
	}{
		{"=", "="},
		{"!=", "!="},
		{"<", "<"},
		{">", ">"},
		{"<=", "<="},
		{">=", ">="},
	}
	for _, tt := range ops {
		t.Run(tt.sym, func(t *testing.T) {
			q, err := Parse("LIST WHERE age " + tt.sym + " 30")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			cmp, ok := q.Where.(ComparisonExpr)
			if !ok {
				t.Fatalf("expected ComparisonExpr, got %T", q.Where)
			}
			if cmp.Op != tt.op {
				t.Errorf("expected op %q, got %q", tt.op, cmp.Op)
			}
		})
	}
}

func TestParseSortMultipleFields(t *testing.T) {
	q, err := Parse("LIST SORT name ASC, date DESC, status")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(q.Sort) != 3 {
		t.Fatalf("expected 3 sort fields, got %d", len(q.Sort))
	}
	if q.Sort[0].Field != "name" || q.Sort[0].Desc {
		t.Errorf("sort[0]: %+v", q.Sort[0])
	}
	if q.Sort[1].Field != "date" || !q.Sort[1].Desc {
		t.Errorf("sort[1]: %+v", q.Sort[1])
	}
	if q.Sort[2].Field != "status" || q.Sort[2].Desc {
		t.Errorf("sort[2]: %+v", q.Sort[2])
	}
}

func TestParseLimit(t *testing.T) {
	q, err := Parse("LIST LIMIT 10")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.Limit != 10 {
		t.Errorf("expected limit 10, got %d", q.Limit)
	}
}

func TestParseGroupBy(t *testing.T) {
	q, err := Parse("TABLE name GROUP BY category")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(q.GroupBy) != 1 || q.GroupBy[0] != "category" {
		t.Errorf("unexpected GroupBy: %v", q.GroupBy)
	}
}

func TestParseFlatten(t *testing.T) {
	q, err := Parse("TABLE name FLATTEN tags")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(q.Flatten) != 1 || q.Flatten[0] != "tags" {
		t.Errorf("unexpected Flatten: %v", q.Flatten)
	}
}

func TestParseErrorMissingMode(t *testing.T) {
	_, err := Parse("FROM \"test\"")
	if err == nil {
		t.Fatal("expected error for missing query mode")
	}
	pe, ok := err.(*ParseError)
	if !ok {
		t.Fatalf("expected ParseError, got %T: %v", err, err)
	}
	if pe.Pos != 0 {
		t.Errorf("expected pos 0, got %d", pe.Pos)
	}
}

func TestParseErrorUnterminatedString(t *testing.T) {
	_, err := Parse(`LIST FROM "unterminated`)
	if err == nil {
		t.Fatal("expected error for unterminated string")
	}
}

func TestParseErrorMissingValueAfterOp(t *testing.T) {
	_, err := Parse("LIST WHERE name =")
	if err == nil {
		t.Fatal("expected error for missing value after operator")
	}
}

func TestParseErrorUnexpectedToken(t *testing.T) {
	_, err := Parse("LIST WHERE =")
	if err == nil {
		t.Fatal("expected error for unexpected token")
	}
}

func TestParseComplexQuery(t *testing.T) {
	q, err := Parse(`TABLE name, status FROM "Projects" WHERE (status = 'active' OR status = 'pending') AND priority >= 3 SORT priority DESC, name ASC LIMIT 20`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.Mode != "TABLE" {
		t.Errorf("expected TABLE, got %s", q.Mode)
	}
	if len(q.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(q.Fields))
	}
	if q.From != "Projects" {
		t.Errorf("expected From 'Projects', got %q", q.From)
	}
	if q.Limit != 20 {
		t.Errorf("expected limit 20, got %d", q.Limit)
	}
	if len(q.Sort) != 2 {
		t.Errorf("expected 2 sort fields, got %d", len(q.Sort))
	}
}

func TestParseGroupByMultiple(t *testing.T) {
	q, err := Parse("TABLE name GROUP BY category, type")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(q.GroupBy) != 2 || q.GroupBy[0] != "category" || q.GroupBy[1] != "type" {
		t.Errorf("unexpected GroupBy: %v", q.GroupBy)
	}
}

func TestParseFlattenMultiple(t *testing.T) {
	q, err := Parse("TABLE name FLATTEN tags, aliases")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(q.Flatten) != 2 || q.Flatten[0] != "tags" || q.Flatten[1] != "aliases" {
		t.Errorf("unexpected Flatten: %v", q.Flatten)
	}
}

func TestParseExistsAndComparison(t *testing.T) {
	q, err := Parse("LIST WHERE status exists AND type = 'A'")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	logical, ok := q.Where.(LogicalExpr)
	if !ok {
		t.Fatalf("expected LogicalExpr, got %T", q.Where)
	}
	if _, ok := logical.Left.(ExistsExpr); !ok {
		t.Errorf("expected ExistsExpr on left, got %T", logical.Left)
	}
	if _, ok := logical.Right.(ComparisonExpr); !ok {
		t.Errorf("expected ComparisonExpr on right, got %T", logical.Right)
	}
}

func TestParseCaseInsensitiveKeywords(t *testing.T) {
	q, err := Parse(`table name from "test" where status = 'active' sort name asc limit 5`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.Mode != "TABLE" {
		t.Errorf("expected TABLE, got %s", q.Mode)
	}
	if q.From != "test" {
		t.Errorf("expected 'test', got %q", q.From)
	}
	if q.Limit != 5 {
		t.Errorf("expected limit 5, got %d", q.Limit)
	}
}
