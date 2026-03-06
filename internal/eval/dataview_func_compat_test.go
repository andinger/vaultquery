package eval

// Tests translated from the Dataview repo:
// https://github.com/blacksmithgu/obsidian-dataview/blob/master/src/test/function/functions.test.ts

import (
	"testing"

	"github.com/andinger/vaultquery/internal/dql"
)

// helper: evaluate a function call with literal args against an empty context
func evalFn(ev *Evaluator, name string, args ...dql.Value) dql.Value {
	exprs := make([]dql.Expr, len(args))
	for i, a := range args {
		exprs[i] = dql.LiteralExpr{Val: a}
	}
	return ev.Eval(dql.FunctionCallExpr{Name: name, Args: exprs}, testCtx())
}

// --- length() ---

func TestDV_FnLengthArray(t *testing.T) {
	ev := setupEval()
	v := evalFn(ev, "length", dql.NewList([]dql.Value{dql.NewNumber(1), dql.NewNumber(2)}))
	if n, ok := v.AsNumber(); !ok || n != 2 {
		t.Errorf("expected 2, got %v", v)
	}

	v = evalFn(ev, "length", dql.NewList([]dql.Value{}))
	if n, ok := v.AsNumber(); !ok || n != 0 {
		t.Errorf("expected 0, got %v", v)
	}
}

func TestDV_FnLengthString(t *testing.T) {
	ev := setupEval()
	v := evalFn(ev, "length", dql.NewString("hello"))
	if n, ok := v.AsNumber(); !ok || n != 5 {
		t.Errorf("expected 5, got %v", v)
	}

	v = evalFn(ev, "length", dql.NewString(""))
	if n, ok := v.AsNumber(); !ok || n != 0 {
		t.Errorf("expected 0, got %v", v)
	}
}

// --- contains() ---

func TestDV_FnContainsArray(t *testing.T) {
	ev := setupEval()
	list := dql.NewList([]dql.Value{dql.NewString("hello"), dql.NewNumber(1)})

	v := evalFn(ev, "contains", list, dql.NewString("hello"))
	if b, ok := v.AsBool(); !ok || !b {
		t.Errorf("expected true, got %v", v)
	}

	v = evalFn(ev, "contains", list, dql.NewNumber(6))
	if b, ok := v.AsBool(); !ok || b {
		t.Errorf("expected false, got %v", v)
	}
}

func TestDV_FnContainsString(t *testing.T) {
	ev := setupEval()

	v := evalFn(ev, "contains", dql.NewString("hello"), dql.NewString("hello"))
	if b, ok := v.AsBool(); !ok || !b {
		t.Error("expected contains('hello', 'hello') = true")
	}

	v = evalFn(ev, "contains", dql.NewString("meep"), dql.NewString("me"))
	if b, ok := v.AsBool(); !ok || !b {
		t.Error("expected contains('meep', 'me') = true")
	}

	v = evalFn(ev, "contains", dql.NewString("hello"), dql.NewString("xd"))
	if b, ok := v.AsBool(); !ok || b {
		t.Error("expected contains('hello', 'xd') = false")
	}
}

func TestDV_FnContainsFuzzyArray(t *testing.T) {
	ev := setupEval()
	list := dql.NewList([]dql.Value{dql.NewString("hello")})

	// Fuzzy: "he" is substring of "hello"
	v := evalFn(ev, "contains", list, dql.NewString("he"))
	if b, ok := v.AsBool(); !ok || !b {
		t.Error("expected fuzzy contains(['hello'], 'he') = true")
	}

	v = evalFn(ev, "contains", list, dql.NewString("no"))
	if b, ok := v.AsBool(); !ok || b {
		t.Error("expected fuzzy contains(['hello'], 'no') = false")
	}
}

