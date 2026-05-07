package db

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"
)

func openTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	db, err := openDB(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	return db, func() { db.Close() }
}

func TestMigrate(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	var version int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != currentSchemaVersion {
		t.Errorf("user_version = %d, want %d", version, currentSchemaVersion)
	}
}

func TestSessionCreate(t *testing.T) {
	sqlDB, cleanup := openTestDB(t)
	defer cleanup()

	// sessions は repository_id NOT NULL なので先にリポジトリを作る
	_, err := sqlDB.Exec(
		`INSERT INTO repositories (path, name) VALUES (?, ?)`,
		"/tmp/testrepo", "testrepo",
	)
	if err != nil {
		t.Fatalf("insert repo: %v", err)
	}
	var repoID int64
	sqlDB.QueryRow(`SELECT last_insert_rowid()`).Scan(&repoID)

	store := NewSessionStore(sqlDB)
	sess := &Session{
		RepositoryID: repoID,
		DiffScope:    "staged",
		StartedAt:    time.Now(),
	}
	if err := store.Create(sess); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if sess.ID == 0 {
		t.Error("expected non-zero ID after Create")
	}

	if err := store.Finish(sess.ID); err != nil {
		t.Fatalf("Finish: %v", err)
	}
}

func TestQuestionSave(t *testing.T) {
	sqlDB, cleanup := openTestDB(t)
	defer cleanup()

	// repository
	sqlDB.Exec(`INSERT INTO repositories (path, name) VALUES (?, ?)`, "/tmp/repo", "repo")
	var repoID int64
	sqlDB.QueryRow(`SELECT last_insert_rowid()`).Scan(&repoID)

	// session
	sessionStore := NewSessionStore(sqlDB)
	sess := &Session{RepositoryID: repoID, StartedAt: time.Now()}
	if err := sessionStore.Create(sess); err != nil {
		t.Fatalf("session Create: %v", err)
	}

	store := NewQuestionStore(sqlDB)
	q := &Question{
		SessionID:    sess.ID,
		Title:        "errors.Is vs errors.As",
		Body:         "この関数で errors.Is を使った理由は？",
		QuestionType: "choice",
		Choices:      `["A","B","C","D"]`,
		Answer:       "A",
		Explanation:  "errors.Is はラップされたエラーも検出できるため",
	}
	if err := store.Save(q); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if q.ID == 0 {
		t.Error("expected non-zero ID after Save")
	}

	got, err := store.FindBySessionID(sess.ID)
	if err != nil {
		t.Fatalf("FindBySessionID: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 question, got %d", len(got))
	}
	if got[0].Title != q.Title {
		t.Errorf("Title = %q, want %q", got[0].Title, q.Title)
	}
}
