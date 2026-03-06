package eval

import (
	"testing"

	"github.com/andinger/vaultquery/internal/dql"
)

func setupEval() *Evaluator {
	ev := New()
	RegisterBuiltins(ev)
	return ev
}

func TestFnLength(t *testing.T) {
	ev := setupEval()
	ctx := testCtx()

	// List length
	v := ev.Eval(dql.FunctionCallExpr{
		Name: "length",
		Args: []dql.Expr{dql.FieldAccessExpr{Parts: []string{"tags"}}},
	}, ctx)
	if n, ok := v.AsNumber(); !ok || n != 2 {
		t.Errorf("expected 2, got %v", v)
	}

	// String length
	v = ev.Eval(dql.FunctionCallExpr{
		Name: "length",
		Args: []dql.Expr{dql.LiteralExpr{Val: dql.NewString("hello")}},
	}, ctx)
	if n, ok := v.AsNumber(); !ok || n != 5 {
		t.Errorf("expected 5, got %v", v)
	}
}

func TestFnLowerUpper(t *testing.T) {
	ev := setupEval()
	ctx := testCtx()

	v := ev.Eval(dql.FunctionCallExpr{
		Name: "lower",
		Args: []dql.Expr{dql.LiteralExpr{Val: dql.NewString("Hello World")}},
	}, ctx)
	if s, ok := v.AsString(); !ok || s != "hello world" {
		t.Errorf("lower: %v", v)
	}

	v = ev.Eval(dql.FunctionCallExpr{
		Name: "upper",
		Args: []dql.Expr{dql.LiteralExpr{Val: dql.NewString("hello")}},
	}, ctx)
	if s, ok := v.AsString(); !ok || s != "HELLO" {
		t.Errorf("upper: %v", v)
	}
}

func TestFnContains(t *testing.T) {
	ev := setupEval()
	ctx := testCtx()

	v := ev.Eval(dql.FunctionCallExpr{
		Name: "contains",
		Args: []dql.Expr{
			dql.FieldAccessExpr{Parts: []string{"tags"}},
			dql.LiteralExpr{Val: dql.NewString("linux")},
		},
	}, ctx)
	if b, ok := v.AsBool(); !ok || !b {
		t.Errorf("expected true, got %v", v)
	}
}

func TestFnDefault(t *testing.T) {
	ev := setupEval()
	ctx := testCtx()

	// Non-null returns first
	v := ev.Eval(dql.FunctionCallExpr{
		Name: "default",
		Args: []dql.Expr{
			dql.LiteralExpr{Val: dql.NewString("value")},
			dql.LiteralExpr{Val: dql.NewString("fallback")},
		},
	}, ctx)
	if s, ok := v.AsString(); !ok || s != "value" {
		t.Errorf("expected 'value', got %v", v)
	}

	// Null returns default
	v = ev.Eval(dql.FunctionCallExpr{
		Name: "default",
		Args: []dql.Expr{
			dql.FieldAccessExpr{Parts: []string{"nonexistent"}},
			dql.LiteralExpr{Val: dql.NewString("fallback")},
		},
	}, ctx)
	if s, ok := v.AsString(); !ok || s != "fallback" {
		t.Errorf("expected 'fallback', got %v", v)
	}
}

func TestFnChoice(t *testing.T) {
	ev := setupEval()
	ctx := testCtx()

	v := ev.Eval(dql.FunctionCallExpr{
		Name: "choice",
		Args: []dql.Expr{
			dql.LiteralExpr{Val: dql.NewBool(true)},
			dql.LiteralExpr{Val: dql.NewString("yes")},
			dql.LiteralExpr{Val: dql.NewString("no")},
		},
	}, ctx)
	if s, ok := v.AsString(); !ok || s != "yes" {
		t.Errorf("expected 'yes', got %v", v)
	}
}