func TestDV_FnEcontainsArray(t *testing.T) {
	ev := setupEval()
	list := dql.NewList([]dql.Value{dql.NewString("hello")})

	// Exact: "he" is NOT an exact element
	v := evalFn(ev, "econtains", list, dql.NewString("he"))
	if b, ok := v.AsBool(); !ok || b {
		t.Error("expected econtains(['hello'], 'he') = false")
	}

	v = evalFn(ev, "econtains", list, dql.NewString("hello"))
	if b, ok := v.AsBool(); !ok || !b {
		t.Error("expected econtains(['hello'], 'hello') = true")
	}

	list2 := dql.NewList([]dql.Value{dql.NewString("hello"), dql.NewNumber(19)})
	v = evalFn(ev, "econtains", list2, dql.NewNumber(1))
	if b, ok := v.AsBool(); !ok || b {
		t.Error("expected econtains(['hello', 19], 1) = false")
	}

	v = evalFn(ev, "econtains", list2, dql.NewNumber(19))
	if b, ok := v.AsBool(); !ok || !b {
		t.Error("expected econtains(['hello', 19], 19) = true")
	}
}

// --- reverse() ---

func TestDV_FnReverse(t *testing.T) {
	ev := setupEval()
	list := dql.NewList([]dql.Value{dql.NewNumber(1), dql.NewNumber(2), dql.NewNumber(3)})
	v := evalFn(ev, "reverse", list)
	items, ok := v.AsList()
	if !ok || len(items) != 3 {
		t.Fatalf("expected list of 3, got %v", v)
	}
	expected := []float64{3, 2, 1}
	for i, e := range expected {
		if n, _ := items[i].AsNumber(); n != e {
			t.Errorf("index %d: expected %v, got %v", i, e, n)
		}
	}
}

// --- sort() ---

func TestDV_FnSort(t *testing.T) {
	ev := setupEval()
	list := dql.NewList([]dql.Value{dql.NewNumber(2), dql.NewNumber(3), dql.NewNumber(1)})
	v := evalFn(ev, "sort", list)
	items, ok := v.AsList()
	if !ok || len(items) != 3 {
		t.Fatalf("expected list of 3, got %v", v)
	}
	expected := []float64{1, 2, 3}
	for i, e := range expected {
		if n, _ := items[i].AsNumber(); n != e {
			t.Errorf("index %d: expected %v, got %v", i, e, n)
		}
	}
}

// --- default() ---

func TestDV_FnDefault(t *testing.T) {
	ev := setupEval()

	// default(null, 1) → 1
	v := evalFn(ev, "default", dql.NewNull(), dql.NewNumber(1))
	if n, ok := v.AsNumber(); !ok || n != 1 {
		t.Errorf("expected 1, got %v", v)
	}

	// default(2, 1) → 2
	v = evalFn(ev, "default", dql.NewNumber(2), dql.NewNumber(1))
	if n, ok := v.AsNumber(); !ok || n != 2 {
		t.Errorf("expected 2, got %v", v)
	}
}

// --- choice() ---

func TestDV_FnChoice(t *testing.T) {
	ev := setupEval()

	// choice(true, 1, 2) → 1
	v := evalFn(ev, "choice", dql.NewBool(true), dql.NewNumber(1), dql.NewNumber(2))
	if n, ok := v.AsNumber(); !ok || n != 1 {
		t.Errorf("expected 1, got %v", v)
	}

	// choice(false, 1, 2) → 2
	v = evalFn(ev, "choice", dql.NewBool(false), dql.NewNumber(1), dql.NewNumber(2))
	if n, ok := v.AsNumber(); !ok || n != 2 {
		t.Errorf("expected 2, got %v", v)
	}
}

// --- nonnull() ---

