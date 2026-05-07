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

// Open opens (or creates) ~/.osarai/data.db and returns a Store.
func Open() (Store, error) {
	dir := filepath.Join(os.Getenv("HOME"), ".osarai")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	return openStore(filepath.Join(dir, "data.db"))
}

func openStore(path string) (Store, error) {
	db, err := openDB(path)
	if err != nil {
		return nil, err
	}
	return newStore(db), nil
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

func newStore(db *sql.DB) Store {
	return &sqlStore{
		db:           db,
		repositories: NewRepositoryStore(db),
		commits:      NewCommitStore(db),
		sessions:     NewSessionStore(db),
		questions:    NewQuestionStore(db),
		answers:      NewAnswerStore(db),
		reviews:      NewReviewStore(db),
		reviewLogs:   NewReviewLogStore(db),
	}
}

type sqlStore struct {
	db           *sql.DB
	repositories RepositoryStore
	commits      CommitStore
	sessions     SessionStore
	questions    QuestionStore
	answers      AnswerStore
	reviews      ReviewStore
	reviewLogs   ReviewLogStore
}

func (s *sqlStore) Repositories() RepositoryStore { return s.repositories }
func (s *sqlStore) Commits() CommitStore          { return s.commits }
func (s *sqlStore) Sessions() SessionStore         { return s.sessions }
func (s *sqlStore) Questions() QuestionStore       { return s.questions }
func (s *sqlStore) Answers() AnswerStore           { return s.answers }
func (s *sqlStore) Reviews() ReviewStore           { return s.reviews }
func (s *sqlStore) ReviewLogs() ReviewLogStore     { return s.reviewLogs }
func (s *sqlStore) Close() error                   { return s.db.Close() }

func applyPragmas(db *sql.DB) error {
	for _, p := range []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
		"PRAGMA busy_timeout = 5000",
	} {
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
	if err := createIndexes(tx); err != nil {
		return err
	}
	if _, err := tx.Exec(fmt.Sprintf("PRAGMA user_version = %d", currentSchemaVersion)); err != nil {
		return fmt.Errorf("set user_version: %w", err)
	}
	return tx.Commit()
}

func createTables(tx *sql.Tx) error {
	for _, stmt := range []string{
		`CREATE TABLE IF NOT EXISTS repositories (
			id         INTEGER  PRIMARY KEY AUTOINCREMENT,
			name       TEXT     NOT NULL UNIQUE,
			path       TEXT     NOT NULL UNIQUE,
			remote_url TEXT,
			created_at TEXT     NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS commits (
			id            INTEGER  PRIMARY KEY AUTOINCREMENT,
			repository_id INTEGER  NOT NULL REFERENCES repositories(id),
			hash          TEXT     NOT NULL CHECK(length(hash) = 40),
			message       TEXT     NOT NULL,
			author_name   TEXT     NOT NULL,
			author_email  TEXT     NOT NULL,
			diff_summary  TEXT,
			reviewed      INTEGER  NOT NULL DEFAULT 0,
			committed_at  TEXT     NOT NULL,
			created_at    TEXT     NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(repository_id, hash)
		)`,

		`CREATE TABLE IF NOT EXISTS sessions (
			id              INTEGER  PRIMARY KEY AUTOINCREMENT,
			mode            TEXT     NOT NULL CHECK(mode IN ('check','recall')),
			repository_id   INTEGER  REFERENCES repositories(id),
			source_ref      TEXT,
			total_questions INTEGER  NOT NULL DEFAULT 0,
			correct_count   INTEGER  NOT NULL DEFAULT 0,
			max_streak      INTEGER  NOT NULL DEFAULT 0,
			started_at      TEXT     NOT NULL DEFAULT CURRENT_TIMESTAMP,
			finished_at     TEXT
		)`,

		`CREATE TABLE IF NOT EXISTS questions (
			id             INTEGER  PRIMARY KEY AUTOINCREMENT,
			session_id     INTEGER  NOT NULL REFERENCES sessions(id),
			commit_id      INTEGER  REFERENCES commits(id),
			title          TEXT     NOT NULL,
			category       TEXT     NOT NULL CHECK(category IN ('design','language','framework')),
			question_type  TEXT     NOT NULL CHECK(question_type IN ('choice','written')),
			body           TEXT     NOT NULL,
			choices        TEXT,
			correct_answer TEXT     NOT NULL,
			diff_context   TEXT,
			sort_order     INTEGER  NOT NULL DEFAULT 0,
			created_at     TEXT     NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS answers (
			id             INTEGER  PRIMARY KEY AUTOINCREMENT,
			question_id    INTEGER  NOT NULL UNIQUE REFERENCES questions(id),
			user_answer    TEXT     NOT NULL,
			is_correct     INTEGER,
			ai_score       REAL,
			ai_score_label TEXT,
			ai_explanation TEXT,
			grade_status   TEXT     NOT NULL DEFAULT 'graded'
			                        CHECK(grade_status IN ('graded','local_only','failed_retryable')),
			saved          INTEGER  NOT NULL DEFAULT 0,
			export_path    TEXT,
			created_at     TEXT     NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS reviews (
			id            INTEGER  PRIMARY KEY AUTOINCREMENT,
			answer_id     INTEGER  NOT NULL UNIQUE REFERENCES answers(id),
			ease_factor   REAL     NOT NULL DEFAULT 2.5 CHECK(ease_factor >= 1.3),
			interval_days INTEGER  NOT NULL DEFAULT 1   CHECK(interval_days >= 1),
			repetitions   INTEGER  NOT NULL DEFAULT 0   CHECK(repetitions >= 0),
			next_review_at TEXT    NOT NULL,
			updated_at    TEXT     NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS review_logs (
			id                 INTEGER  PRIMARY KEY AUTOINCREMENT,
			answer_id          INTEGER  NOT NULL REFERENCES answers(id),
			self_rating        TEXT     NOT NULL CHECK(self_rating IN ('again','hard','good','easy')),
			ease_factor_before REAL     NOT NULL,
			ease_factor_after  REAL     NOT NULL,
			interval_before    INTEGER  NOT NULL,
			interval_after     INTEGER  NOT NULL,
			created_at         TEXT     NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
	} {
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("create table: %w", err)
		}
	}
	return nil
}

func createIndexes(tx *sql.Tx) error {
	for _, stmt := range []string{
		`CREATE INDEX IF NOT EXISTS idx_commits_repo_reviewed   ON commits(repository_id, reviewed)`,
		`CREATE INDEX IF NOT EXISTS idx_reviews_next_review     ON reviews(next_review_at)`,
		`CREATE INDEX IF NOT EXISTS idx_review_logs_answer      ON review_logs(answer_id, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_answers_saved           ON answers(saved)`,
		`CREATE INDEX IF NOT EXISTS idx_questions_session_order ON questions(session_id, sort_order)`,
	} {
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("create index: %w", err)
		}
	}
	return nil
}

// parseTime は SQLite の TEXT 日時を time.Time に変換する。
func parseTime(b []byte) (time.Time, error) {
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, string(b)); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse time: %q", string(b))
}
