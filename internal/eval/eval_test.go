package eval

import (
	"testing"

	"github.com/andinger/vaultquery/internal/dql"
)

func testCtx() *EvalContext {
	return NewEvalContext("Clients/Acme Corp/CLUSTER.md", "Acme Cluster", map[string]dql.Value{
		"type":     dql.NewString("Kubernetes Cluster"),
		"customer": dql.NewString("Acme Corp"),
		"status":   dql.NewString("active"),
		"priority": dql.NewNumber(3),
		"tags":     dql.NewList([]dql.Value{dql.NewString("linux"), dql.NewString("prod")}),
		"meta": dql.NewObject(map[string]dql.Value{
			"region": dql.NewString("eu-west-1"),
		}),
	})
}

func TestEvalLiteral(t *testing.T) {
	ev := New()
	ctx := testCtx()

	v := ev.Eval(dql.LiteralExpr{Val: dql.NewNumber(42)}, ctx)
	if n, ok := v.AsNumber(); !ok || n != 42 {
		t.Errorf("expected 42, got %v", v)
	}
}

func TestEvalFieldAccess(t *testing.T) {
	ev := New()
	ctx := testCtx()

	v := ev.Eval(dql.FieldAccessExpr{Parts: []string{"customer"}}, ctx)
	if s, ok := v.AsString(); !ok || s != "Acme Corp" {
		t.Errorf("expected 'Acme Corp', got %v", v)
	}
}

func TestEvalFieldAccessNested(t *testing.T) {
	ev := New()
	ctx := testCtx()

	v := ev.Eval(dql.FieldAccessExpr{Parts: []string{"meta", "region"}}, ctx)
	if s, ok := v.AsString(); !ok || s != "eu-west-1" {
		t.Errorf("expected 'eu-west-1', got %v", v)
	}
}

func TestEvalFieldAccessMissing(t *testing.T) {
	ev := New()
	ctx := testCtx()

	v := ev.Eval(dql.FieldAccessExpr{Parts: []string{"nonexistent"}}, ctx)
	if !v.IsNull() {
		t.Errorf("expected null, got %v", v)
	}
}

func TestEvalComparisonEquals(t *testing.T) {
	ev := New()
	ctx := testCtx()

	v := ev.Eval(dql.ComparisonExpr{Field: "type", Op: "=", Value: "Kubernetes Cluster"}, ctx)
	if b, ok := v.AsBool(); !ok || !b {
		t.Errorf("expected true, got %v", v)
	}

	v = ev.Eval(dql.ComparisonExpr{Field: "type", Op: "=", Value: "VM"}, ctx)
	if b, ok := v.AsBool(); !ok || b {
		t.Errorf("expected false, got %v", v)
	}
}

func TestEvalComparisonNotEquals(t *testing.T) {
	ev := New()
	ctx := testCtx()

	v := ev.Eval(dql.ComparisonExpr{Field: "status", Op: "!=", Value: "lost"}, ctx)
	if b, ok := v.AsBool(); !ok || !b {
		t.Errorf("expected true, got %v", v)
	}
}

func TestEvalComparisonNumeric(t *testing.T) {
	ev := New()
	ctx := testCtx()

	v := ev.Eval(dql.ComparisonExpr{Field: "priority", Op: ">=", Value: "3"}, ctx)
	if b, ok := v.AsBool(); !ok || !b {
		t.Errorf("expected true for priority >= 3, got %v", v)
	}

	v = ev.Eval(dql.ComparisonExpr{Field: "priority", Op: ">", Value: "3"}, ctx)
	if b, ok := v.AsBool(); !ok || b {
		t.Errorf("expected false for priority > 3, got %v", v)
	}
}

func TestEvalContains(t *testing.T) {
	ev := New()
	ctx := testCtx()

	v := ev.Eval(dql.ComparisonExpr{Field: "tags", Op: "contains", Value: "linux"}, ctx)
	if b, ok := v.AsBool(); !ok || !b {
		t.Errorf("expected true for tags contains 'linux', got %v", v)
	}

	v = ev.Eval(dql.ComparisonExpr{Field: "tags", Op: "contains", Value: "windows"}, ctx)
	if b, ok := v.AsBool(); !ok || b {
		t.Errorf("expected false for tags contains 'windows', got %v", v)
	}

	v = ev.Eval(dql.ComparisonExpr{Field: "tags", Op: "!contains", Value: "linux"}, ctx)
	if b, ok := v.AsBool(); !ok || b {
		t.Errorf("expected false for tags !contains 'linux', got %v", v)
	}
}

