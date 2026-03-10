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
	if query.FromSource != nil {
		fromSQL, fromArgs, err := buildFromSource(query.FromSource)
		if err != nil {
			return "", nil, err
		}
		if fromSQL != "" {
			where = append(where, fromSQL)
			whereArgs = append(whereArgs, fromArgs...)
		}
	} else if query.From != "" {
		fromSQL, fromArgs := buildFolderMatch(query.From)
		where = append(where, fromSQL)
		whereArgs = append(whereArgs, fromArgs...)
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
	// null comparison: e.Value == "" means the parser saw the null keyword.
	// "field = null" → field does NOT exist; "field != null" → field EXISTS.
	if e.Value == "" {
		switch e.Op {
		case "=":
			return "f.id NOT IN (SELECT file_id FROM fields WHERE key = ?)",
				[]any{e.Field}, nil
		case "!=":
			return "f.id IN (SELECT file_id FROM fields WHERE key = ?)",
				[]any{e.Field}, nil
		default:
			// Other comparisons against null always fail (Dataview semantics).
			return "0=1", nil, nil
		}
	}

	switch e.Op {
	case "=", "contains":
		return "f.id IN (SELECT file_id FROM fields WHERE key = ? AND value = ?)",
			[]any{e.Field, e.Value}, nil
	case "!=", "!contains":
		// Exclude files that have this key with this value, but also include
		// files that don't have the field at all (Dataview semantics: missing
		// field is null, and null != "anything" is true... but actually in
		// Dataview, null != "value" is false). We keep the existing behavior
		// for non-null != comparisons.
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

func buildFromSource(src dql.FromSource) (string, []any, error) {
	switch s := src.(type) {
	case dql.FolderSource:
		sql, args := buildFolderMatch(s.Path)
		return sql, args, nil

	case dql.TagSource:
		return "f.id IN (SELECT file_id FROM tags WHERE tag = ?)", []any{s.Tag}, nil

	case dql.LinkSource:
		if s.Outgoing {
			// Files that are linked FROM the target page
			return "f.id IN (SELECT l.file_id FROM links l JOIN files tf ON tf.path LIKE ? || '%' OR tf.title = ? WHERE l.file_id = tf.id)",
				[]any{s.Target, s.Target}, nil
		}
		// Files that link TO the target page (incoming links)
		return "f.id IN (SELECT file_id FROM links WHERE target = ?)", []any{s.Target}, nil

	case dql.BooleanFromSource:
		leftSQL, leftArgs, err := buildFromSource(s.Left)
		if err != nil {
			return "", nil, err
		}
		rightSQL, rightArgs, err := buildFromSource(s.Right)
		if err != nil {
			return "", nil, err
		}
		sql := fmt.Sprintf("(%s %s %s)", leftSQL, s.Op, rightSQL)
		return sql, append(leftArgs, rightArgs...), nil

	case dql.NegatedFromSource:
		innerSQL, innerArgs, err := buildFromSource(s.Inner)
		if err != nil {
			return "", nil, err
		}
		return fmt.Sprintf("NOT (%s)", innerSQL), innerArgs, nil

	default:
		return "", nil, fmt.Errorf("unsupported FROM source type: %T", src)
	}
}

// buildFolderMatch generates SQL for a folder path, supporting * wildcards.
// Without wildcards: exact prefix match (existing behavior).
// With wildcards: * maps to SQL %, with a special case for leading */ to also
// match root-level folders.
func buildFolderMatch(path string) (string, []any) {
	if strings.Contains(path, "*") {
		// Wildcard mode: convert * to SQL %
		if !strings.HasSuffix(path, "/") && !strings.HasSuffix(path, "*") {
			path += "/"
		}
		pattern := strings.ReplaceAll(path, "*", "%")
		if !strings.HasSuffix(pattern, "%") {
			pattern += "%"
		}
		// Leading %/ edge case: also match root-level paths
		if strings.HasPrefix(pattern, "%/") {
			rootPattern := pattern[2:] // strip leading "%/"
			return "(f.path LIKE ? OR f.path LIKE ?)", []any{pattern, rootPattern}
		}
		return "f.path LIKE ?", []any{pattern}
	}
	// Existing prefix-match behavior
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	return "f.path LIKE ? || '%'", []any{path}
}

// CanPushToSQL returns true if the expression can be fully evaluated in SQL.
// Used by the hybrid executor to decide what to push down.
func CanPushToSQL(expr dql.Expr) bool {
	switch e := expr.(type) {
	case dql.ComparisonExpr:
		return true
	case dql.ExistsExpr:
		return true
	case dql.LogicalExpr:
		return CanPushToSQL(e.Left) && CanPushToSQL(e.Right)
	case dql.ParenExpr:
		return CanPushToSQL(e.Inner)
	default:
		return false
	}
}
