package executor

import (
	"database/sql"

	"github.com/andinger/vaultquery/internal/dql"
	"github.com/andinger/vaultquery/internal/eval"
)

// executeTask runs a TASK query, returning tasks filtered by WHERE.
func (e *Executor) executeTask(query *dql.Query) (*Result, error) {
	db := e.store.DB()

	// Get candidate files first (using FROM/WHERE for file filtering)
	fileQuery := &dql.Query{
		Mode:       "LIST",
		From:       query.From,
		FromSource: query.FromSource,
	}

	sqlStr, args, err := GenerateSQL(fileQuery)
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var fileIDs []int64
	fileMap := make(map[int64]fileRow)
	for rows.Next() {
		var fr fileRow
		if err := rows.Scan(&fr.id, &fr.path, &fr.title); err != nil {
			return nil, err
		}
		fileIDs = append(fileIDs, fr.id)
		fileMap[fr.id] = fr
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := &Result{
		Mode:    "TASK",
		Results: make([]map[string]any, 0),
	}

	for _, fid := range fileIDs {
		tasks, err := queryTasks(db, fid)
		if err != nil {
			return nil, err
		}
		fr := fileMap[fid]

		for _, task := range tasks {
			row := map[string]any{
				"path":      fr.path,
				"title":     fr.title,
				"text":      task.text,
				"completed": task.completed,
				"line":      task.line,
				"section":   task.section,
			}

			// Apply WHERE filter if present
			if query.Where != nil {
				ctx := buildTaskEvalContext(fr, task)
				if !e.eval.EvalBool(query.Where, ctx) {
					continue
				}
			}

			result.Results = append(result.Results, row)
		}
	}

	// Apply LIMIT
	if query.Limit > 0 && len(result.Results) > query.Limit {
		result.Results = result.Results[:query.Limit]
	}

	return result, nil
}

// executeCalendar runs a CALENDAR query, grouping results by date.
func (e *Executor) executeCalendar(query *dql.Query) (*Result, error) {
	// Calendar is essentially a LIST/TABLE with date grouping
	// For now, return results with a date field
	listQuery := &dql.Query{
		Mode:       "LIST",
		From:       query.From,
		FromSource: query.FromSource,
		Where:      query.Where,
		Sort:       query.Sort,
		Limit:      query.Limit,
	}

	result, err := e.Execute(listQuery)
	if err != nil {
		return nil, err
	}
	result.Mode = "CALENDAR"
	return result, nil
}

type taskRow struct {
	line      int
	text      string
	completed bool
	section   string
}

func queryTasks(db *sql.DB, fileID int64) ([]taskRow, error) {
	rows, err := db.Query(
		"SELECT line, text, completed, section FROM tasks WHERE file_id = ? ORDER BY line",
		fileID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var tasks []taskRow
	for rows.Next() {
		var t taskRow
		var completed int
		if err := rows.Scan(&t.line, &t.text, &completed, &t.section); err != nil {
			return nil, err
		}
		t.completed = completed != 0
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func buildTaskEvalContext(fr fileRow, task taskRow) *eval.EvalContext {
	fields := map[string]dql.Value{
		"text":      dql.NewString(task.text),
		"completed": dql.NewBool(task.completed),
		"line":      dql.NewNumber(float64(task.line)),
		"section":   dql.NewString(task.section),
		"path":      dql.NewString(fr.path),
		"title":     dql.NewString(fr.title),
	}
	return eval.NewEvalContext(fr.path, fr.title, fields)
}