func TestDV_FnNonnull(t *testing.T) {
	ev := setupEval()

	v := evalFn(ev, "nonnull", dql.NewNull(), dql.NewNull(), dql.NewNumber(1))
	items, ok := v.AsList()
	if !ok || len(items) != 1 {
		t.Fatalf("expected list of 1, got %v", v)
	}
	if n, _ := items[0].AsNumber(); n != 1 {
		t.Errorf("expected 1, got %v", items[0])
	}

	v = evalFn(ev, "nonnull", dql.NewString("yes"))
	items, ok = v.AsList()
	if !ok || len(items) != 1 {
		t.Fatalf("expected list of 1, got %v", v)
	}
	if s, _ := items[0].AsString(); s != "yes" {
		t.Errorf("expected 'yes', got %v", items[0])
	}
}

// --- Numeric functions ---

func TestDV_FnRound(t *testing.T) {
	ev := setupEval()
	v := evalFn(ev, "round", dql.NewNumber(3.7))
	if n, _ := v.AsNumber(); n != 4 {
		t.Errorf("expected 4, got %v", n)
	}
}

func TestDV_FnFloor(t *testing.T) {
	ev := setupEval()
	v := evalFn(ev, "floor", dql.NewNumber(3.7))
	if n, _ := v.AsNumber(); n != 3 {
		t.Errorf("expected 3, got %v", n)
	}
}

func TestDV_FnCeil(t *testing.T) {
	ev := setupEval()
	v := evalFn(ev, "ceil", dql.NewNumber(3.2))
	if n, _ := v.AsNumber(); n != 4 {
		t.Errorf("expected 4, got %v", n)
	}
}

func TestDV_FnMinMax(t *testing.T) {
	ev := setupEval()

	v := evalFn(ev, "min", dql.NewList([]dql.Value{dql.NewNumber(5), dql.NewNumber(3), dql.NewNumber(8)}))
	if n, _ := v.AsNumber(); n != 3 {
		t.Errorf("min: expected 3, got %v", n)
	}

	v = evalFn(ev, "max", dql.NewList([]dql.Value{dql.NewNumber(5), dql.NewNumber(3), dql.NewNumber(8)}))
	if n, _ := v.AsNumber(); n != 8 {
		t.Errorf("max: expected 8, got %v", n)
	}
}

func TestDV_FnSum(t *testing.T) {
	ev := setupEval()
	v := evalFn(ev, "sum", dql.NewList([]dql.Value{dql.NewNumber(1), dql.NewNumber(2), dql.NewNumber(3)}))
	if n, _ := v.AsNumber(); n != 6 {
		t.Errorf("expected 6, got %v", n)
	}
}

func TestDV_FnAverage(t *testing.T) {
	ev := setupEval()
	v := evalFn(ev, "average", dql.NewList([]dql.Value{dql.NewNumber(2), dql.NewNumber(4), dql.NewNumber(6)}))
	if n, _ := v.AsNumber(); n != 4 {
		t.Errorf("expected 4, got %v", n)
	}
}

func TestDV_FnProduct(t *testing.T) {
	ev := setupEval()
	v := evalFn(ev, "product", dql.NewList([]dql.Value{dql.NewNumber(2), dql.NewNumber(3), dql.NewNumber(4)}))
	if n, _ := v.AsNumber(); n != 24 {
		t.Errorf("expected 24, got %v", n)
	}
}

// --- String functions ---

func TestDV_FnLowerUpper(t *testing.T) {
	ev := setupEval()

	v := evalFn(ev, "lower", dql.NewString("Hello World"))
	if s, _ := v.AsString(); s != "hello world" {
		t.Errorf("lower: expected 'hello world', got %q", s)
	}

	v = evalFn(ev, "upper", dql.NewString("hello"))
	if s, _ := v.AsString(); s != "HELLO" {
		t.Errorf("upper: expected 'HELLO', got %q", s)
	}
}

func TestDV_FnReplace(t *testing.T) {
	ev := setupEval()
	v := evalFn(ev, "replace", dql.NewString("hello world"), dql.NewString("world"), dql.NewString("there"))
	if s, _ := v.AsString(); s != "hello there" {
		t.Errorf("expected 'hello there', got %q", s)
	}
}

