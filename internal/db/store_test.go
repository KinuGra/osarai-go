package db

import (
	"database/sql"
	"testing"
	"time"
)

// ==================== fixture helpers ====================

// makeAnswer は repository → session → question → answer まで一気に作って
// 末端の answer ID を返すヘルパー。
func makeAnswer(t *testing.T, db *sql.DB) (int64, *Session, *Question, *Answer) {
	t.Helper()
	repoID := insertRepo(t, db)
	sess := &Session{Mode: "check", RepositoryID: &repoID, StartedAt: time.Now()}
	if err := NewSessionStore(db).Create(sess); err != nil {
		t.Fatalf("session create: %v", err)
	}
	q := &Question{
		SessionID: sess.ID, Title: "t", Category: "language",
		QuestionType: "choice", Body: "b", Choices: `["A","B"]`, CorrectAnswer: "A",
	}
	if err := NewQuestionStore(db).Save(q); err != nil {
		t.Fatalf("question save: %v", err)
	}
	correct := true
	a := &Answer{
		QuestionID:  q.ID,
		UserAnswer:  "A",
		IsCorrect:   &correct,
		GradeStatus: "graded",
	}
	if err := NewAnswerStore(db).Save(a); err != nil {
		t.Fatalf("answer save: %v", err)
	}
	return a.ID, sess, q, a
}

// ==================== SessionStore ====================

func TestSessionFinishStoresStats(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	repoID := insertRepo(t, db)
	store := NewSessionStore(db)
	sess := &Session{Mode: "check", RepositoryID: &repoID, StartedAt: time.Now()}
	if err := store.Create(sess); err != nil {
		t.Fatal(err)
	}

	if err := store.Finish(sess.ID, 5, 4, 3); err != nil {
		t.Fatalf("Finish: %v", err)
	}

	var total, correct, streak int
	var finishedAt sql.NullString
	err := db.QueryRow(
		`SELECT total_questions, correct_count, max_streak, finished_at FROM sessions WHERE id = ?`,
		sess.ID,
	).Scan(&total, &correct, &streak, &finishedAt)
	if err != nil {
		t.Fatal(err)
	}

	if total != 5 || correct != 4 || streak != 3 {
		t.Errorf("got total=%d correct=%d streak=%d, want 5/4/3", total, correct, streak)
	}
	if !finishedAt.Valid {
		t.Error("finished_at should be set after Finish")
	}
	if _, err := time.Parse(time.RFC3339, finishedAt.String); err != nil {
		t.Errorf("finished_at not ISO-8601: %q", finishedAt.String)
	}
}

func TestSessionRecallModeNullRepoID(t *testing.T) {
	// recall は全リポ対象で repository_id = NULL のセッションを作れる
	db, cleanup := openTestDB(t)
	defer cleanup()

	store := NewSessionStore(db)
	sess := &Session{Mode: "recall", RepositoryID: nil, StartedAt: time.Now()}
	if err := store.Create(sess); err != nil {
		t.Fatalf("recall mode with NULL repo_id should succeed: %v", err)
	}

	var repoID sql.NullInt64
	if err := db.QueryRow(`SELECT repository_id FROM sessions WHERE id = ?`, sess.ID).Scan(&repoID); err != nil {
		t.Fatal(err)
	}
	if repoID.Valid {
		t.Errorf("repository_id should be NULL, got %d", repoID.Int64)
	}
}

func TestSessionInvalidModeRejected(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	repoID := insertRepo(t, db)
	store := NewSessionStore(db)

	for _, mode := range []string{"invalid", "", "Check", "RECALL"} {
		t.Run("mode="+mode, func(t *testing.T) {
			sess := &Session{Mode: mode, RepositoryID: &repoID, StartedAt: time.Now()}
			if err := store.Create(sess); err == nil {
				t.Errorf("mode %q should be rejected by CHECK constraint", mode)
			}
		})
	}
}

// ==================== QuestionStore ====================

