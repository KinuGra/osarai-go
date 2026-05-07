package db

import "time"

// ========== DB モデル型（テーブルの各行に対応する構造体）==========

// Repository はリポジトリ情報（repositories テーブル）。
type Repository struct {
	ID        int64
	Path      string // リポジトリの絶対パス
	Name      string // リポジトリ名（ディレクトリ名）
	CreatedAt time.Time
}

// Commit はコミット情報（commits テーブル）。
type Commit struct {
	ID           int64
	RepositoryID int64
	Hash         string // コミットハッシュ（SHA）
	Message      string // コミットメッセージ
	DiffBody     string // diff の内容
	Reviewed     bool   // 振り返り済みか
	CreatedAt    time.Time
}

// Session は学習セッション（sessions テーブル）。
// 1回の `osarai check` 実行 = 1セッション。
type Session struct {
	ID           int64
	RepositoryID int64
	CommitHash   string // 対象コミット（空なら全体 diff）
	DiffScope    string // "all", "staged", "commit"
	StartedAt    time.Time
	FinishedAt   *time.Time // nil なら未完了
}

// Question は生成された問題（questions テーブル）。
type Question struct {
	ID           int64
	SessionID    int64
	CommitID     *int64 // nil なら全体 diff から生成
	Title        string
	Body         string
	QuestionType string // "choice" or "written"
	Choices      string // JSON 文字列（選択肢の配列）
	Answer       string // 正解
	Explanation  string // 出題意図
	SavedForMD   bool   // md エクスポート用に保存済みか
	CreatedAt    time.Time
}

// Answer はユーザーの回答（answers テーブル）。
type Answer struct {
	ID          int64
	QuestionID  int64
	UserAnswer  string
	IsCorrect   bool
	Score       int    // 0〜100
	GradeStatus string // "graded", "failed_retryable", "skipped"
	Explanation string // AI が生成した参考書風解説
	CreatedAt   time.Time
}

// Review は復習スケジュール（reviews テーブル）。SM-2 のパラメータを持つ。
type Review struct {
	ID           int64
	QuestionID   int64
	Interval     float64   // 次回までの間隔（日）
	Repetitions  int       // 連続正解回数
	EaseFactor   float64   // 容易さ係数（SM-2: 初期値 2.5）
	NextReviewAt time.Time // 次回復習日
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ReviewLog は復習履歴（review_logs テーブル）。
type ReviewLog struct {
	ID        int64
	ReviewID  int64
	Rating    int // 0=Again, 1=Hard, 2=Good, 3=Easy
	CreatedAt time.Time
}

// ========== Store インターフェース ==========
// 各テーブルの CRUD 操作を定義。実装は Issue #2 以降。

// RepositoryStore はリポジトリの CRUD。
type RepositoryStore interface {
	FindByPath(path string) (*Repository, error)
	Create(repo *Repository) error
}

// CommitStore はコミットの CRUD。
type CommitStore interface {
	Create(commit *Commit) error
	FindUnreviewed(repoID int64) ([]Commit, error)
	MarkReviewed(id int64) error
}

// SessionStore はセッションの CRUD。
type SessionStore interface {
	Create(session *Session) error
	Finish(id int64) error
}

// QuestionStore は問題の CRUD。
type QuestionStore interface {
	Save(question *Question) error
	FindBySessionID(sessionID int64) ([]Question, error)
	FindSaved() ([]Question, error) // md エクスポート用
	MarkSavedForMD(id int64) error
}

// AnswerStore は回答の CRUD。
type AnswerStore interface {
	Save(answer *Answer) error
	FindByQuestionID(questionID int64) (*Answer, error)
	FindRetryable() ([]Answer, error) // grade_status="failed_retryable"
	UpdateGradeStatus(id int64, status string) error
}

// ReviewStore は復習スケジュールの CRUD。
type ReviewStore interface {
	Create(review *Review) error             // B の責務: 初回 INSERT
	FindDue(now time.Time) ([]Review, error) // C の責務: 復習対象取得
	Update(review *Review) error             // C の責務: SM-2 パラメータ更新
}

// ReviewLogStore は復習履歴の CRUD。
type ReviewLogStore interface {
	Create(log *ReviewLog) error
	FindByReviewID(reviewID int64) ([]ReviewLog, error)
}

// Store は全 Store をまとめたインターフェース。
// DI コンテナ（core/service.go）に渡す用。
type Store interface {
	Repositories() RepositoryStore
	Commits() CommitStore
	Sessions() SessionStore
	Questions() QuestionStore
	Answers() AnswerStore
	Reviews() ReviewStore
	ReviewLogs() ReviewLogStore
	Close() error // DB 接続を閉じる
}
