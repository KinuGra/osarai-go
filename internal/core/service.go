package core

import (
	"context"

	"github.com/KinuGra/osarai-go/internal/ai"
	"github.com/KinuGra/osarai-go/internal/db"
)

// Service は check / recall / stats / export / retry-grading の
// 全フローを統括するビジネスロジック層。
//
// tui からは Service のメソッドを呼ぶだけでよく、
// v3 で Web API ハンドラを追加する際も core 側の変更は不要。
type Service struct {
	questionStore  db.QuestionStore
	answerStore    db.AnswerStore
	reviewStore    db.ReviewStore
	reviewLogStore db.ReviewLogStore
	sessionStore   db.SessionStore
	commitStore    db.CommitStore
	repoStore      db.RepositoryStore
	generator      *ai.Generator
	grader         *ai.Grader
}

// Option は Service の依存を注入する Functional Options 型。
type Option func(*Service)

// WithStore は db.Store（複合インターフェース）から全 Store を一括設定する。
func WithStore(s db.Store) Option {
	return func(svc *Service) {
		svc.questionStore = s.Questions()
		svc.answerStore = s.Answers()
		svc.reviewStore = s.Reviews()
		svc.reviewLogStore = s.ReviewLogs()
		svc.sessionStore = s.Sessions()
		svc.commitStore = s.Commits()
		svc.repoStore = s.Repositories()
	}
}

func WithQuestionStore(s db.QuestionStore) Option {
	return func(svc *Service) { svc.questionStore = s }
}

func WithAnswerStore(s db.AnswerStore) Option {
	return func(svc *Service) { svc.answerStore = s }
}

func WithReviewStore(s db.ReviewStore) Option {
	return func(svc *Service) { svc.reviewStore = s }
}

func WithReviewLogStore(s db.ReviewLogStore) Option {
	return func(svc *Service) { svc.reviewLogStore = s }
}

func WithSessionStore(s db.SessionStore) Option {
	return func(svc *Service) { svc.sessionStore = s }
}

func WithCommitStore(s db.CommitStore) Option {
	return func(svc *Service) { svc.commitStore = s }
}

func WithRepoStore(s db.RepositoryStore) Option {
	return func(svc *Service) { svc.repoStore = s }
}

func WithGenerator(g *ai.Generator) Option {
	return func(svc *Service) { svc.generator = g }
}

func WithGrader(g *ai.Grader) Option {
	return func(svc *Service) { svc.grader = g }
}

// NewService は Functional Options パターンで Service を生成する。
func NewService(opts ...Option) *Service {
	svc := &Service{}
	for _, opt := range opts {
		opt(svc)
	}
	return svc
}

// ---- フロー用オプション型 ----

// CheckOptions は `osarai check` コマンドのオプション。
type CheckOptions struct {
	Staged   bool
	FilePath string
}

// RecallOptions は `osarai recall` コマンドのオプション。
type RecallOptions struct {
	RepoName   string
	NewOnly    bool
	ReviewOnly bool
}

// ExportOptions は `osarai export` コマンドのオプション。
type ExportOptions struct {
	OutputDir string
}

// Stats は学習統計データ。
type Stats struct {
	TotalQuestions int
	CorrectAnswers int
	DueReviews     int // 今日の SM-2 復習対象数
	RetryableCount int // 再採点対象数
}

// ---- フローメソッド ----

// RunCheck は未コミット差分から問題を生成して DB に保存し、[]CheckQuestion を返す。
func (s *Service) RunCheck(ctx context.Context, opts CheckOptions) ([]CheckQuestion, error) {
	return s.runCheck(ctx, opts)
}

// RunRecall は SM-2 復習対象 + 未振り返りコミットから問題を返す。
func (s *Service) RunRecall(ctx context.Context, opts RecallOptions) ([]CheckQuestion, error) {
	return s.runRecall(ctx, opts)
}

// GradeAnswer はユーザーの回答を採点し、DB に保存して結果を返す。
func (s *Service) GradeAnswer(ctx context.Context, req GradeAnswerRequest) (GradeAnswerResult, error) {
	return s.gradeAnswer(ctx, req)
}

// GradeAnswerForRecall は SM-2 復習アイテムのローカル採点を行う（新しい DB 回答は作らない）。
func (s *Service) GradeAnswerForRecall(_ context.Context, req GradeAnswerRequest) (GradeAnswerResult, error) {
	result := gradeLocally(req.Question, req.UserAnswer)
	// 既存回答の解説で上書き（AI 解説が保存されていれば使う）
	if req.ExistingResult != nil && req.ExistingResult.Result.Explanation != "" {
		result.Explanation = req.ExistingResult.Result.Explanation
	}
	return GradeAnswerResult{
		DBAnswerID: req.ExistingAnswerID,
		Result:     result,
	}, nil
}

// SaveRating は SM-2 自己評価を保存して review を更新する。
func (s *Service) SaveRating(ctx context.Context, req SaveRatingRequest) error {
	return s.saveRating(ctx, req)
}

// GetStats は学習統計を集計して返す。
func (s *Service) GetStats(ctx context.Context) (Stats, error) {
	return s.getStats(ctx)
}

// Export は保存済み問題を md ファイルにエクスポートする。
func (s *Service) Export(ctx context.Context, opts ExportOptions) error {
	return s.runExport(ctx, opts)
}

// ExportAnswer は指定 answerID の問題を md ファイルに書き出す。
func (s *Service) ExportAnswer(ctx context.Context, answerID int64, outputDir string) (string, error) {
	return s.exportAnswer(ctx, answerID, outputDir)
}

// RetryGrading は grade_status="failed_retryable" の回答を再採点する。
func (s *Service) RetryGrading(ctx context.Context) (int, error) {
	return s.retryGrading(ctx)
}

// ListSaved は保存済みの回答と問題一覧を返す。
func (s *Service) ListSaved() ([]SavedItem, error) {
	return s.listSaved()
}

// SavedItem は `osarai list --saved` 用のデータ。
type SavedItem struct {
	AnswerID   int64
	Title      string
	Category   string
	ExportPath string
	CreatedAt  string
}