func TestQuestionCheckModeNullCommitID(t *testing.T) {
	// check で未コミット差分から生成された問題は commit_id = NULL
	db, cleanup := openTestDB(t)
	defer cleanup()

	repoID := insertRepo(t, db)
	sess := &Session{Mode: "check", RepositoryID: &repoID, StartedAt: time.Now()}
	NewSessionStore(db).Create(sess) //nolint:errcheck

	store := NewQuestionStore(db)
	q := &Question{
		SessionID:     sess.ID,
		CommitID:      nil, // 未コミット差分
		Title:         "未コミット",
		Category:      "design",
		QuestionType:  "written",
		Body:          "なぜそう書いた？",
		CorrectAnswer: "...",
	}
	if err := store.Save(q); err != nil {
		t.Fatalf("save with NULL commit_id should succeed: %v", err)
	}

	got, err := store.FindBySessionID(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1, got %d", len(got))
	}
	if got[0].CommitID != nil {
		t.Errorf("CommitID should be nil, got %v", got[0].CommitID)
	}
}

func TestQuestionSortOrderRespected(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	repoID := insertRepo(t, db)
	sess := &Session{Mode: "check", RepositoryID: &repoID, StartedAt: time.Now()}
	NewSessionStore(db).Create(sess) //nolint:errcheck

	store := NewQuestionStore(db)
	// 逆順に挿入
	for _, order := range []int{2, 0, 1} {
		q := &Question{
			SessionID: sess.ID, Title: "q", Category: "language",
			QuestionType: "choice", Body: "b", Choices: `["A"]`, CorrectAnswer: "A",
			SortOrder: order,
		}
		if err := store.Save(q); err != nil {
			t.Fatal(err)
		}
	}

	got, _ := store.FindBySessionID(sess.ID)
	if len(got) != 3 {
		t.Fatalf("want 3, got %d", len(got))
	}
	for i, q := range got {
		if q.SortOrder != i {
			t.Errorf("position %d: SortOrder = %d, want %d", i, q.SortOrder, i)
		}
	}
}

func TestQuestionAllCategoriesAccepted(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	repoID := insertRepo(t, db)
	sess := &Session{Mode: "check", RepositoryID: &repoID, StartedAt: time.Now()}
	NewSessionStore(db).Create(sess) //nolint:errcheck

	store := NewQuestionStore(db)
	for _, cat := range []string{"design", "language", "framework"} {
		t.Run("category="+cat, func(t *testing.T) {
			q := &Question{
				SessionID: sess.ID, Title: "t", Category: cat,
				QuestionType: "choice", Body: "b", Choices: `["A"]`, CorrectAnswer: "A",
			}
			if err := store.Save(q); err != nil {
				t.Errorf("category %q rejected: %v", cat, err)
			}
		})
	}

	// カテゴリのバリデーションはアプリケーション層の責務
	// DB 制約は外しているため、未知のカテゴリも DB レベルでは受け入れる
	t.Run("unknown category accepted at DB level", func(t *testing.T) {
		q := &Question{
			SessionID: sess.ID, Title: "t", Category: "performance",
			QuestionType: "choice", Body: "b", CorrectAnswer: "A",
		}
		if err := store.Save(q); err != nil {
			t.Errorf("DB should accept unknown category (validation is app-layer): %v", err)
		}
	})
}

// ==================== AnswerStore ====================

