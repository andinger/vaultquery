package executor

import (
	"database/sql"
	"sort"

	"github.com/andinger/vaultquery/internal/dql"
	"github.com/andinger/vaultquery/internal/eval"
)

// Executor runs DQL queries against the index store.
type Executor struct {
	store interface{ DB() *sql.DB }
	eval  *eval.Evaluator
}

// New creates a new Executor backed by the given store.
func New(store interface{ DB() *sql.DB }) *Executor {
	ev := eval.New()
	eval.RegisterBuiltins(ev)
	return &Executor{
		store: store,
		eval:  ev,
	}
}

// Evaluator returns the evaluator for registering functions.
func (e *Executor) Evaluator() *eval.Evaluator {
	return e.eval
}

// Execute runs a parsed DQL query and returns the result.
func (e *Executor) Execute(query *dql.Query) (*Result, error) {
	// Handle TASK and CALENDAR modes
	if query.Mode == "TASK" {
		return e.executeTask(query)
	}
	if query.Mode == "CALENDAR" {
		return e.executeCalendar(query)
	}

	// Determine if WHERE can be fully pushed to SQL
	canPush := query.Where == nil || CanPushToSQL(query.Where)

	if canPush {
		return e.executePushDown(query)
	}
	return e.executeHybrid(query)
}

// executePushDown handles queries where everything can be done in SQL (backward compat path).
func (e *Executor) executePushDown(query *dql.Query) (*Result, error) {
	sqlStr, args, err := GenerateSQL(query)
	if err != nil {
		return nil, err
	}

	db := e.store.DB()

	rows, err := db.Query(sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var files []fileRow
	for rows.Next() {
		var fr fileRow
		if err := rows.Scan(&fr.id, &fr.path, &fr.title); err != nil {
			return nil, err
		}
		files = append(files, fr)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := &Result{
		Mode:    query.Mode,
		Results: make([]map[string]any, 0, len(files)),
	}

	if query.Mode == "TABLE" {
		fieldNames := dql.FieldDefNames(query.Fields)
		result.Fields = fieldNames
		for _, f := range files {
			row := map[string]any{
				"path":  f.path,
				"title": f.title,
			}
			for i, fd := range query.Fields {
				key := dql.FieldDefName(fd)
				displayName := fieldNames[i]
				values, err := queryFieldValues(db, f.id, key)
				if err != nil {
					return nil, err
				}
				switch len(values) {
				case 0:
					row[displayName] = nil
				case 1:
					row[displayName] = values[0]
				default:
					row[displayName] = values
				}
			}
			result.Results = append(result.Results, row)
		}
	} else {
		for _, f := range files {
			result.Results = append(result.Results, map[string]any{
				"path":  f.path,
				"title": f.title,
			})
		}
	}

	// Post-processing: FLATTEN
	result.Results = applyFlatten(result.Results, query.Flatten, e.eval)

	return result, nil
}

// executeHybrid: SQL for candidate set, Go for expression evaluation.
func (e *Executor) executeHybrid(query *dql.Query) (*Result, error) {
	db := e.store.DB()

	// Build a simplified SQL query: push FROM and pushable WHERE parts only
	pushableQuery := &dql.Query{
		Mode: query.Mode,
		From: query.From,
	}

	// Split WHERE into pushable and Go-evaluated parts
	if query.Where != nil {
		pushable, goEval := splitWhere(query.Where)
		pushableQuery.Where = pushable
		_ = goEval // evaluated below
	}

	sqlStr, args, err := GenerateSQL(pushableQuery)
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var candidates []fileRow
	for rows.Next() {
		var fr fileRow
		if err := rows.Scan(&fr.id, &fr.path, &fr.title); err != nil {
			return nil, err
		}
		candidates = append(candidates, fr)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Filter candidates through Go evaluator
	var filtered []fileRow
	for _, f := range candidates {
		fields, err := queryAllFields(db, f.id)
		if err != nil {
			return nil, err
		}
		ctx := eval.BuildEvalContextFromEAV(f.path, f.title, fields)

		if query.Where == nil || e.eval.EvalBool(query.Where, ctx) {
			filtered = append(filtered, f)
		}
	}

	// Apply SORT in Go
	if len(query.Sort) > 0 {
		e.sortRows(db, filtered, query.Sort)
	}

	// Apply LIMIT
	if query.Limit > 0 && len(filtered) > query.Limit {
		filtered = filtered[:query.Limit]
	}

	// Build result rows
	var resultRows []map[string]any

	if query.Mode == "TABLE" {
		fieldNames := dql.FieldDefNames(query.Fields)
		for _, f := range filtered {
			row := map[string]any{
				"path":  f.path,
				"title": f.title,
			}
			for i, fd := range query.Fields {
				key := dql.FieldDefName(fd)
				displayName := fieldNames[i]
				values, err := queryFieldValues(db, f.id, key)
				if err != nil {
					return nil, err
				}
				switch len(values) {
				case 0:
					row[displayName] = nil
				case 1:
					row[displayName] = values[0]
				default:
					row[displayName] = values
				}
			}
			resultRows = append(resultRows, row)
		}
	} else {
		for _, f := range filtered {
			resultRows = append(resultRows, map[string]any{
				"path":  f.path,
				"title": f.title,
			})
		}
	}

	// Post-processing: FLATTEN then GROUP BY
	resultRows = applyFlatten(resultRows, query.Flatten, e.eval)

	result := &Result{
		Mode:    query.Mode,
		Results: resultRows,
	}
	if query.Mode == "TABLE" {
		result.Fields = dql.FieldDefNames(query.Fields)
	}

	return result, nil
}

type fileRow struct {
	id    int64
	path  string
	title string
}

func (e *Executor) sortRows(db *sql.DB, files []fileRow, sorts []dql.SortField) {
	// Pre-fetch sort field values for each file
	type sortData struct {
		values []dql.Value
	}
	data := make([]sortData, len(files))
	for i, f := range files {
		vals := make([]dql.Value, len(sorts))
		for j, sf := range sorts {
			raw, _ := queryFieldValues(db, f.id, sf.Field)
			if len(raw) > 0 {
				vals[j] = dql.CoerceFromString(raw[0])
			} else {
				vals[j] = dql.NewNull()
			}
		}
		data[i] = sortData{values: vals}
	}

	sort.SliceStable(files, func(a, b int) bool {
		for j, sf := range sorts {
			cmp := data[a].values[j].Compare(data[b].values[j])
			if cmp == 0 {
				continue
			}
			if sf.Desc {
				return cmp > 0
			}
			return cmp < 0
		}
		return false
	})
}

// splitWhere splits a WHERE expression into SQL-pushable and Go-evaluated parts.
// For now: if the whole thing is pushable, push it all. Otherwise, nothing is pushed.
func splitWhere(expr dql.Expr) (pushable dql.Expr, goEval dql.Expr) {
	if CanPushToSQL(expr) {
		return expr, nil
	}
	return nil, expr
}

func queryFieldValues(db *sql.DB, fileID int64, key string) ([]string, error) {
	rows, err := db.Query("SELECT value FROM fields WHERE file_id = ? AND key = ?", fileID, key)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var values []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		values = append(values, v)
	}
	return values, rows.Err()
}

func queryAllFields(db *sql.DB, fileID int64) (map[string][]string, error) {
	rows, err := db.Query("SELECT key, value FROM fields WHERE file_id = ?", fileID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	fields := make(map[string][]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		fields[k] = append(fields[k], v)
	}
	return fields, rows.Err()
}
