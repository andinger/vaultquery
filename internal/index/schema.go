package index

import "database/sql"

const schema = `
CREATE TABLE IF NOT EXISTS files (
    id    INTEGER PRIMARY KEY AUTOINCREMENT,
    path  TEXT NOT NULL UNIQUE,
    mtime INTEGER NOT NULL,
    size  INTEGER NOT NULL,
    title TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS fields (
    file_id INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    key     TEXT NOT NULL,
    value   TEXT NOT NULL,
    PRIMARY KEY (file_id, key, value)
);

CREATE INDEX IF NOT EXISTS idx_fields_key_value ON fields(key, value);

CREATE TABLE IF NOT EXISTS meta (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
`

func migrate(db *sql.DB) error {
	_, err := db.Exec(schema)
	return err
}
