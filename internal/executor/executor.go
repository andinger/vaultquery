package executor

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

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
		exprEval := needsExprEval(query.Fields)
		for _, f := range files {
			row := map[string]any{
				"path":  f.path,
				"title": f.title,
			}
			// If any field needs expression evaluation, build a full eval context
			var ctx *eval.EvalContext
			if exprEval {
				allFields, err := queryAllFields(db, f.id)
				if err != nil {
					return nil, err
				}
				meta, err := queryFileMeta(db, f.id)
				if err != nil {
					return nil, err
				}
				ctx = e.buildEvalContext(db, f, allFields, meta)
			}
			for i, fd := range query.Fields {
				displayName := fieldNames[i]
				if isSimpleFieldDef(fd) && ctx == nil {
					// Fast path: simple EAV lookup
					key := dql.FieldDefName(fd)
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
				} else {
					// Expression evaluation path
					if ctx == nil {
						allFields, err := queryAllFields(db, f.id)
						if err != nil {
							return nil, err
						}
						meta, err := queryFileMeta(db, f.id)
						if err != nil {
							return nil, err
						}
						ctx = e.buildEvalContext(db, f, allFields, meta)
					}
					row[displayName] = e.evaluateFieldExpr(fd, ctx)
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
		exprEval := needsExprEval(query.Fields)
		for _, f := range filtered {
			row := map[string]any{
				"path":  f.path,
				"title": f.title,
			}
			var ctx *eval.EvalContext
			if exprEval {
				allFields, err := queryAllFields(db, f.id)
				if err != nil {
					return nil, err
				}
				meta, err := queryFileMeta(db, f.id)
				if err != nil {
					return nil, err
				}
				ctx = e.buildEvalContext(db, f, allFields, meta)
			}
			for i, fd := range query.Fields {
				displayName := fieldNames[i]
				if isSimpleFieldDef(fd) && ctx == nil {
					key := dql.FieldDefName(fd)
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
				} else {
					if ctx == nil {
						allFields, err := queryAllFields(db, f.id)
						if err != nil {
							return nil, err
						}
						meta, err := queryFileMeta(db, f.id)
						if err != nil {
							return nil, err
						}
						ctx = e.buildEvalContext(db, f, allFields, meta)
					}
					row[displayName] = e.evaluateFieldExpr(fd, ctx)
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
	// Check if any sort field uses expressions
	hasExprSort := false
	for _, sf := range sorts {
		if sf.Expr != nil {
			if _, ok := sf.Expr.(dql.FieldAccessExpr); !ok {
				hasExprSort = true
				break
			}
		}
	}

	// Pre-fetch sort field values for each file
	type sortData struct {
		values []dql.Value
	}
	data := make([]sortData, len(files))
	for i, f := range files {
		vals := make([]dql.Value, len(sorts))
		if hasExprSort {
			// Build eval context for expression-based sorting
			allFields, _ := queryAllFields(db, f.id)
			meta, _ := queryFileMeta(db, f.id)
			ctx := e.buildEvalContext(db, f, allFields, meta)
			for j, sf := range sorts {
				if sf.Expr != nil {
					vals[j] = e.eval.Eval(sf.Expr, ctx)
				} else {
					raw, _ := queryFieldValues(db, f.id, sf.Field)
					if len(raw) > 0 {
						vals[j] = dql.CoerceFromString(raw[0])
					} else {
						vals[j] = dql.NewNull()
					}
				}
			}
		} else {
			for j, sf := range sorts {
				field := sf.Field
				if field == "" && sf.Expr != nil {
					if fa, ok := sf.Expr.(dql.FieldAccessExpr); ok {
						field = dql.FieldName(fa.Parts)
					}
				}
				raw, _ := queryFieldValues(db, f.id, field)
				if len(raw) > 0 {
					vals[j] = dql.CoerceFromString(raw[0])
				} else {
					vals[j] = dql.NewNull()
				}
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

// isSimpleFieldDef returns true if the FieldDef is a plain field access (e.g. "customer" or "file.name"),
// not a function call, arithmetic, etc.
func isSimpleFieldDef(fd dql.FieldDef) bool {
	_, ok := fd.Expr.(dql.FieldAccessExpr)
	return ok
}

// needsExprEval returns true if any TABLE field requires Go-side expression evaluation.
func needsExprEval(fields []dql.FieldDef) bool {
	for _, fd := range fields {
		if !isSimpleFieldDef(fd) {
			return true
		}
	}
	return false
}

// fileMeta holds metadata from the files table.
type fileMeta struct {
	mtime int64
	ctime int64
	size  int64
}

func queryFileMeta(db *sql.DB, fileID int64) (fileMeta, error) {
	var m fileMeta
	err := db.QueryRow("SELECT mtime, ctime, size FROM files WHERE id = ?", fileID).Scan(&m.mtime, &m.ctime, &m.size)
	return m, err
}

// buildEvalContext creates a full EvalContext for a file, including file.* implicit fields.
func (e *Executor) buildEvalContext(db *sql.DB, f fileRow, fields map[string][]string, meta fileMeta) *eval.EvalContext {
	ctx := eval.BuildEvalContextFromEAV(f.path, f.title, fields)

	// Populate file.* implicit fields
	base := filepath.Base(f.path)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	folder := filepath.Dir(f.path)

	ctx.Fields["file.path"] = dql.NewString(f.path)
	ctx.Fields["file.folder"] = dql.NewString(folder)
	ctx.Fields["file.name"] = dql.NewString(name)
	ctx.Fields["file.ext"] = dql.NewString(ext)
	ctx.Fields["file.link"] = dql.NewLink(name)
	ctx.Fields["file.size"] = dql.NewNumber(float64(meta.size))

	if meta.mtime > 0 {
		mt := time.Unix(meta.mtime, 0)
		ctx.Fields["file.mtime"] = dql.NewDate(mt)
		ctx.Fields["file.mday"] = dql.NewDate(time.Date(mt.Year(), mt.Month(), mt.Day(), 0, 0, 0, 0, mt.Location()))
	}
	if meta.ctime > 0 {
		ct := time.Unix(meta.ctime, 0)
		ctx.Fields["file.ctime"] = dql.NewDate(ct)
		ctx.Fields["file.cday"] = dql.NewDate(time.Date(ct.Year(), ct.Month(), ct.Day(), 0, 0, 0, 0, ct.Location()))
	}

	// file.day — date parsed from filename
	if d, ok := dql.ParseDateFromFilename(base); ok {
		ctx.Fields["file.day"] = dql.NewDate(d)
	}

	return ctx
}

// evaluateFieldExpr evaluates a single TABLE field expression and returns the display value.
func (e *Executor) evaluateFieldExpr(fd dql.FieldDef, ctx *eval.EvalContext) any {
	val := e.eval.Eval(fd.Expr, ctx)
	if val.IsNull() {
		return nil
	}
	return valueToDisplay(val)
}

// valueToDisplay converts a dql.Value to a display-friendly string or native type.
func valueToDisplay(v dql.Value) any {
	switch v.Type {
	case dql.TypeString:
		s, _ := v.AsString()
		return s
	case dql.TypeNumber:
		n, _ := v.AsNumber()
		if n == float64(int64(n)) {
			return fmt.Sprintf("%d", int64(n))
		}
		return fmt.Sprintf("%g", n)
	case dql.TypeBool:
		b, _ := v.AsBool()
		if b {
			return "true"
		}
		return "false"
	case dql.TypeDate:
		d, _ := v.AsDate()
		return d.Format("2006-01-02T15:04:05")
	case dql.TypeDuration:
		d, _ := v.AsDuration()
		return d.String()
	case dql.TypeLink:
		s, _ := v.AsLink()
		return s
	case dql.TypeList:
		items, _ := v.AsList()
		parts := make([]string, len(items))
		for i, item := range items {
			parts[i] = fmt.Sprintf("%v", valueToDisplay(item))
		}
		return strings.Join(parts, ", ")
	default:
		return fmt.Sprintf("%v", v.Inner)
	}
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
