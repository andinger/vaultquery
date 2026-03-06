package eval

import (
	"fmt"
	"strings"

	"github.com/andinger/vaultquery/internal/dql"
)

// FuncImpl is the signature for built-in function implementations.
type FuncImpl func(args []dql.Value, ctx *EvalContext) dql.Value

// Evaluator evaluates DQL expressions against an EvalContext.
type Evaluator struct {
	Functions map[string]FuncImpl
}

// New creates an Evaluator with an empty function registry.
func New() *Evaluator {
	return &Evaluator{
		Functions: make(map[string]FuncImpl),
	}
}

// RegisterFunc adds a built-in function.
func (ev *Evaluator) RegisterFunc(name string, fn FuncImpl) {
	ev.Functions[strings.ToLower(name)] = fn
}

// Eval evaluates an expression and returns a Value.
func (ev *Evaluator) Eval(expr dql.Expr, ctx *EvalContext) dql.Value {
	switch e := expr.(type) {
	case dql.LiteralExpr:
		return e.Val

	case dql.FieldAccessExpr:
		return ev.evalFieldAccess(e, ctx)

	case dql.ComparisonExpr:
		return ev.evalComparison(e, ctx)

	case dql.LogicalExpr:
		return ev.evalLogical(e, ctx)

	case dql.ParenExpr:
		return ev.Eval(e.Inner, ctx)

	case dql.ExistsExpr:
		return ev.evalExists(e, ctx)

	case dql.ArithmeticExpr:
		return ev.evalArithmetic(e, ctx)

	case dql.NegationExpr:
		return ev.Eval(e.Inner, ctx).Negate()

	case dql.FunctionCallExpr:
		return ev.evalFunctionCall(e, ctx)

	case dql.IndexExpr:
		return ev.evalIndex(e, ctx)

	case dql.LambdaExpr:
		// Lambdas are not directly evaluable; they're handled by function calls.
		return dql.NewNull()

	default:
		return dql.NewNull()
	}
}

// EvalBool evaluates an expression and returns its boolean interpretation.
func (ev *Evaluator) EvalBool(expr dql.Expr, ctx *EvalContext) bool {
	return ev.Eval(expr, ctx).Truthy()
}

func (ev *Evaluator) evalFieldAccess(e dql.FieldAccessExpr, ctx *EvalContext) dql.Value {
	if len(e.Parts) == 1 {
		return ctx.Lookup(e.Parts[0])
	}

	// Multi-part: try full dotted name first, then walk the chain
	fullName := dql.FieldName(e.Parts)
	if v := ctx.Lookup(fullName); !v.IsNull() {
		return v
	}

	// Walk chain: resolve first part, then traverse object fields
	current := ctx.Lookup(e.Parts[0])
	for _, part := range e.Parts[1:] {
		if obj, ok := current.AsObject(); ok {
			if v, exists := obj[part]; exists {
				current = v
			} else {
				return dql.NewNull()
			}
		} else {
			return dql.NewNull()
		}
	}
	return current
}

func (ev *Evaluator) evalComparison(e dql.ComparisonExpr, ctx *EvalContext) dql.Value {
	left := ctx.Lookup(e.Field)
	right := dql.CoerceFromString(e.Value)

	// For "contains" / "!contains", check if left (as list) contains right
	if e.Op == "contains" || e.Op == "!contains" {
		result := valueContains(left, right)
		if e.Op == "!contains" {
			result = !result
		}
		return dql.NewBool(result)
	}

	cmp := left.Compare(right)
	var result bool
	switch e.Op {
	case "=":
		result = cmp == 0
	case "!=":
		result = cmp != 0
	case "<":
		result = cmp < 0
	case ">":
		result = cmp > 0
	case "<=":
		result = cmp <= 0
	case ">=":
		result = cmp >= 0
	}
	return dql.NewBool(result)
}

func valueContains(haystack, needle dql.Value) bool {
	// If haystack is a list, check membership
	if items, ok := haystack.AsList(); ok {
		for _, item := range items {
			if item.Compare(needle) == 0 {
				return true
			}
		}
		return false
	}
	// If haystack is a string, check substring
	if hs, ok := haystack.AsString(); ok {
		if ns, ok := needle.AsString(); ok {
			return strings.Contains(strings.ToLower(hs), strings.ToLower(ns))
		}
	}
	// Single value equality
	return haystack.Compare(needle) == 0
}

func (ev *Evaluator) evalLogical(e dql.LogicalExpr, ctx *EvalContext) dql.Value {
	left := ev.EvalBool(e.Left, ctx)
	switch e.Op {
	case "AND":
		if !left {
			return dql.NewBool(false)
		}
		return dql.NewBool(ev.EvalBool(e.Right, ctx))
	case "OR":
		if left {
			return dql.NewBool(true)
		}
		return dql.NewBool(ev.EvalBool(e.Right, ctx))
	}
	return dql.NewNull()
}