func TestAnswerNullableFieldsAllNull(t *testing.T) {
	// 採点失敗（failed_retryable）で全ての AI 関連フィールドが NULL のケース
	db, cleanup := openTestDB(t)
	defer cleanup()

	repoID := insertRepo(t, db)
	sess := &Session{Mode: "check", RepositoryID: &repoID, StartedAt: time.Now()}
	NewSessionStore(db).Create(sess) //nolint:errcheck

	q := &Question{
		SessionID: sess.ID, Title: "t", Category: "language",
		QuestionType: "written", Body: "b", CorrectAnswer: "x",
	}
	NewQuestionStore(db).Save(q) //nolint:errcheck

	store := NewAnswerStore(db)
	a := &Answer{
		QuestionID:    q.ID,
		UserAnswer:    "user input",
		IsCorrect:     nil,
		AIScore:       nil,
		AIScoreLabel:  nil,
		AIExplanation: nil,
		GradeStatus:   "failed_retryable",
		ExportPath:    nil,
	}
	if err := store.Save(a); err != nil {
		t.Fatalf("save with all-null AI fields: %v", err)
	}

	got, err := store.FindByQuestionID(q.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.IsCorrect != nil || got.AIScore != nil || got.AIScoreLabel != nil ||
		got.AIExplanation != nil || got.ExportPath != nil {
		t.Errorf("expected all nullables to be nil, got %+v", got)
	}
}

func TestAnswerAllGradeStatusAccepted(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	for _, status := range []string{"graded", "local_only", "failed_retryable"} {
		t.Run("status="+status, func(t *testing.T) {
			repoID := insertRepo(t, db)
			// 同じ DB を使い回すので一意なリポ名が必要 → 別 DB にしてもいいがここでは
			// 親のループ内で都度作る代わりにループ外で session を作っておく
			sess := &Session{Mode: "check", RepositoryID: &repoID, StartedAt: time.Now()}
			NewSessionStore(db).Create(sess) //nolint:errcheck

			q := &Question{
				SessionID: sess.ID, Title: "t", Category: "language",
				QuestionType: "choice", Body: "b", Choices: `["A"]`, CorrectAnswer: "A",
			}
			NewQuestionStore(db).Save(q) //nolint:errcheck

			a := &Answer{QuestionID: q.ID, UserAnswer: "A", GradeStatus: status}
			if err := NewAnswerStore(db).Save(a); err != nil {
				t.Errorf("status %q rejected: %v", status, err)
			}
		})
	}
}

func TestAnswerInvalidGradeStatusRejected(t *testing.T) {
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

	a := &Answer{QuestionID: q.ID, UserAnswer: "A", GradeStatus: "skipped"}
	if err := NewAnswerStore(db).Save(a); err == nil {
		t.Error("'skipped' should be rejected — spec only allows graded/local_only/failed_retryable")
	}
}

func TestAnswerMarkSavedAndFindSaved(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	id, _, _, _ := makeAnswer(t, db)
	store := NewAnswerStore(db)

	saved, err := store.FindSaved()
	if err != nil {
		t.Fatal(err)
	}
	if len(saved) != 0 {
		t.Errorf("initial saved count = %d, want 0", len(saved))
	}

	if err := store.MarkSaved(id); err != nil {
		t.Fatalf("MarkSaved: %v", err)
	}

	saved, err = store.FindSaved()
	if err != nil {
		t.Fatal(err)
	}
	if len(saved) != 1 || saved[0].ID != id || !saved[0].Saved {
		t.Errorf("FindSaved returned %+v, want [{ID:%d Saved:true}]", saved, id)
	}
}

func TestAnswerUpdateGradeStatus(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	id, _, _, _ := makeAnswer(t, db)
	store := NewAnswerStore(db)

	if err := store.UpdateGradeStatus(id, "failed_retryable"); err != nil {
		t.Fatalf("UpdateGradeStatus: %v", err)
	}

	var status string
	db.QueryRow(`SELECT grade_status FROM answers WHERE id = ?`, id).Scan(&status) //nolint:errcheck
	if status != "failed_retryable" {
		t.Errorf("status = %q, want failed_retryable", status)
	}
}

func TestAnswerFindRetryable(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	store := NewAnswerStore(db)

	// graded のみ → 0 件
	gradedID, _, _, _ := makeAnswer(t, db)
	got, _ := store.FindRetryable()
	if len(got) != 0 {
		t.Errorf("want 0 retryable, got %d", len(got))
	}

	// gradedID を failed_retryable に変更 → 1 件
	store.UpdateGradeStatus(gradedID, "failed_retryable") //nolint:errcheck
	got, _ = store.FindRetryable()
	if len(got) != 1 || got[0].ID != gradedID {
		t.Errorf("want [%d], got %+v", gradedID, got)
	}

	// local_only は対象外（再採点不要）
	id2, _, _, _ := makeAnswer(t, db)
	store.UpdateGradeStatus(id2, "local_only") //nolint:errcheck
	got, _ = store.FindRetryable()
	if len(got) != 1 {
		t.Errorf("local_only should not be retryable, got %d", len(got))
	}
}

func TestAnswerUniqueConstraint(t *testing.T) {
	// answers.question_id は UNIQUE — 1 question に 1 answer のみ
	db, cleanup := openTestDB(t)
	defer cleanup()

	_, _, q, _ := makeAnswer(t, db)
	store := NewAnswerStore(db)

	a := &Answer{QuestionID: q.ID, UserAnswer: "B", GradeStatus: "graded"}
	if err := store.Save(a); err == nil {
		t.Error("second answer for same question should be rejected (UNIQUE)")
	}
}

// ==================== ReviewStore ====================

func TestReviewUpdate(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	answerID, _, _, _ := makeAnswer(t, db)
	store := NewReviewStore(db)

	r := &Review{
		AnswerID:     answerID,
		EaseFactor:   2.5,
		IntervalDays: 1,
		Repetitions:  0,
		NextReviewAt: time.Now().AddDate(0, 0, 1),
	}
	if err := store.Create(r); err != nil {
		t.Fatal(err)
	}

	// SM-2 で更新（Good 評価相当）
	r.EaseFactor = 2.6
	r.IntervalDays = 3
	r.Repetitions = 1
	r.NextReviewAt = time.Now().AddDate(0, 0, 3)
	if err := store.Update(r); err != nil {
		t.Fatalf("Update: %v", err)
	}

	var ef float64
	var interval, reps int
	db.QueryRow(
		`SELECT ease_factor, interval_days, repetitions FROM reviews WHERE id = ?`, r.ID,
	).Scan(&ef, &interval, &reps) //nolint:errcheck

	if ef != 2.6 || interval != 3 || reps != 1 {
		t.Errorf("got ef=%v interval=%d reps=%d, want 2.6/3/1", ef, interval, reps)
	}
}

func TestReviewFindDueBoundary(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	store := NewReviewStore(db)
	now := time.Now()

	// 3つの review を昨日・今日・明日で作る
	cases := []struct {
		name      string
		next      time.Time
		shouldDue bool
	}{
		{"yesterday", now.AddDate(0, 0, -1), true},
		{"today", now, true},
		{"tomorrow", now.AddDate(0, 0, 1), false},
	}
	ids := map[string]int64{}
	for _, c := range cases {
		answerID, _, _, _ := makeAnswer(t, db)
		r := &Review{
			AnswerID:     answerID,
			EaseFactor:   2.5,
			IntervalDays: 1,
			Repetitions:  0,
			NextReviewAt: c.next,
		}
		if err := store.Create(r); err != nil {
			t.Fatal(err)
		}
		ids[c.name] = r.ID
	}

	due, err := store.FindDue(now)
	if err != nil {
		t.Fatal(err)
	}

	dueIDs := map[int64]bool{}
	for _, r := range due {
		dueIDs[r.ID] = true
	}
	for _, c := range cases {
		got := dueIDs[ids[c.name]]
		if got != c.shouldDue {
			t.Errorf("%s: due=%v, want %v", c.name, got, c.shouldDue)
		}
	}
}

func TestReviewBoundaryValues(t *testing.T) {
	// CHECK 制約の境界値: ease_factor=1.3, interval_days=1, repetitions=0 は OK
	db, cleanup := openTestDB(t)
	defer cleanup()

	answerID, _, _, _ := makeAnswer(t, db)
	r := &Review{
		AnswerID:     answerID,
		EaseFactor:   1.3, // 境界値
		IntervalDays: 1,   // 境界値
		Repetitions:  0,   // 境界値
		NextReviewAt: time.Now(),
	}
	if err := NewReviewStore(db).Create(r); err != nil {
		t.Errorf("boundary values should be accepted: %v", err)
	}
}

func TestReviewUniqueAnswerID(t *testing.T) {
	// reviews.answer_id は UNIQUE — 1 answer に 1 review のみ
	db, cleanup := openTestDB(t)
	defer cleanup()

	answerID, _, _, _ := makeAnswer(t, db)
	store := NewReviewStore(db)

	r := &Review{AnswerID: answerID, EaseFactor: 2.5, IntervalDays: 1, NextReviewAt: time.Now()}
	if err := store.Create(r); err != nil {
		t.Fatal(err)
	}

	r2 := &Review{AnswerID: answerID, EaseFactor: 2.5, IntervalDays: 1, NextReviewAt: time.Now()}
	if err := store.Create(r2); err == nil {
		t.Error("second review for same answer should be rejected (UNIQUE)")
	}
}

// ==================== ReviewLogStore ====================

func TestReviewLogAllRatings(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	store := NewReviewLogStore(db)

	for _, rating := range []string{"again", "hard", "good", "easy"} {
		t.Run("rating="+rating, func(t *testing.T) {
			answerID, _, _, _ := makeAnswer(t, db)
			log := &ReviewLog{
				AnswerID: answerID, SelfRating: rating,
				EaseFactorBefore: 2.5, EaseFactorAfter: 2.5,
				IntervalBefore: 1, IntervalAfter: 1,
			}
			if err := store.Create(log); err != nil {
				t.Errorf("rating %q rejected: %v", rating, err)
			}
		})
	}

	t.Run("invalid rejected", func(t *testing.T) {
		answerID, _, _, _ := makeAnswer(t, db)
		log := &ReviewLog{
			AnswerID: answerID, SelfRating: "perfect",
			EaseFactorBefore: 2.5, EaseFactorAfter: 2.5,
			IntervalBefore: 1, IntervalAfter: 1,
		}
		if err := store.Create(log); err == nil {
			t.Error("'perfect' should be rejected — spec allows again/hard/good/easy only")
		}
	})
}

func TestReviewLogOrdering(t *testing.T) {
	// FindByAnswerID は created_at 順で返る
	db, cleanup := openTestDB(t)
	defer cleanup()

	answerID, _, _, _ := makeAnswer(t, db)
	store := NewReviewLogStore(db)

	for _, rating := range []string{"good", "hard", "easy"} {
		log := &ReviewLog{
			AnswerID: answerID, SelfRating: rating,
			EaseFactorBefore: 2.5, EaseFactorAfter: 2.5,
			IntervalBefore: 1, IntervalAfter: 1,
		}
		if err := store.Create(log); err != nil {
			t.Fatal(err)
		}
		// SQLite の CURRENT_TIMESTAMP は秒精度なので、順序を保証するために少し待つ
		time.Sleep(1100 * time.Millisecond)
	}

	logs, err := store.FindByAnswerID(answerID)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 3 {
		t.Fatalf("want 3 logs, got %d", len(logs))
	}
	want := []string{"good", "hard", "easy"}
	for i, log := range logs {
		if log.SelfRating != want[i] {
			t.Errorf("position %d: rating = %q, want %q", i, log.SelfRating, want[i])
		}
	}
}

func TestReviewLogMultiplePerAnswer(t *testing.T) {
	// review_logs は 1:N（同じ answer に複数 log 可能）
	db, cleanup := openTestDB(t)
	defer cleanup()

	answerID, _, _, _ := makeAnswer(t, db)
	store := NewReviewLogStore(db)

	for i := 0; i < 5; i++ {
		log := &ReviewLog{
			AnswerID: answerID, SelfRating: "good",
			EaseFactorBefore: 2.5, EaseFactorAfter: 2.5,
			IntervalBefore: i, IntervalAfter: i + 1,
		}
		if err := store.Create(log); err != nil {
			t.Fatalf("log %d: %v", i, err)
		}
	}

	logs, _ := store.FindByAnswerID(answerID)
	if len(logs) != 5 {
		t.Errorf("want 5 logs, got %d", len(logs))
	}
}

// ==================== Repository / Commit (NULLABLE フィールド) ====================

func TestRepositoryWithNullRemoteURL(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	_, err := db.Exec(`INSERT INTO repositories (name, path) VALUES (?, ?)`, "norm", "/p")
	if err != nil {
		t.Fatalf("repo with NULL remote_url should be allowed: %v", err)
	}

	var remoteURL sql.NullString
	db.QueryRow(`SELECT remote_url FROM repositories WHERE name = 'norm'`).Scan(&remoteURL) //nolint:errcheck
	if remoteURL.Valid {
		t.Errorf("remote_url should be NULL")
	}
}

func TestCommitWithNullDiffSummary(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	repoID := insertRepo(t, db)
	_, err := db.Exec(
		`INSERT INTO commits (repository_id, hash, message, author_name, author_email, committed_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		repoID, "1234567890123456789012345678901234567890",
		"msg", "name", "e@example.com", "2024-01-01T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("commit with NULL diff_summary: %v", err)
	}
}

// ==================== Store DI コンテナ ====================

func TestStoreAggregator(t *testing.T) {
	store, err := openStore(t.TempDir() + "/data.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	// 全サブストアが nil でないこと
	if store.Repositories() == nil ||
		store.Commits() == nil ||
		store.Sessions() == nil ||
		store.Questions() == nil ||
		store.Answers() == nil ||
		store.Reviews() == nil ||
		store.ReviewLogs() == nil {
		t.Error("Store sub-stores should all be non-nil")
	}

	// 同じインスタンスが返ること（毎回作り直してない）
	if store.Sessions() != store.Sessions() {
		t.Error("Sessions() should return the same instance")
	}
}

func TestStoreClose(t *testing.T) {
	store, err := openStore(t.TempDir() + "/data.db")
	if err != nil {
		t.Fatal(err)
	}

	if err := store.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}