func TestDV_FnSplit(t *testing.T) {
	ev := setupEval()
	v := evalFn(ev, "split", dql.NewString("a,b,c"), dql.NewString(","))
	items, ok := v.AsList()
	if !ok || len(items) != 3 {
		t.Fatalf("expected list of 3, got %v", v)
	}
	expected := []string{"a", "b", "c"}
	for i, e := range expected {
		if s, _ := items[i].AsString(); s != e {
			t.Errorf("index %d: expected %q, got %q", i, e, s)
		}
	}
}

func TestDV_FnStartsWithEndsWith(t *testing.T) {
	ev := setupEval()

	v := evalFn(ev, "startswith", dql.NewString("hello world"), dql.NewString("hello"))
	if b, _ := v.AsBool(); !b {
		t.Error("expected startswith('hello world', 'hello') = true")
	}

	v = evalFn(ev, "endswith", dql.NewString("hello world"), dql.NewString("world"))
	if b, _ := v.AsBool(); !b {
		t.Error("expected endswith('hello world', 'world') = true")
	}

	v = evalFn(ev, "startswith", dql.NewString("hello"), dql.NewString("xyz"))
	if b, _ := v.AsBool(); b {
		t.Error("expected startswith('hello', 'xyz') = false")
	}
}

func TestDV_FnRegextest(t *testing.T) {
	ev := setupEval()

	v := evalFn(ev, "regextest", dql.NewString(`\d+`), dql.NewString("hello123"))
	if b, _ := v.AsBool(); !b {
		t.Error(`expected regextest('\d+', 'hello123') = true`)
	}

	v = evalFn(ev, "regextest", dql.NewString(`\d+`), dql.NewString("hello"))
	if b, _ := v.AsBool(); b {
		t.Error(`expected regextest('\d+', 'hello') = false`)
	}
}

func TestDV_FnJoin(t *testing.T) {
	ev := setupEval()
	list := dql.NewList([]dql.Value{dql.NewString("a"), dql.NewString("b"), dql.NewString("c")})
	v := evalFn(ev, "join", list, dql.NewString(", "))
	if s, _ := v.AsString(); s != "a, b, c" {
		t.Errorf("expected 'a, b, c', got %q", s)
	}
}

func TestDV_FnUnique(t *testing.T) {
	ev := setupEval()
	list := dql.NewList([]dql.Value{
		dql.NewNumber(1), dql.NewNumber(2), dql.NewNumber(1), dql.NewNumber(3), dql.NewNumber(2),
	})
	v := evalFn(ev, "unique", list)
	items, ok := v.AsList()
	if !ok || len(items) != 3 {
		t.Fatalf("expected 3 unique items, got %v", v)
	}
}

func TestDV_FnFlat(t *testing.T) {
	ev := setupEval()
	// flat([1, 2, 3, [11, 12]]) → [1, 2, 3, 11, 12]
	inner := dql.NewList([]dql.Value{dql.NewNumber(11), dql.NewNumber(12)})
	list := dql.NewList([]dql.Value{dql.NewNumber(1), dql.NewNumber(2), dql.NewNumber(3), inner})
	v := evalFn(ev, "flat", list)
	items, ok := v.AsList()
	if !ok || len(items) != 5 {
		t.Fatalf("expected 5 items after flat, got %d: %v", len(items), v)
	}
}

// --- Type constructors ---

func TestDV_FnObject(t *testing.T) {
	ev := setupEval()
	v := evalFn(ev, "object",
		dql.NewString("hello"), dql.NewNumber(1),
		dql.NewString("world"), dql.NewNumber(2))
	obj, ok := v.AsObject()
	if !ok {
		t.Fatalf("expected object, got %v", v)
	}
	if val, exists := obj["hello"]; !exists {
		t.Error("missing key 'hello'")
	} else if n, _ := val.AsNumber(); n != 1 {
		t.Errorf("expected 1, got %v", n)
	}
	if val, exists := obj["world"]; !exists {
		t.Error("missing key 'world'")
	} else if n, _ := val.AsNumber(); n != 2 {
		t.Errorf("expected 2, got %v", n)
	}
}