func TestFnNumeric(t *testing.T) {
	ev := setupEval()
	ctx := testCtx()

	tests := []struct {
		name string
		args []dql.Expr
		want float64
	}{
		{"round", []dql.Expr{dql.LiteralExpr{Val: dql.NewNumber(3.7)}}, 4},
		{"floor", []dql.Expr{dql.LiteralExpr{Val: dql.NewNumber(3.7)}}, 3},
		{"ceil", []dql.Expr{dql.LiteralExpr{Val: dql.NewNumber(3.2)}}, 4},
		{"sum", []dql.Expr{dql.LiteralExpr{Val: dql.NewList([]dql.Value{dql.NewNumber(1), dql.NewNumber(2), dql.NewNumber(3)})}}, 6},
		{"average", []dql.Expr{dql.LiteralExpr{Val: dql.NewList([]dql.Value{dql.NewNumber(2), dql.NewNumber(4)})}}, 3},
		{"min", []dql.Expr{dql.LiteralExpr{Val: dql.NewList([]dql.Value{dql.NewNumber(5), dql.NewNumber(2), dql.NewNumber(8)})}}, 2},
		{"max", []dql.Expr{dql.LiteralExpr{Val: dql.NewList([]dql.Value{dql.NewNumber(5), dql.NewNumber(2), dql.NewNumber(8)})}}, 8},
		{"product", []dql.Expr{dql.LiteralExpr{Val: dql.NewList([]dql.Value{dql.NewNumber(2), dql.NewNumber(3), dql.NewNumber(4)})}}, 24},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := ev.Eval(dql.FunctionCallExpr{Name: tt.name, Args: tt.args}, ctx)
			if n, ok := v.AsNumber(); !ok || n != tt.want {
				t.Errorf("expected %v, got %v", tt.want, v)
			}
		})
	}
}

func TestFnJoin(t *testing.T) {
	ev := setupEval()
	ctx := testCtx()

	v := ev.Eval(dql.FunctionCallExpr{
		Name: "join",
		Args: []dql.Expr{
			dql.LiteralExpr{Val: dql.NewList([]dql.Value{dql.NewString("a"), dql.NewString("b"), dql.NewString("c")})},
			dql.LiteralExpr{Val: dql.NewString(" - ")},
		},
	}, ctx)
	if s, ok := v.AsString(); !ok || s != "a - b - c" {
		t.Errorf("expected 'a - b - c', got %v", v)
	}
}

func TestFnSplit(t *testing.T) {
	ev := setupEval()
	ctx := testCtx()

	v := ev.Eval(dql.FunctionCallExpr{
		Name: "split",
		Args: []dql.Expr{
			dql.LiteralExpr{Val: dql.NewString("a,b,c")},
			dql.LiteralExpr{Val: dql.NewString(",")},
		},
	}, ctx)
	items, ok := v.AsList()
	if !ok || len(items) != 3 {
		t.Errorf("expected 3 items, got %v", v)
	}
}

func TestFnReplace(t *testing.T) {
	ev := setupEval()
	ctx := testCtx()

	v := ev.Eval(dql.FunctionCallExpr{
		Name: "replace",
		Args: []dql.Expr{
			dql.LiteralExpr{Val: dql.NewString("hello world")},
			dql.LiteralExpr{Val: dql.NewString("world")},
			dql.LiteralExpr{Val: dql.NewString("there")},
		},
	}, ctx)
	if s, ok := v.AsString(); !ok || s != "hello there" {
		t.Errorf("expected 'hello there', got %v", v)
	}
}

