package dql

// Query represents a parsed DQL query.
type Query struct {
	Mode      string      // "TABLE", "LIST", "TASK", "CALENDAR"
	Fields    []FieldDef  // TABLE fields (empty for LIST); expression-based
	WithoutID bool        // WITHOUT ID clause
	From      string      // FROM path (empty = all) — simple string for backward compat
	FromSource FromSource // Rich FROM tree (nil = use From string)
	Where     Expr        // nil if no WHERE
	Sort      []SortField // nil if no SORT
	Limit     int         // 0 = no limit
	GroupBy   []FieldDef
	Flatten   []FieldDef
}

// FieldDef represents a field expression with an optional alias.
type FieldDef struct {
	Expr  Expr
	Alias string
}

// SortField represents a sort expression with direction.
type SortField struct {
	Field string // simple field name (backward compat)
	Expr  Expr   // expression to sort by (preferred over Field)
	Desc  bool
}

// --- FROM source types ---

// FromSource is the interface for FROM clause sources.
type FromSource interface {
	fromSourceNode()
}

// FolderSource represents FROM "folder".
type FolderSource struct {
	Path string
}

// TagSource represents FROM #tag.
type TagSource struct {
	Tag string
}

// LinkSource represents FROM [[page]] (incoming links).
type LinkSource struct {
	Target   string
	Outgoing bool // true for outgoing([[page]])
}

// BooleanFromSource represents AND/OR of two FROM sources.
type BooleanFromSource struct {
	Op    string // "AND", "OR"
	Left  FromSource
	Right FromSource
}

// NegatedFromSource represents NOT/negation of a FROM source.
type NegatedFromSource struct {
	Inner FromSource
}

func (FolderSource) fromSourceNode()      {}
func (TagSource) fromSourceNode()         {}
func (LinkSource) fromSourceNode()        {}
func (BooleanFromSource) fromSourceNode() {}
func (NegatedFromSource) fromSourceNode() {}

// --- Expression types ---

// Expr is the interface for all expression nodes.
type Expr interface {
	exprNode()
}

// ComparisonExpr represents a field-operator-value comparison.
// Kept for backward compatibility with existing code that constructs these directly.
type ComparisonExpr struct {
	Field string
	Op    string // "=", "!=", "<", ">", "<=", ">=", "contains", "!contains"
	Value string
}

// LogicalExpr represents AND/OR expressions.
type LogicalExpr struct {
	Op    string // "AND", "OR"
	Left  Expr
	Right Expr
}

// ParenExpr represents a parenthesized expression.
type ParenExpr struct {
	Inner Expr
}

// ExistsExpr represents an exists/!exists check.
type ExistsExpr struct {
	Field   string
	Negated bool
}

// LiteralExpr holds a compile-time literal value.
type LiteralExpr struct {
	Val Value
}

// FieldAccessExpr represents a field reference, possibly dotted (e.g., file.name).
type FieldAccessExpr struct {
	Parts []string // ["file", "name"] for file.name
}

// FunctionCallExpr represents a function call: name(args...).
type FunctionCallExpr struct {
	Name string
	Args []Expr
}

// ArithmeticExpr represents a binary arithmetic operation.
type ArithmeticExpr struct {
	Op    string // "+", "-", "*", "/", "%"
	Left  Expr
	Right Expr
}

// IndexExpr represents array/object indexing: expr[index].
type IndexExpr struct {
	Object Expr
	Index  Expr
}

// LambdaExpr represents a lambda: (x) => expr or (x, y) => expr.
type LambdaExpr struct {
	Params []string
	Body   Expr
}

// NegationExpr represents logical negation: !expr.
type NegationExpr struct {
	Inner Expr
}

func (ComparisonExpr) exprNode()    {}
func (LogicalExpr) exprNode()       {}
func (ParenExpr) exprNode()         {}
func (ExistsExpr) exprNode()        {}
func (LiteralExpr) exprNode()       {}
func (FieldAccessExpr) exprNode()   {}
func (FunctionCallExpr) exprNode()  {}
func (ArithmeticExpr) exprNode()    {}
func (IndexExpr) exprNode()         {}
func (LambdaExpr) exprNode()        {}
func (NegationExpr) exprNode()      {}

// FieldName returns the dotted field name for a FieldAccessExpr,
// or the Field string for a ComparisonExpr (for backward compat helpers).
func FieldName(parts []string) string {
	result := parts[0]
	for _, p := range parts[1:] {
		result += "." + p
	}
	return result
}