func TestDV_FnList(t *testing.T) {
	ev := setupEval()
	v := evalFn(ev, "list", dql.NewNumber(1), dql.NewNumber(2), dql.NewNumber(3))
	items, ok := v.AsList()
	if !ok || len(items) != 3 {
		t.Fatalf("expected list of 3, got %v", v)
	}
	for i, expected := range []float64{1, 2, 3} {
		if n, _ := items[i].AsNumber(); n != expected {
			t.Errorf("index %d: expected %v, got %v", i, expected, n)
		}
	}
}

func TestDV_FnTypeof(t *testing.T) {
	ev := setupEval()

	tests := []struct {
		val    dql.Value
		expect string
	}{
		{dql.NewNull(), "null"},
		{dql.NewNumber(42), "number"},
		{dql.NewString("hi"), "string"},
		{dql.NewBool(true), "boolean"},
		{dql.NewList(nil), "array"},
		{dql.NewObject(nil), "object"},
	}
	for _, tt := range tests {
		v := evalFn(ev, "typeof", tt.val)
		if s, _ := v.AsString(); s != tt.expect {
			t.Errorf("typeof(%v): expected %q, got %q", tt.val, tt.expect, s)
		}
	}
}

// --- All / Any / None ---

func TestDV_FnAllAnyNone(t *testing.T) {
	ev := setupEval()

	allTrue := dql.NewList([]dql.Value{dql.NewBool(true), dql.NewBool(true)})
	mixed := dql.NewList([]dql.Value{dql.NewBool(true), dql.NewBool(false)})
	allFalse := dql.NewList([]dql.Value{dql.NewBool(false), dql.NewBool(false)})

	v := evalFn(ev, "all", allTrue)
	if b, _ := v.AsBool(); !b {
		t.Error("expected all([true, true]) = true")
	}

	v = evalFn(ev, "all", mixed)
	if b, _ := v.AsBool(); b {
		t.Error("expected all([true, false]) = false")
	}

	v = evalFn(ev, "any", mixed)
	if b, _ := v.AsBool(); !b {
		t.Error("expected any([true, false]) = true")
	}

	v = evalFn(ev, "any", allFalse)
	if b, _ := v.AsBool(); b {
		t.Error("expected any([false, false]) = false")
	}

	v = evalFn(ev, "none", allFalse)
	if b, _ := v.AsBool(); !b {
		t.Error("expected none([false, false]) = true")
	}

	v = evalFn(ev, "none", mixed)
	if b, _ := v.AsBool(); b {
		t.Error("expected none([true, false]) = false")
	}
}

// --- Substring ---

func TestDV_FnSubstring(t *testing.T) {
	ev := setupEval()
	v := evalFn(ev, "substring", dql.NewString("hello world"), dql.NewNumber(0), dql.NewNumber(5))
	if s, _ := v.AsString(); s != "hello" {
		t.Errorf("expected 'hello', got %q", s)
	}
}

func TestDV_FnTruncate(t *testing.T) {
	ev := setupEval()
	v := evalFn(ev, "truncate", dql.NewString("hello world"), dql.NewNumber(5))
	s, _ := v.AsString()
	if len(s) > 8 { // 5 chars + "..."
		t.Errorf("expected truncated string, got %q", s)
	}
}

func TestDV_FnPadleftPadright(t *testing.T) {
	ev := setupEval()

	v := evalFn(ev, "padleft", dql.NewString("hi"), dql.NewNumber(5))
	if s, _ := v.AsString(); s != "   hi" {
		t.Errorf("padleft: expected '   hi', got %q", s)
	}

	v = evalFn(ev, "padright", dql.NewString("hi"), dql.NewNumber(5))
	if s, _ := v.AsString(); s != "hi   " {
		t.Errorf("padright: expected 'hi   ', got %q", s)
	}
}