func (ev *Evaluator) evalExists(e dql.ExistsExpr, ctx *EvalContext) dql.Value {
	v := ctx.Lookup(e.Field)
	exists := !v.IsNull()
	if e.Negated {
		exists = !exists
	}
	return dql.NewBool(exists)
}

func (ev *Evaluator) evalArithmetic(e dql.ArithmeticExpr, ctx *EvalContext) dql.Value {
	left := ev.Eval(e.Left, ctx)
	right := ev.Eval(e.Right, ctx)
	switch e.Op {
	case "+":
		return left.Add(right)
	case "-":
		return left.Sub(right)
	case "*":
		return left.Mul(right)
	case "/":
		return left.Div(right)
	case "%":
		return left.Mod(right)
	}
	return dql.NewNull()
}

func (ev *Evaluator) evalFunctionCall(e dql.FunctionCallExpr, ctx *EvalContext) dql.Value {
	name := strings.ToLower(e.Name)
	fn, ok := ev.Functions[name]
	if !ok {
		return dql.NewNull()
	}

	// Evaluate arguments, but pass lambdas unevaluated
	args := make([]dql.Value, len(e.Args))
	for i, arg := range e.Args {
		if _, isLambda := arg.(dql.LambdaExpr); isLambda {
			// Store lambda as a special value — functions handle it themselves
			args[i] = dql.NewNull() // placeholder; functions that need lambdas get the raw expr
		} else {
			args[i] = ev.Eval(arg, ctx)
		}
	}

	// For functions that accept lambdas (filter, map, etc.), we need a different approach.
	// Check if any arg is a lambda and pass a wrapper.
	return fn(args, ctx)
}

// EvalLambda evaluates a lambda expression with the given arguments.
func (ev *Evaluator) EvalLambda(lambda dql.LambdaExpr, args []dql.Value, ctx *EvalContext) dql.Value {
	scope := make(map[string]dql.Value, len(lambda.Params))
	for i, param := range lambda.Params {
		if i < len(args) {
			scope[param] = args[i]
		} else {
			scope[param] = dql.NewNull()
		}
	}
	ctx.PushScope(scope)
	defer ctx.PopScope()
	return ev.Eval(lambda.Body, ctx)
}

// EvalFuncWithLambda evaluates a function call where one arg is a lambda.
// This is used by filter(), map(), etc.
func (ev *Evaluator) EvalFuncWithLambda(callExpr dql.FunctionCallExpr, ctx *EvalContext) dql.Value {
	name := strings.ToLower(callExpr.Name)
	fn, ok := ev.Functions[name]
	if !ok {
		return dql.NewNull()
	}

	// For lambda-aware functions, we build a special invocation
	args := make([]dql.Value, len(callExpr.Args))
	for i, arg := range callExpr.Args {
		if _, isLambda := arg.(dql.LambdaExpr); isLambda {
			args[i] = dql.NewNull()
		} else {
			args[i] = ev.Eval(arg, ctx)
		}
	}

	return fn(args, ctx)
}

func (ev *Evaluator) evalIndex(e dql.IndexExpr, ctx *EvalContext) dql.Value {
	obj := ev.Eval(e.Object, ctx)
	idx := ev.Eval(e.Index, ctx)

	// List indexing
	if items, ok := obj.AsList(); ok {
		if n, ok := idx.AsNumber(); ok {
			i := int(n)
			if i < 0 {
				i = len(items) + i
			}
			if i >= 0 && i < len(items) {
				return items[i]
			}
		}
		return dql.NewNull()
	}

	// Object field access
	if fields, ok := obj.AsObject(); ok {
		if key, ok := idx.AsString(); ok {
			if v, exists := fields[key]; exists {
				return v
			}
		}
		return dql.NewNull()
	}

	return dql.NewNull()
}

// BuildEvalContextFromEAV creates an EvalContext from raw EAV string data.
func BuildEvalContextFromEAV(path, title string, fields map[string][]string) *EvalContext {
	vals := make(map[string]dql.Value, len(fields))
	for key, values := range fields {
		switch len(values) {
		case 0:
			// skip
		case 1:
			vals[key] = dql.CoerceFromString(values[0])
		default:
			items := make([]dql.Value, len(values))
			for i, v := range values {
				items[i] = dql.CoerceFromString(v)
			}
			vals[key] = dql.NewList(items)
		}
	}
	return NewEvalContext(path, title, vals)
}

// String implements fmt.Stringer for better error messages.
func (ev *Evaluator) String() string {
	return fmt.Sprintf("Evaluator{functions: %d}", len(ev.Functions))
}
