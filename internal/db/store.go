package db

import "time"

// ========== DB モデル型 ==========

type Repository struct {
	ID        int64
	Name      string
	Path      string
	RemoteURL *string
	CreatedAt time.Time
}

type Commit struct {
	ID           int64
	RepositoryID int64
	Hash         string // フル SHA (40文字)
	Message      string
	AuthorName   string
	AuthorEmail  string
	DiffSummary  *string // トークン節約用キャッシュ
	Reviewed     bool
	CommittedAt  time.Time
	CreatedAt    time.Time
}

type Session struct {
	ID             int64
	Mode           string  // "check" or "recall"
	RepositoryID   *int64  // nil = recall で全リポ対象
	SourceRef      *string // check 時: staged/file等
	TotalQuestions int
	CorrectCount   int
	MaxStreak      int
	StartedAt      time.Time
	FinishedAt     *time.Time
}

type Question struct {
	ID            int64
	SessionID     int64
	CommitID      *int64 // nil = 未コミット diff から生成
	Title         string
	Category      string // "design" | "language" | "framework"
	QuestionType  string // "choice" | "written"
	Body          string
	Choices       string // JSON 配列
	CorrectAnswer string
	Explanation   *string // AI が生成した問題解説
	DiffContext   *string // 問題生成に使った diff 断片
	SortOrder     int
	CreatedAt     time.Time
}

type Answer struct {
	ID            int64
	QuestionID    int64
	UserAnswer    string
	IsCorrect     *bool    // nil = 記述式採点 API エラー
	AIScore       *float64 // 0.0〜10.0, nil = API エラー
	AIScoreLabel  *string  // "9/10" 等, nil = API エラー
	AIExplanation *string  // 参考書風解説 (md 形式), nil = API エラー
	GradeStatus   string   // "graded" | "local_only" | "failed_retryable"
	Saved         bool
	ExportPath    *string
	CreatedAt     time.Time
}

type Review struct {
	ID           int64
	AnswerID     int64
	EaseFactor   float64
	IntervalDays int
	Repetitions  int
	NextReviewAt time.Time
	UpdatedAt    time.Time
}

type ReviewLog struct {
	ID               int64
	AnswerID         int64
	SelfRating       string // "again" | "hard" | "good" | "easy"
	EaseFactorBefore float64
	EaseFactorAfter  float64
	IntervalBefore   int
	IntervalAfter    int
	CreatedAt        time.Time
}

// ========== Store インターフェース ==========

type RepositoryStore interface {
	FindByPath(path string) (*Repository, error)
	Create(repo *Repository) error
}

type CommitStore interface {
	Create(commit *Commit) error
	FindByHash(repoID int64, hash string) (*Commit, error)
	FindUnreviewed(repoID int64) ([]Commit, error)
	MarkReviewed(id int64) error
}

type SessionStore interface {
	Create(session *Session) error
	Finish(id int64, totalQuestions, correctCount, maxStreak int) error
}

type QuestionStore interface {
	Save(question *Question) error
	FindByID(id int64) (*Question, error)
	FindBySessionID(sessionID int64) ([]Question, error)
	FindByCommitID(commitID int64) ([]Question, error)
}

type AnswerStore interface {
	Save(answer *Answer) error
	FindByID(id int64) (*Answer, error)
	FindByQuestionID(questionID int64) (*Answer, error)
	FindRetryable() ([]Answer, error)
	UpdateGradeStatus(id int64, status string, explanation *string) error
	MarkSaved(id int64, exportPath string) error
	FindSaved() ([]Answer, error)
}

type ReviewStore interface {
	Create(review *Review) error
	FindByAnswerID(answerID int64) (*Review, error)
	FindDue(now time.Time) ([]Review, error)
	Update(review *Review) error
}

type ReviewLogStore interface {
	Create(log *ReviewLog) error
	FindByAnswerID(answerID int64) ([]ReviewLog, error)
}

// Store は全 Store をまとめた DI コンテナ用インターフェース。
type Store interface {
	Repositories() RepositoryStore
	Commits() CommitStore
	Sessions() SessionStore
	Questions() QuestionStore
	Answers() AnswerStore
	Reviews() ReviewStore
	ReviewLogs() ReviewLogStore
	Close() error
}
