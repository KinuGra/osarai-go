package db

import (
	"context"
	"time"
)

// ---------- Model types ----------

type Repository struct {
	ID        int64
	Name      string
	Path      string
	CreatedAt time.Time
}

type Commit struct {
	ID           int64
	RepositoryID int64
	Hash         string
	Message      string
	Author       string
	CommittedAt  time.Time
	Reviewed     bool
	CreatedAt    time.Time
}

type Session struct {
	ID             int64
	RepositoryID   *int64
	StartedAt      time.Time
	EndedAt        *time.Time
	TotalQuestions int
	CorrectCount   int
	MaxStreak      int
}

type Question struct {
	ID            int64
	SessionID     int64
	CommitID      *int64
	Title         string
	Content       string
	QuestionType  string // "choice" or "written"
	Choices       []string
	CorrectAnswer string
	Category      string
	Saved         bool
	CreatedAt     time.Time
}

type Answer struct {
	ID            int64
	QuestionID    int64
	UserAnswer    string
	IsCorrect     *bool
	AIScore       *int
	AIExplanation *string
	GradeStatus   string // "success" | "failed_local" | "failed_retryable"
	SelfRating    string // "again" | "hard" | "good" | "easy"
	AnsweredAt    time.Time
}

type Review struct {
	ID             int64
	QuestionID     int64
	Interval       int
	EasinessFactor float64
	Repetitions    int
	NextReviewAt   time.Time
	LastReviewedAt *time.Time
	CreatedAt      time.Time
}

type ReviewLog struct {
	ID         int64
	ReviewID   int64
	SelfRating string
	ReviewedAt time.Time
}

// ---------- Store interfaces ----------

type RepositoryStore interface {
	Save(ctx context.Context, r *Repository) error
	FindByPath(ctx context.Context, path string) (*Repository, error)
	List(ctx context.Context) ([]*Repository, error)
}

type CommitStore interface {
	Save(ctx context.Context, c *Commit) error
	FindByHash(ctx context.Context, repoID int64, hash string) (*Commit, error)
	ListUnreviewed(ctx context.Context, repoID int64) ([]*Commit, error)
	MarkReviewed(ctx context.Context, id int64) error
}

type SessionStore interface {
	Create(ctx context.Context, s *Session) error
	Update(ctx context.Context, s *Session) error
	FindByID(ctx context.Context, id int64) (*Session, error)
}

type QuestionStore interface {
	Save(ctx context.Context, q *Question) error
	FindByID(ctx context.Context, id int64) (*Question, error)
	ListBySession(ctx context.Context, sessionID int64) ([]*Question, error)
	MarkSaved(ctx context.Context, id int64) error
}

type AnswerStore interface {
	Save(ctx context.Context, a *Answer) error
	FindByQuestion(ctx context.Context, questionID int64) (*Answer, error)
	ListRetryable(ctx context.Context) ([]*Answer, error)
}

type ReviewStore interface {
	Create(ctx context.Context, r *Review) error
	Update(ctx context.Context, r *Review) error
	ListDue(ctx context.Context) ([]*Review, error)
	FindByQuestion(ctx context.Context, questionID int64) (*Review, error)
}

type ReviewLogStore interface {
	Append(ctx context.Context, log *ReviewLog) error
	ListByReview(ctx context.Context, reviewID int64) ([]*ReviewLog, error)
}