func TestFnStartsEnds(t *testing.T) {
	ev := setupEval()
	ctx := testCtx()

	v := ev.Eval(dql.FunctionCallExpr{
		Name: "startswith",
		Args: []dql.Expr{
			dql.LiteralExpr{Val: dql.NewString("hello world")},
			dql.LiteralExpr{Val: dql.NewString("hello")},
		},
	}, ctx)
	if b, ok := v.AsBool(); !ok || !b {
		t.Errorf("startswith: %v", v)
	}

	v = ev.Eval(dql.FunctionCallExpr{
		Name: "endswith",
		Args: []dql.Expr{
			dql.LiteralExpr{Val: dql.NewString("hello world")},
			dql.LiteralExpr{Val: dql.NewString("world")},
		},
	}, ctx)
	if b, ok := v.AsBool(); !ok || !b {
		t.Errorf("endswith: %v", v)
	}
}

func TestFnSort(t *testing.T) {
	ev := setupEval()
	ctx := testCtx()

	v := ev.Eval(dql.FunctionCallExpr{
		Name: "sort",
		Args: []dql.Expr{
			dql.LiteralExpr{Val: dql.NewList([]dql.Value{dql.NewNumber(3), dql.NewNumber(1), dql.NewNumber(2)})},
		},
	}, ctx)
	items, ok := v.AsList()
	if !ok || len(items) != 3 {
		t.Fatalf("expected 3 items, got %v", v)
	}
	if n, _ := items[0].AsNumber(); n != 1 {
		t.Errorf("expected first=1, got %v", items[0])
	}
}

func TestFnReverse(t *testing.T) {
	ev := setupEval()
	ctx := testCtx()

	v := ev.Eval(dql.FunctionCallExpr{
		Name: "reverse",
		Args: []dql.Expr{
			dql.LiteralExpr{Val: dql.NewList([]dql.Value{dql.NewNumber(1), dql.NewNumber(2), dql.NewNumber(3)})},
		},
	}, ctx)
	items, ok := v.AsList()
	if !ok || len(items) != 3 {
		t.Fatalf("expected 3 items, got %v", v)
	}
	if n, _ := items[0].AsNumber(); n != 3 {
		t.Errorf("expected first=3, got %v", items[0])
	}
}

func TestFnUnique(t *testing.T) {
	ev := setupEval()
	ctx := testCtx()

	v := ev.Eval(dql.FunctionCallExpr{
		Name: "unique",
		Args: []dql.Expr{
			dql.LiteralExpr{Val: dql.NewList([]dql.Value{dql.NewNumber(1), dql.NewNumber(2), dql.NewNumber(1)})},
		},
	}, ctx)
	items, ok := v.AsList()
	if !ok || len(items) != 2 {
		t.Errorf("expected 2 unique items, got %v", v)
	}
}

func TestFnTypeof(t *testing.T) {
	ev := setupEval()
	ctx := testCtx()

	tests := []struct {
		val  dql.Value
		want string
	}{
		{dql.NewNull(), "null"},
		{dql.NewNumber(1), "number"},
		{dql.NewString("x"), "string"},
		{dql.NewBool(true), "boolean"},
	}
	for _, tt := range tests {
		v := ev.Eval(dql.FunctionCallExpr{
			Name: "typeof",
			Args: []dql.Expr{dql.LiteralExpr{Val: tt.val}},
		}, ctx)
		if s, ok := v.AsString(); !ok || s != tt.want {
			t.Errorf("typeof(%v) = %v, want %q", tt.val, v, tt.want)
		}
	}
}

func TestFnRegextest(t *testing.T) {
	ev := setupEval()
	ctx := testCtx()

	v := ev.Eval(dql.FunctionCallExpr{
		Name: "regextest",
		Args: []dql.Expr{
			dql.LiteralExpr{Val: dql.NewString(`\d+`)},
			dql.LiteralExpr{Val: dql.NewString("abc123")},
		},
	}, ctx)
	if b, ok := v.AsBool(); !ok || !b {
		t.Errorf("expected true, got %v", v)
	}
}

