package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

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
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			name       TEXT    NOT NULL,
			path       TEXT    NOT NULL UNIQUE,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS commits (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			repository_id INTEGER NOT NULL REFERENCES repositories(id),
			hash          TEXT    NOT NULL,
			message       TEXT,
			author        TEXT,
			committed_at  DATETIME,
			reviewed      INTEGER NOT NULL DEFAULT 0,
			created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(repository_id, hash)
		)`,

		`CREATE TABLE IF NOT EXISTS sessions (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			repository_id   INTEGER REFERENCES repositories(id),
			started_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			ended_at        DATETIME,
			total_questions INTEGER NOT NULL DEFAULT 0,
			correct_count   INTEGER NOT NULL DEFAULT 0,
			max_streak      INTEGER NOT NULL DEFAULT 0
		)`,

		`CREATE TABLE IF NOT EXISTS questions (
			id             INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id     INTEGER NOT NULL REFERENCES sessions(id),
			commit_id      INTEGER REFERENCES commits(id),
			title          TEXT,
			content        TEXT    NOT NULL,
			question_type  TEXT    NOT NULL,
			choices        TEXT,
			correct_answer TEXT,
			category       TEXT,
			saved          INTEGER NOT NULL DEFAULT 0,
			created_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS answers (
			id             INTEGER PRIMARY KEY AUTOINCREMENT,
			question_id    INTEGER NOT NULL REFERENCES questions(id),
			user_answer    TEXT,
			is_correct     INTEGER,
			ai_score       INTEGER,
			ai_explanation TEXT,
			grade_status   TEXT    NOT NULL,
			self_rating    TEXT,
			answered_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS reviews (
			id               INTEGER PRIMARY KEY AUTOINCREMENT,
			question_id      INTEGER NOT NULL UNIQUE REFERENCES questions(id),
			interval         INTEGER NOT NULL DEFAULT 1,
			easiness_factor  REAL    NOT NULL DEFAULT 2.5,
			repetitions      INTEGER NOT NULL DEFAULT 0,
			next_review_at   DATE    NOT NULL,
			last_reviewed_at DATETIME,
			created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS review_logs (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			review_id   INTEGER NOT NULL REFERENCES reviews(id),
			self_rating TEXT    NOT NULL,
			reviewed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, stmt := range statements {
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("create table: %w", err)
		}
	}
	return nil
}
