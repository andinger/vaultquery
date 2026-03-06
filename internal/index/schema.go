package index

import "database/sql"

const schema = `
CREATE TABLE IF NOT EXISTS files (
    id    INTEGER PRIMARY KEY AUTOINCREMENT,
    path  TEXT NOT NULL UNIQUE,
    mtime INTEGER NOT NULL,
    size  INTEGER NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    ctime INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS fields (
    file_id INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    key     TEXT NOT NULL,
    value   TEXT NOT NULL,
    PRIMARY KEY (file_id, key, value)
);

CREATE INDEX IF NOT EXISTS idx_fields_key_value ON fields(key, value);

CREATE TABLE IF NOT EXISTS tags (
    file_id INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    tag     TEXT NOT NULL,
    PRIMARY KEY (file_id, tag)
);

CREATE INDEX IF NOT EXISTS idx_tags_tag ON tags(tag);

CREATE TABLE IF NOT EXISTS links (
    file_id INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    target  TEXT NOT NULL,
    PRIMARY KEY (file_id, target)
);

CREATE INDEX IF NOT EXISTS idx_links_target ON links(target);

CREATE TABLE IF NOT EXISTS tasks (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    file_id   INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    line      INTEGER NOT NULL,
    text      TEXT NOT NULL,
    completed INTEGER NOT NULL DEFAULT 0,
    section   TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_tasks_file_id ON tasks(file_id);

CREATE TABLE IF NOT EXISTS meta (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
`

func migrate(db *sql.DB) error {
	_, err := db.Exec(schema)
	if err != nil {
		return err
	}

	// Run migrations for existing databases
	migrations := []string{
		"ALTER TABLE files ADD COLUMN ctime INTEGER NOT NULL DEFAULT 0",
	}
	for _, m := range migrations {
		// Ignore errors (column already exists, etc.)
		_, _ = db.Exec(m)
	}

	// Create tables that may not exist in older databases
	newTables := []string{
		`CREATE TABLE IF NOT EXISTS tags (
			file_id INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
			tag     TEXT NOT NULL,
			PRIMARY KEY (file_id, tag)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_tags_tag ON tags(tag)`,
		`CREATE TABLE IF NOT EXISTS links (
			file_id INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
			target  TEXT NOT NULL,
			PRIMARY KEY (file_id, target)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_links_target ON links(target)`,
		`CREATE TABLE IF NOT EXISTS tasks (
			id        INTEGER PRIMARY KEY AUTOINCREMENT,
			file_id   INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
			line      INTEGER NOT NULL,
			text      TEXT NOT NULL,
			completed INTEGER NOT NULL DEFAULT 0,
			section   TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_file_id ON tasks(file_id)`,
	}
	for _, s := range newTables {
		if _, err := db.Exec(s); err != nil {
			return err
		}
	}

	return nil
}
