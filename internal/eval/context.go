package eval

import "github.com/andinger/vaultquery/internal/dql"

// EvalContext provides field lookups for evaluating expressions against a file.
type EvalContext struct {
	// Fields maps frontmatter field names to their values.
	Fields map[string]dql.Value

	// FilePath is the full path of the file being evaluated.
	FilePath string

	// FileTitle is the title derived from the file.
	FileTitle string

	// Variables holds the variable scope stack for lambda evaluation.
	Variables []map[string]dql.Value
}

// NewEvalContext creates a context for a file with the given EAV fields.
func NewEvalContext(path, title string, fields map[string]dql.Value) *EvalContext {
	if fields == nil {
		fields = make(map[string]dql.Value)
	}
	return &EvalContext{
		Fields:    fields,
		FilePath:  path,
		FileTitle: title,
	}
}

// Lookup resolves a field name, checking variables first, then fields.
func (ctx *EvalContext) Lookup(name string) dql.Value {
	// Check variable scopes (innermost first)
	for i := len(ctx.Variables) - 1; i >= 0; i-- {
		if v, ok := ctx.Variables[i][name]; ok {
			return v
		}
	}

	// Check frontmatter fields
	if v, ok := ctx.Fields[name]; ok {
		return v
	}

	return dql.NewNull()
}

// PushScope adds a new variable scope.
func (ctx *EvalContext) PushScope(vars map[string]dql.Value) {
	ctx.Variables = append(ctx.Variables, vars)
}

// PopScope removes the innermost variable scope.
func (ctx *EvalContext) PopScope() {
	if len(ctx.Variables) > 0 {
		ctx.Variables = ctx.Variables[:len(ctx.Variables)-1]
	}
}