func TestEvalLogicalAnd(t *testing.T) {
	ev := New()
	ctx := testCtx()

	expr := dql.LogicalExpr{
		Op:    "AND",
		Left:  dql.ComparisonExpr{Field: "type", Op: "=", Value: "Kubernetes Cluster"},
		Right: dql.ComparisonExpr{Field: "status", Op: "=", Value: "active"},
	}
	if !ev.EvalBool(expr, ctx) {
		t.Error("expected AND to be true")
	}
}

func TestEvalLogicalOr(t *testing.T) {
	ev := New()
	ctx := testCtx()

	expr := dql.LogicalExpr{
		Op:    "OR",
		Left:  dql.ComparisonExpr{Field: "type", Op: "=", Value: "VM"},
		Right: dql.ComparisonExpr{Field: "status", Op: "=", Value: "active"},
	}
	if !ev.EvalBool(expr, ctx) {
		t.Error("expected OR to be true")
	}
}

func TestEvalExists(t *testing.T) {
	ev := New()
	ctx := testCtx()

	v := ev.Eval(dql.ExistsExpr{Field: "status", Negated: false}, ctx)
	if b, ok := v.AsBool(); !ok || !b {
		t.Errorf("expected status exists = true, got %v", v)
	}

	v = ev.Eval(dql.ExistsExpr{Field: "nonexistent", Negated: false}, ctx)
	if b, ok := v.AsBool(); !ok || b {
		t.Errorf("expected nonexistent exists = false, got %v", v)
	}

	v = ev.Eval(dql.ExistsExpr{Field: "status", Negated: true}, ctx)
	if b, ok := v.AsBool(); !ok || b {
		t.Errorf("expected status !exists = false, got %v", v)
	}
}

func TestEvalArithmetic(t *testing.T) {
	ev := New()
	ctx := testCtx()

	expr := dql.ArithmeticExpr{
		Op:    "+",
		Left:  dql.LiteralExpr{Val: dql.NewNumber(10)},
		Right: dql.LiteralExpr{Val: dql.NewNumber(5)},
	}
	v := ev.Eval(expr, ctx)
	if n, ok := v.AsNumber(); !ok || n != 15 {
		t.Errorf("expected 15, got %v", v)
	}
}

func TestEvalNegation(t *testing.T) {
	ev := New()
	ctx := testCtx()

	expr := dql.NegationExpr{
		Inner: dql.ComparisonExpr{Field: "status", Op: "=", Value: "active"},
	}
	v := ev.Eval(expr, ctx)
	if b, ok := v.AsBool(); !ok || b {
		t.Errorf("expected !true = false, got %v", v)
	}
}

func TestEvalFunctionCall(t *testing.T) {
	ev := New()
	ev.RegisterFunc("length", func(args []dql.Value, ctx *EvalContext) dql.Value {
		if len(args) == 0 {
			return dql.NewNumber(0)
		}
		if items, ok := args[0].AsList(); ok {
			return dql.NewNumber(float64(len(items)))
		}
		if s, ok := args[0].AsString(); ok {
			return dql.NewNumber(float64(len(s)))
		}
		return dql.NewNumber(0)
	})

	ctx := testCtx()
	expr := dql.FunctionCallExpr{
		Name: "length",
		Args: []dql.Expr{dql.FieldAccessExpr{Parts: []string{"tags"}}},
	}
	v := ev.Eval(expr, ctx)
	if n, ok := v.AsNumber(); !ok || n != 2 {
		t.Errorf("expected length(tags) = 2, got %v", v)
	}
}

func TestEvalIndexExpr(t *testing.T) {
	ev := New()
	ctx := testCtx()

	expr := dql.IndexExpr{
		Object: dql.FieldAccessExpr{Parts: []string{"tags"}},
		Index:  dql.LiteralExpr{Val: dql.NewNumber(0)},
	}
	v := ev.Eval(expr, ctx)
	if s, ok := v.AsString(); !ok || s != "linux" {
		t.Errorf("expected tags[0] = 'linux', got %v", v)
	}

	// Negative index
	expr.Index = dql.LiteralExpr{Val: dql.NewNumber(-1)}
	v = ev.Eval(expr, ctx)
	if s, ok := v.AsString(); !ok || s != "prod" {
		t.Errorf("expected tags[-1] = 'prod', got %v", v)
	}

	// Out of bounds
	expr.Index = dql.LiteralExpr{Val: dql.NewNumber(99)}
	v = ev.Eval(expr, ctx)
	if !v.IsNull() {
		t.Errorf("expected null for out of bounds, got %v", v)
	}
}

