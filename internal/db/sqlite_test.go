package db

import (
	"database/sql"
	"fmt"
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

// repoCounter は同一テスト内で複数の repository を作るときに一意な name/path を生成するためのカウンター。
var repoCounter int64

func insertRepo(t *testing.T, db *sql.DB) int64 {
	t.Helper()
	repoCounter++
	res, err := db.Exec(
		`INSERT INTO repositories (name, path) VALUES (?, ?)`,
		fmt.Sprintf("testrepo%d", repoCounter),
		fmt.Sprintf("/tmp/testrepo%d", repoCounter),
	)
	if err != nil {
		t.Fatalf("insert repo: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
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

	repoID := insertRepo(t, db)
	store := NewSessionStore(db)

	sess := &Session{
		Mode:         "check",
		RepositoryID: &repoID,
		StartedAt:    time.Now(),
	}
	if err := store.Create(sess); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if sess.ID == 0 {
		t.Error("expected non-zero ID after Create")
	}

	if err := store.Finish(sess.ID, 5, 4, 3); err != nil {
		t.Fatalf("Finish: %v", err)
	}
}

func TestQuestionSave(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	repoID := insertRepo(t, db)
	sess := &Session{Mode: "check", RepositoryID: &repoID, StartedAt: time.Now()}
	if err := NewSessionStore(db).Create(sess); err != nil {
		t.Fatalf("session Create: %v", err)
	}

	store := NewQuestionStore(db)
	q := &Question{
		SessionID:     sess.ID,
		Title:         "errors.Is vs errors.As",
		Category:      "language",
		QuestionType:  "choice",
		Body:          "この関数で errors.Is を使った理由は？",
		Choices:       `["A. ラップされたエラーも検出できるから","B. パフォーマンスが良いから","C. Go の慣習だから","D. 型アサーションができないから"]`,
		CorrectAnswer: "A",
		SortOrder:     0,
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

func TestAnswerSave(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	repoID := insertRepo(t, db)
	sess := &Session{Mode: "check", RepositoryID: &repoID, StartedAt: time.Now()}
	NewSessionStore(db).Create(sess) //nolint:errcheck

	q := &Question{
		SessionID: sess.ID, Title: "test", Category: "language",
		QuestionType: "choice", Body: "q?", Choices: `["A","B"]`,
		CorrectAnswer: "A",
	}
	NewQuestionStore(db).Save(q) //nolint:errcheck

	store := NewAnswerStore(db)

	// 採点成功ケース
	score := 9.0
	label := "9/10"
	expl := "errors.Is はラップされたエラーも検出できます"
	correct := true
	answer := &Answer{
		QuestionID:    q.ID,
		UserAnswer:    "A",
		IsCorrect:     &correct,
		AIScore:       &score,
		AIScoreLabel:  &label,
		AIExplanation: &expl,
		GradeStatus:   "graded",
	}
	if err := store.Save(answer); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if answer.ID == 0 {
		t.Error("expected non-zero ID after Save")
	}

	got, err := store.FindByQuestionID(q.ID)
	if err != nil {
		t.Fatalf("FindByQuestionID: %v", err)
	}
	if got.IsCorrect == nil || !*got.IsCorrect {
		t.Error("IsCorrect should be true")
	}
	if got.AIScore == nil || *got.AIScore != 9.0 {
		t.Errorf("AIScore = %v, want 9.0", got.AIScore)
	}

	// 採点失敗ケース: IsCorrect は nil
	q2 := &Question{
		SessionID: sess.ID, Title: "test2", Category: "language",
		QuestionType: "written", Body: "why?", CorrectAnswer: "...",
	}
	NewQuestionStore(db).Save(q2) //nolint:errcheck
	failAnswer := &Answer{
		QuestionID:  q2.ID,
		UserAnswer:  "some answer",
		IsCorrect:   nil, // 採点失敗
		GradeStatus: "failed_retryable",
	}
	if err := store.Save(failAnswer); err != nil {
		t.Fatalf("Save failed answer: %v", err)
	}
	gotFail, err := store.FindByQuestionID(q2.ID)
	if err != nil {
		t.Fatalf("FindByQuestionID: %v", err)
	}
	if gotFail.IsCorrect != nil {
		t.Error("IsCorrect should be nil for failed grading")
	}
}

func TestReviewCreateAndFindDue(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	repoID := insertRepo(t, db)
	sess := &Session{Mode: "check", RepositoryID: &repoID, StartedAt: time.Now()}
	NewSessionStore(db).Create(sess) //nolint:errcheck

	q := &Question{
		SessionID: sess.ID, Title: "t", Category: "language",
		QuestionType: "choice", Body: "b", Choices: `["A"]`, CorrectAnswer: "A",
	}
	NewQuestionStore(db).Save(q) //nolint:errcheck

	correct := true
	a := &Answer{QuestionID: q.ID, UserAnswer: "A", IsCorrect: &correct, GradeStatus: "graded"}
	NewAnswerStore(db).Save(a) //nolint:errcheck

	store := NewReviewStore(db)
	yesterday := time.Now().AddDate(0, 0, -1)
	review := &Review{
		AnswerID:     a.ID,
		EaseFactor:   2.5,
		IntervalDays: 1,
		Repetitions:  0,
		NextReviewAt: yesterday,
	}
	if err := store.Create(review); err != nil {
		t.Fatalf("Create: %v", err)
	}

	due, err := store.FindDue(time.Now())
	if err != nil {
		t.Fatalf("FindDue: %v", err)
	}
	if len(due) != 1 {
		t.Fatalf("want 1 due review, got %d", len(due))
	}
	if due[0].AnswerID != a.ID {
		t.Errorf("AnswerID = %d, want %d", due[0].AnswerID, a.ID)
	}
}

func TestReviewLogCreate(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	repoID := insertRepo(t, db)
	sess := &Session{Mode: "check", RepositoryID: &repoID, StartedAt: time.Now()}
	NewSessionStore(db).Create(sess) //nolint:errcheck

	q := &Question{
		SessionID: sess.ID, Title: "t", Category: "language",
		QuestionType: "choice", Body: "b", Choices: `["A"]`, CorrectAnswer: "A",
	}
	NewQuestionStore(db).Save(q) //nolint:errcheck

	correct := true
	a := &Answer{QuestionID: q.ID, UserAnswer: "A", IsCorrect: &correct, GradeStatus: "graded"}
	NewAnswerStore(db).Save(a) //nolint:errcheck

	store := NewReviewLogStore(db)
	log := &ReviewLog{
		AnswerID:         a.ID,
		SelfRating:       "good",
		EaseFactorBefore: 2.5,
		EaseFactorAfter:  2.6,
		IntervalBefore:   1,
		IntervalAfter:    3,
	}
	if err := store.Create(log); err != nil {
		t.Fatalf("Create: %v", err)
	}

	logs, err := store.FindByAnswerID(a.ID)
	if err != nil {
		t.Fatalf("FindByAnswerID: %v", err)
	}
	if len(logs) != 1 || logs[0].SelfRating != "good" {
		t.Errorf("unexpected logs: %+v", logs)
	}
}