func TestFnNonnull(t *testing.T) {
	ev := setupEval()
	ctx := testCtx()

	v := ev.Eval(dql.FunctionCallExpr{
		Name: "nonnull",
		Args: []dql.Expr{
			dql.LiteralExpr{Val: dql.NewList([]dql.Value{dql.NewNull(), dql.NewNumber(1), dql.NewNull(), dql.NewNumber(2)})},
		},
	}, ctx)
	items, ok := v.AsList()
	if !ok || len(items) != 2 {
		t.Errorf("expected 2 nonnull items, got %v", v)
	}
}

func TestFnAllAnyNone(t *testing.T) {
	ev := setupEval()
	ctx := testCtx()

	allTrue := dql.NewList([]dql.Value{dql.NewBool(true), dql.NewBool(true)})
	mixed := dql.NewList([]dql.Value{dql.NewBool(true), dql.NewBool(false)})
	allFalse := dql.NewList([]dql.Value{dql.NewBool(false), dql.NewBool(false)})

	// all
	if v := ev.Eval(dql.FunctionCallExpr{Name: "all", Args: []dql.Expr{dql.LiteralExpr{Val: allTrue}}}, ctx); !v.Truthy() {
		t.Error("all(true,true) should be true")
	}
	if v := ev.Eval(dql.FunctionCallExpr{Name: "all", Args: []dql.Expr{dql.LiteralExpr{Val: mixed}}}, ctx); v.Truthy() {
		t.Error("all(true,false) should be false")
	}

	// any
	if v := ev.Eval(dql.FunctionCallExpr{Name: "any", Args: []dql.Expr{dql.LiteralExpr{Val: mixed}}}, ctx); !v.Truthy() {
		t.Error("any(true,false) should be true")
	}
	if v := ev.Eval(dql.FunctionCallExpr{Name: "any", Args: []dql.Expr{dql.LiteralExpr{Val: allFalse}}}, ctx); v.Truthy() {
		t.Error("any(false,false) should be false")
	}

	// none
	if v := ev.Eval(dql.FunctionCallExpr{Name: "none", Args: []dql.Expr{dql.LiteralExpr{Val: allFalse}}}, ctx); !v.Truthy() {
		t.Error("none(false,false) should be true")
	}
}

func TestFnTruncate(t *testing.T) {
	ev := setupEval()
	ctx := testCtx()

	v := ev.Eval(dql.FunctionCallExpr{
		Name: "truncate",
		Args: []dql.Expr{
			dql.LiteralExpr{Val: dql.NewString("Hello, World!")},
			dql.LiteralExpr{Val: dql.NewNumber(10)},
		},
	}, ctx)
	if s, ok := v.AsString(); !ok || s != "Hello, ..." {
		t.Errorf("expected 'Hello, ...', got %q", s)
	}
}

func TestFnPadLeftRight(t *testing.T) {
	ev := setupEval()
	ctx := testCtx()

	v := ev.Eval(dql.FunctionCallExpr{
		Name: "padleft",
		Args: []dql.Expr{
			dql.LiteralExpr{Val: dql.NewString("5")},
			dql.LiteralExpr{Val: dql.NewNumber(3)},
			dql.LiteralExpr{Val: dql.NewString("0")},
		},
	}, ctx)
	if s, ok := v.AsString(); !ok || s != "005" {
		t.Errorf("padleft: expected '005', got %q", s)
	}
}

func TestFnSlice(t *testing.T) {
	ev := setupEval()
	ctx := testCtx()

	v := ev.Eval(dql.FunctionCallExpr{
		Name: "slice",
		Args: []dql.Expr{
			dql.LiteralExpr{Val: dql.NewList([]dql.Value{dql.NewNumber(1), dql.NewNumber(2), dql.NewNumber(3), dql.NewNumber(4)})},
			dql.LiteralExpr{Val: dql.NewNumber(1)},
			dql.LiteralExpr{Val: dql.NewNumber(3)},
		},
	}, ctx)
	items, ok := v.AsList()
	if !ok || len(items) != 2 {
		t.Errorf("expected 2 items, got %v", v)
	}
}
