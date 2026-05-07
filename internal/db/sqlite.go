package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

const currentSchemaVersion = 1

// Open opens (or creates) the SQLite database at ~/.osarai/data.db.
// It applies PRAGMAs and runs any pending migrations before returning.
func Open() (*sql.DB, error) {
	dir := filepath.Join(os.Getenv("HOME"), ".osarai")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	return openDB(filepath.Join(dir, "data.db"))
}

func openDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	if err := applyPragmas(db); err != nil {
		db.Close()
		return nil, err
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func applyPragmas(db *sql.DB) error {
	pragmas := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
		"PRAGMA busy_timeout = 5000",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			return fmt.Errorf("pragma %q: %w", p, err)
		}
	}
	return nil
}

func migrate(db *sql.DB) error {
	var version int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return fmt.Errorf("get user_version: %w", err)
	}

	if version >= currentSchemaVersion {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin migration tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := createTables(tx); err != nil {
		return err
	}

	if _, err := tx.Exec(fmt.Sprintf("PRAGMA user_version = %d", currentSchemaVersion)); err != nil {
		return fmt.Errorf("set user_version: %w", err)
	}

	return tx.Commit()
}

func createTables(tx *sql.Tx) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS repositories (
			id         INTEGER  PRIMARY KEY AUTOINCREMENT,
			path       TEXT     NOT NULL UNIQUE,
			name       TEXT     NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS commits (
			id            INTEGER  PRIMARY KEY AUTOINCREMENT,
			repository_id INTEGER  NOT NULL REFERENCES repositories(id),
			hash          TEXT     NOT NULL,
			message       TEXT,
			diff_body     TEXT,
			reviewed      INTEGER  NOT NULL DEFAULT 0,
			created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(repository_id, hash)
		)`,

		`CREATE TABLE IF NOT EXISTS sessions (
			id            INTEGER  PRIMARY KEY AUTOINCREMENT,
			repository_id INTEGER  NOT NULL REFERENCES repositories(id),
			commit_hash   TEXT,
			diff_scope    TEXT,
			started_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			finished_at   DATETIME
		)`,

		`CREATE TABLE IF NOT EXISTS questions (
			id            INTEGER  PRIMARY KEY AUTOINCREMENT,
			session_id    INTEGER  NOT NULL REFERENCES sessions(id),
			commit_id     INTEGER  REFERENCES commits(id),
			title         TEXT,
			body          TEXT     NOT NULL,
			question_type TEXT     NOT NULL,
			choices       TEXT,
			answer        TEXT,
			explanation   TEXT,
			saved_for_md  INTEGER  NOT NULL DEFAULT 0,
			created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS answers (
			id           INTEGER  PRIMARY KEY AUTOINCREMENT,
			question_id  INTEGER  NOT NULL REFERENCES questions(id),
			user_answer  TEXT,
			is_correct   INTEGER,
			score        INTEGER,
			grade_status TEXT     NOT NULL,
			explanation  TEXT,
			created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS reviews (
			id             INTEGER  PRIMARY KEY AUTOINCREMENT,
			question_id    INTEGER  NOT NULL UNIQUE REFERENCES questions(id),
			interval       REAL     NOT NULL DEFAULT 1.0,
			repetitions    INTEGER  NOT NULL DEFAULT 0,
			ease_factor    REAL     NOT NULL DEFAULT 2.5,
			next_review_at DATE     NOT NULL,
			created_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS review_logs (
			id          INTEGER  PRIMARY KEY AUTOINCREMENT,
			review_id   INTEGER  NOT NULL REFERENCES reviews(id),
			rating      INTEGER  NOT NULL,
			created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, stmt := range statements {
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("create table: %w", err)
		}
	}
	return nil
}

// parseTime は SQLite の DATETIME 文字列を time.Time に変換する。
func parseTime(b []byte) (time.Time, error) {
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, string(b)); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse time: %q", string(b))
}
