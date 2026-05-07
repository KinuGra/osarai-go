package db

import (
	"context"
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
	db, cleanup := openTestDB(t)
	defer cleanup()

	ctx := context.Background()
	store := NewSessionStore(db)

	sess := &Session{StartedAt: time.Now()}
	if err := store.Create(ctx, sess); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if sess.ID == 0 {
		t.Error("expected non-zero ID after Create")
	}
}

func TestQuestionSave(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	ctx := context.Background()

	sess := &Session{StartedAt: time.Now()}
	if err := NewSessionStore(db).Create(ctx, sess); err != nil {
		t.Fatalf("session Create: %v", err)
	}

	store := NewQuestionStore(db)
	q := &Question{
		SessionID:     sess.ID,
		Title:         "errors.Is vs errors.As",
		Content:       "この関数で errors.Is を使った理由は？",
		QuestionType:  "choice",
		Choices:       []string{"A", "B", "C", "D"},
		CorrectAnswer: "A",
		Category:      "language_knowledge",
	}
	if err := store.Save(ctx, q); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if q.ID == 0 {
		t.Error("expected non-zero ID after Save")
	}

	got, err := store.FindByID(ctx, q.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.Title != q.Title {
		t.Errorf("Title = %q, want %q", got.Title, q.Title)
	}
	if len(got.Choices) != 4 {
		t.Errorf("Choices len = %d, want 4", len(got.Choices))
	}
}
