package db

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestForeignKeyEnforced は FK 制約が実際に効いていることを検証する。
// PRAGMA foreign_keys = ON が新しいコネクションでも有効か確認するため。
func TestForeignKeyEnforced(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	// 存在しない repository_id への INSERT は失敗するべき
	_, err := db.Exec(
		`INSERT INTO commits
		 (repository_id, hash, message, author_name, author_email, committed_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		99999,
		"0000000000000000000000000000000000000000",
		"test", "test", "test@example.com",
		"2024-01-01T00:00:00Z",
	)
	if err == nil {
		t.Fatal("expected FK constraint violation, got nil — foreign_keys PRAGMA may not be applied per connection")
	}
	if !strings.Contains(err.Error(), "FOREIGN KEY") && !strings.Contains(err.Error(), "constraint") {
		t.Errorf("unexpected error type: %v", err)
	}
}

// TestForeignKeyEnforcedOnFreshConnection は、新しく開かれたコネクション
// (プールに既存のコネクションがない状態) でも FK が有効か検証する。
// applyPragmas は1つの接続にしか効かないため、並行クエリで新しい接続を強制する。
func TestForeignKeyEnforcedOnFreshConnection(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	db.SetMaxOpenConns(5)

	// 並行に長時間トランザクションを保持して、追加コネクションを生成する
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback() //nolint:errcheck
	if _, err := tx.Exec(`SELECT 1`); err != nil {
		t.Fatal(err)
	}

	// この Exec は別のコネクションで実行される
	_, err = db.Exec(
		`INSERT INTO commits
		 (repository_id, hash, message, author_name, author_email, committed_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		99999,
		"0000000000000000000000000000000000000000",
		"test", "test", "test@example.com",
		"2024-01-01T00:00:00Z",
	)
	if err == nil {
		t.Fatal("expected FK violation on fresh connection, got nil — PRAGMA not applied per-connection")
	}
	if !strings.Contains(err.Error(), "FOREIGN KEY") && !strings.Contains(err.Error(), "constraint") {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestCheckConstraints は CHECK 制約が効いていることを検証する。
func TestCheckConstraints(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	repoID := insertRepo(t, db)

	tests := []struct {
		name    string
		query   string
		args    []interface{}
		wantErr bool
	}{
		{
			// SHA-1(40文字) も SHA-256(64文字) も受け入れる（将来の移行を考慮）
			name:    "commit hash SHA-256(64文字)を許可",
			query:   `INSERT INTO commits (repository_id, hash, message, author_name, author_email, committed_at) VALUES (?, ?, ?, ?, ?, ?)`,
			args:    []interface{}{repoID, "0000000000000000000000000000000000000000000000000000000000000000", "m", "n", "e", "2024-01-01T00:00:00Z"},
			wantErr: false,
		},
		{
			name:    "session.mode が check/recall 以外は拒否",
			query:   `INSERT INTO sessions (mode, repository_id) VALUES (?, ?)`,
			args:    []interface{}{"invalid", repoID},
			wantErr: true,
		},
		{
			name:    "session.mode = check は許可",
			query:   `INSERT INTO sessions (mode, repository_id) VALUES (?, ?)`,
			args:    []interface{}{"check", repoID},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := db.Exec(tt.query, tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("err = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

// TestSM2Invariants は SM-2 の不変条件が DB レベルで守られていることを検証する。
func TestSM2Invariants(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	// セットアップ: repository → session → question → answer
	repoID := insertRepo(t, db)
	res, _ := db.Exec(`INSERT INTO sessions (mode, repository_id) VALUES (?, ?)`, "check", repoID)
	sessID, _ := res.LastInsertId()
	res, _ = db.Exec(
		`INSERT INTO questions (session_id, title, category, question_type, body, correct_answer, sort_order)
		 VALUES (?, 't', 'language', 'choice', 'b', 'A', 0)`, sessID,
	)
	qID, _ := res.LastInsertId()
	res, _ = db.Exec(
		`INSERT INTO answers (question_id, user_answer, grade_status) VALUES (?, 'A', 'graded')`, qID,
	)
	answerID, _ := res.LastInsertId()

	tests := []struct {
		name string
		args []interface{} // ease_factor, interval_days, repetitions
	}{
		{"ease_factor < 1.3 は拒否", []interface{}{answerID, 1.2, 1, 0, "2024-01-01"}},
		{"interval_days = 0 は拒否", []interface{}{answerID, 2.5, 0, 0, "2024-01-01"}},
		{"repetitions = -1 は拒否", []interface{}{answerID, 2.5, 1, -1, "2024-01-01"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := db.Exec(
				`INSERT INTO reviews (answer_id, ease_factor, interval_days, repetitions, next_review_at)
				 VALUES (?, ?, ?, ?, ?)`,
				tt.args...,
			)
			if err == nil {
				t.Errorf("expected CHECK constraint violation")
			}
		})
	}
}

// TestUniqueConstraints は UNIQUE 制約が効いていることを検証する。
func TestUniqueConstraints(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	// repositories: name と path が UNIQUE
	if _, err := db.Exec(`INSERT INTO repositories (name, path) VALUES (?, ?)`, "uniqrepo", "/tmp/uniq"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO repositories (name, path) VALUES (?, ?)`, "uniqrepo", "/tmp/other"); err == nil {
		t.Error("expected UNIQUE violation on repositories.name")
	}
	if _, err := db.Exec(`INSERT INTO repositories (name, path) VALUES (?, ?)`, "other", "/tmp/uniq"); err == nil {
		t.Error("expected UNIQUE violation on repositories.path")
	}

	// commits: (repository_id, hash) の複合 UNIQUE
	repoID := insertRepo(t, db)
	hash := "1234567890123456789012345678901234567890"
	if _, err := db.Exec(
		`INSERT INTO commits (repository_id, hash, message, author_name, author_email, committed_at) VALUES (?, ?, ?, ?, ?, ?)`,
		repoID, hash, "m", "n", "e", "2024-01-01T00:00:00Z",
	); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(
		`INSERT INTO commits (repository_id, hash, message, author_name, author_email, committed_at) VALUES (?, ?, ?, ?, ?, ?)`,
		repoID, hash, "m2", "n", "e", "2024-01-01T00:00:00Z",
	); err == nil {
		t.Error("expected UNIQUE violation on (repository_id, hash)")
	}
}

// TestIndexesCreated はインデックスが作られていることを検証する。
func TestIndexesCreated(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type = 'index' AND name LIKE 'idx_%'`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	expected := map[string]bool{
		"idx_commits_repo_reviewed":   false,
		"idx_reviews_next_review":     false,
		"idx_review_logs_answer":      false,
		"idx_answers_saved":           false,
		"idx_questions_session_order": false,
	}
	for rows.Next() {
		var name string
		rows.Scan(&name) //nolint:errcheck
		if _, ok := expected[name]; ok {
			expected[name] = true
		}
	}
	for name, found := range expected {
		if !found {
			t.Errorf("index %s not created", name)
		}
	}
}

// TestTimeStorageFormat は SessionStore.Create 経由で時刻が ISO-8601 UTC で
// 保存され、ラウンドトリップでロスしないことを検証する。
func TestTimeStorageFormat(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	repoID := insertRepo(t, db)

	// JST のローカル時刻を渡しても UTC で保存されるか確認
	jst := time.FixedZone("JST", 9*60*60)
	tm := time.Date(2026, 4, 21, 16, 30, 0, 0, jst)
	expectedUTC := tm.UTC() // 2026-04-21T07:30:00Z

	store := NewSessionStore(db)
	sess := &Session{Mode: "check", RepositoryID: &repoID, StartedAt: tm}
	if err := store.Create(sess); err != nil {
		t.Fatal(err)
	}

	var stored string
	if err := db.QueryRow(`SELECT started_at FROM sessions WHERE id = ?`, sess.ID).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	t.Logf("stored time format: %q", stored)

	expectedString := "2026-04-21T07:30:00Z"
	if stored != expectedString {
		t.Errorf("stored = %q, want %q (must be ISO-8601 UTC)", stored, expectedString)
	}

	parsed, err := parseTime([]byte(stored))
	if err != nil {
		t.Fatalf("cannot parse: %v", err)
	}
	if !parsed.Equal(expectedUTC) {
		t.Errorf("roundtrip: parsed = %v, want %v", parsed, expectedUTC)
	}
}

// TestDefaultTimestampFormat は CURRENT_TIMESTAMP デフォルト値が ISO-8601 UTC で
// 保存されることを検証する（strftime ラッパーが効いているか）。
func TestDefaultTimestampFormat(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	insertRepo(t, db) // 上で created_at がデフォルト値で入る

	var stored string
	if err := db.QueryRow(`SELECT created_at FROM repositories LIMIT 1`).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	t.Logf("default created_at format: %q", stored)

	if _, err := time.Parse(time.RFC3339, stored); err != nil {
		t.Errorf("default created_at not ISO-8601: %q (%v)", stored, err)
	}
}

// TestPragmaApplied は3つの PRAGMA が新規接続でも有効になっていることを検証する。
func TestPragmaApplied(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	tests := []struct {
		name  string
		query string
		want  string
	}{
		{"foreign_keys", "PRAGMA foreign_keys", "1"},
		{"busy_timeout", "PRAGMA busy_timeout", "5000"},
		{"journal_mode", "PRAGMA journal_mode", "wal"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string
			if err := db.QueryRow(tt.query).Scan(&got); err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("%s = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

// TestAllTablesCreated は仕様書記載の7テーブルが全て作られていることを検証する。
func TestAllTablesCreated(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	rows, err := db.Query(
		`SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	got := map[string]bool{}
	for rows.Next() {
		var name string
		rows.Scan(&name) //nolint:errcheck
		got[name] = true
	}

	want := []string{
		"repositories", "commits", "sessions",
		"questions", "answers", "reviews", "review_logs",
	}
	for _, name := range want {
		if !got[name] {
			t.Errorf("table %q not created", name)
		}
	}
}

// TestSchemaVersionPersisted は user_version が正しくセットされ、再 Open でも保持されることを検証する。
func TestSchemaVersionPersisted(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/test.db"

	db1, _ := openDB(path)
	var v1 int
	db1.QueryRow("PRAGMA user_version").Scan(&v1) //nolint:errcheck
	db1.Close()

	if v1 != currentSchemaVersion {
		t.Fatalf("first open: version = %d, want %d", v1, currentSchemaVersion)
	}

	db2, _ := openDB(path)
	defer db2.Close()
	var v2 int
	db2.QueryRow("PRAGMA user_version").Scan(&v2) //nolint:errcheck
	if v2 != currentSchemaVersion {
		t.Errorf("second open: version = %d, want %d", v2, currentSchemaVersion)
	}
}

// TestMigrationSkippedWhenUpToDate は既に最新バージョンなら migration がスキップされることを検証する。
func TestMigrationSkippedWhenUpToDate(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/test.db"

	db, err := openDB(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// 1回目で必ずテーブルが出来ているはず
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'sessions'`).Scan(&count) //nolint:errcheck
	if count != 1 {
		t.Fatal("sessions table should exist after first open")
	}

	// 2回目の migrate は何も変更しないはず（エラーにもならない）
	if err := migrate(db); err != nil {
		t.Errorf("re-migrate should be no-op, got error: %v", err)
	}
}

// TestMigrateIdempotent は同じバージョンで2回 Open しても問題ないことを検証する。
func TestMigrateIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	db1, err := openDB(path)
	if err != nil {
		t.Fatal(err)
	}
	db1.Close()

	db2, err := openDB(path)
	if err != nil {
		t.Fatalf("re-open failed: %v", err)
	}
	defer db2.Close()

	var version int
	if err := db2.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != currentSchemaVersion {
		t.Errorf("version = %d, want %d", version, currentSchemaVersion)
	}
}
