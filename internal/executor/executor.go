package executor

import (
	"database/sql"

	"github.com/andinger/vaultquery/internal/dql"
)

// Executor runs DQL queries against the index store.
type Executor struct {
	store interface{ DB() *sql.DB }
}

// New creates a new Executor backed by the given store.
func New(store interface{ DB() *sql.DB }) *Executor {
	return &Executor{store: store}
}

// Execute runs a parsed DQL query and returns the result.
func (e *Executor) Execute(query *dql.Query) (*Result, error) {
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

	type fileRow struct {
		id    int64
		path  string
		title string
	}
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
		result.Fields = query.Fields
		for _, f := range files {
			row := map[string]any{
				"path":  f.path,
				"title": f.title,
			}
			for _, field := range query.Fields {
				values, err := queryFieldValues(db, f.id, field)
				if err != nil {
					return nil, err
				}
				switch len(values) {
				case 0:
					row[field] = nil
				case 1:
					row[field] = values[0]
				default:
					row[field] = values
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

	return result, nil
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