func TestEvalLambda(t *testing.T) {
	ev := New()
	ctx := testCtx()

	lambda := dql.LambdaExpr{
		Params: []string{"x"},
		Body: dql.ArithmeticExpr{
			Op:    "*",
			Left:  dql.FieldAccessExpr{Parts: []string{"x"}},
			Right: dql.LiteralExpr{Val: dql.NewNumber(2)},
		},
	}

	result := ev.EvalLambda(lambda, []dql.Value{dql.NewNumber(5)}, ctx)
	if n, ok := result.AsNumber(); !ok || n != 10 {
		t.Errorf("expected lambda(5) = 10, got %v", result)
	}
}

func TestBuildEvalContextFromEAV(t *testing.T) {
	ctx := BuildEvalContextFromEAV("path.md", "Title", map[string][]string{
		"name":   {"Acme"},
		"tags":   {"a", "b", "c"},
		"count":  {"42"},
		"active": {"true"},
	})

	if s, ok := ctx.Fields["name"].AsString(); !ok || s != "Acme" {
		t.Errorf("expected 'Acme', got %v", ctx.Fields["name"])
	}
	if items, ok := ctx.Fields["tags"].AsList(); !ok || len(items) != 3 {
		t.Errorf("expected list of 3, got %v", ctx.Fields["tags"])
	}
	if n, ok := ctx.Fields["count"].AsNumber(); !ok || n != 42 {
		t.Errorf("expected 42, got %v", ctx.Fields["count"])
	}
	if b, ok := ctx.Fields["active"].AsBool(); !ok || !b {
		t.Errorf("expected true, got %v", ctx.Fields["active"])
	}
}

func TestEvalVariableScope(t *testing.T) {
	ev := New()
	ctx := testCtx()

	// Outer scope
	ctx.PushScope(map[string]dql.Value{"x": dql.NewNumber(10)})

	v := ev.Eval(dql.FieldAccessExpr{Parts: []string{"x"}}, ctx)
	if n, ok := v.AsNumber(); !ok || n != 10 {
		t.Errorf("expected x=10 from outer scope, got %v", v)
	}

	// Inner scope shadows
	ctx.PushScope(map[string]dql.Value{"x": dql.NewNumber(20)})
	v = ev.Eval(dql.FieldAccessExpr{Parts: []string{"x"}}, ctx)
	if n, ok := v.AsNumber(); !ok || n != 20 {
		t.Errorf("expected x=20 from inner scope, got %v", v)
	}

	ctx.PopScope()
	v = ev.Eval(dql.FieldAccessExpr{Parts: []string{"x"}}, ctx)
	if n, ok := v.AsNumber(); !ok || n != 10 {
		t.Errorf("expected x=10 after pop, got %v", v)
	}

	ctx.PopScope()
}

func TestEvalParenExpr(t *testing.T) {
	ev := New()
	ctx := testCtx()

	expr := dql.ParenExpr{
		Inner: dql.ComparisonExpr{Field: "status", Op: "=", Value: "active"},
	}
	v := ev.Eval(expr, ctx)
	if b, ok := v.AsBool(); !ok || !b {
		t.Errorf("expected true, got %v", v)
	}
}

func TestEvalUnknownFunction(t *testing.T) {
	ev := New()
	ctx := testCtx()

	v := ev.Eval(dql.FunctionCallExpr{Name: "unknown", Args: nil}, ctx)
	if !v.IsNull() {
		t.Errorf("expected null for unknown function, got %v", v)
	}
}

func TestEvalObjectIndexing(t *testing.T) {
	ev := New()
	ctx := testCtx()

	expr := dql.IndexExpr{
		Object: dql.FieldAccessExpr{Parts: []string{"meta"}},
		Index:  dql.LiteralExpr{Val: dql.NewString("region")},
	}
	v := ev.Eval(expr, ctx)
	if s, ok := v.AsString(); !ok || s != "eu-west-1" {
		t.Errorf("expected 'eu-west-1', got %v", v)
	}
}
