package dql

// Query represents a parsed DQL query.
type Query struct {
	Mode    string      // "TABLE" or "LIST"
	Fields  []string    // TABLE fields (empty for LIST)
	From    string      // FROM path (empty = all)
	Where   Expr        // nil if no WHERE
	Sort    []SortField // nil if no SORT
	Limit   int         // 0 = no limit
	GroupBy []string
	Flatten []string
}

// SortField represents a field with sort direction.
type SortField struct {
	Field string
	Desc  bool
}

// Expr is the interface for all expression nodes.
type Expr interface {
	exprNode()
}

// ComparisonExpr represents a field-operator-value comparison.
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

func (ComparisonExpr) exprNode() {}
func (LogicalExpr) exprNode()    {}
func (ParenExpr) exprNode()      {}
func (ExistsExpr) exprNode()     {}
