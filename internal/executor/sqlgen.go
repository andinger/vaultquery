package executor

import (
	"fmt"
	"strings"

	"github.com/andinger/vaultquery/internal/dql"
)

// GenerateSQL translates a parsed DQL query into a SQL statement with args.
func GenerateSQL(query *dql.Query) (string, []any, error) {
	var (
		joinArgs  []any
		whereArgs []any
		joins     []string
		where     []string
		orderBy   []string
		sortIdx   int
	)

	base := "SELECT DISTINCT f.id, f.path, f.title FROM files f"

	// SORT clause (joins come before WHERE in SQL)
	for _, sf := range query.Sort {
		alias := fmt.Sprintf("sort%d", sortIdx)
		joins = append(joins, fmt.Sprintf("LEFT JOIN fields %s ON %s.file_id = f.id AND %s.key = ?", alias, alias, alias))
		joinArgs = append(joinArgs, sf.Field)
		dir := "ASC"
		if sf.Desc {
			dir = "DESC"
		}
		orderBy = append(orderBy, fmt.Sprintf("%s.value %s", alias, dir))
		sortIdx++
	}

	// FROM clause
	if query.From != "" {
		from := query.From
		if !strings.HasSuffix(from, "/") {
			from += "/"
		}
		where = append(where, "f.path LIKE ? || '%'")
		whereArgs = append(whereArgs, from)
	}

	// WHERE clause
	if query.Where != nil {
		sql, wArgs, err := buildWhere(query.Where)
		if err != nil {
			return "", nil, err
		}
		where = append(where, sql)
		whereArgs = append(whereArgs, wArgs...)
	}

	// Build final SQL; args order must match SQL placeholder order
	var sb strings.Builder
	sb.WriteString(base)
	for _, j := range joins {
		sb.WriteString(" ")
		sb.WriteString(j)
	}
	if len(where) > 0 {
		sb.WriteString(" WHERE ")
		sb.WriteString(strings.Join(where, " AND "))
	}
	if len(orderBy) > 0 {
		sb.WriteString(" ORDER BY ")
		sb.WriteString(strings.Join(orderBy, ", "))
	}

	args := make([]any, 0, len(joinArgs)+len(whereArgs)+1)
	args = append(args, joinArgs...)
	args = append(args, whereArgs...)

	if query.Limit > 0 {
		sb.WriteString(" LIMIT ?")
		args = append(args, query.Limit)
	}

	return sb.String(), args, nil
}

func buildWhere(expr dql.Expr) (string, []any, error) {
	switch e := expr.(type) {
	case dql.ComparisonExpr:
		return buildComparison(e)
	case dql.ExistsExpr:
		return buildExists(e)
	case dql.LogicalExpr:
		left, lArgs, err := buildWhere(e.Left)
		if err != nil {
			return "", nil, err
		}
		right, rArgs, err := buildWhere(e.Right)
		if err != nil {
			return "", nil, err
		}
		sql := fmt.Sprintf("(%s %s %s)", left, e.Op, right)
		return sql, append(lArgs, rArgs...), nil
	case dql.ParenExpr:
		return buildWhere(e.Inner)
	default:
		return "", nil, fmt.Errorf("unsupported expression type: %T", expr)
	}
}

func buildComparison(e dql.ComparisonExpr) (string, []any, error) {
	switch e.Op {
	case "=", "contains":
		return "f.id IN (SELECT file_id FROM fields WHERE key = ? AND value = ?)",
			[]any{e.Field, e.Value}, nil
	case "!=", "!contains":
		return "f.id NOT IN (SELECT file_id FROM fields WHERE key = ? AND value = ?)",
			[]any{e.Field, e.Value}, nil
	case "<", ">", "<=", ">=":
		return fmt.Sprintf("f.id IN (SELECT file_id FROM fields WHERE key = ? AND CAST(value AS REAL) %s CAST(? AS REAL))", e.Op),
			[]any{e.Field, e.Value}, nil
	default:
		return "", nil, fmt.Errorf("unsupported comparison operator: %s", e.Op)
	}
}

func buildExists(e dql.ExistsExpr) (string, []any, error) {
	if e.Negated {
		return "f.id NOT IN (SELECT file_id FROM fields WHERE key = ?)",
			[]any{e.Field}, nil
	}
	return "f.id IN (SELECT file_id FROM fields WHERE key = ?)",
		[]any{e.Field}, nil
}
