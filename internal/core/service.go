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
// 個別 WithXxxStore と組み合わせる場合、後から呼んだ方が優先される。
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
//
// 使用例:
//
//	provider, _ := ai.NewGeminiProvider(apiKey, "")
//	svc := core.NewService(
//	    core.WithStore(sqliteStore),
//	    core.WithGenerator(ai.NewGenerator(provider)),
//	    core.WithGrader(ai.NewGrader(provider)),
//	)
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
	Staged   bool   // --staged: ステージ済みの差分のみ対象
	FilePath string // <file>: 特定ファイルのみ対象（空文字 = 全体）
}

// RecallOptions は `osarai recall` コマンドのオプション。
type RecallOptions struct {
	RepoName   string // --repo: 特定リポジトリに絞り込む（空文字 = 全リポ）
	NewOnly    bool   // --new: 未振り返りコミットのみ
	ReviewOnly bool   // --review: SM-2 復習分のみ
}

// ExportOptions は `osarai export` コマンドのオプション。
type ExportOptions struct {
	OutputDir string // エクスポート先ディレクトリ（空文字 = config のデフォルト）
}

// Stats は学習統計データ。
type Stats struct {
	TotalSessions  int
	TotalQuestions int
	CorrectAnswers int
	StudyDayStreak int // 連続学習日数
	MaxStreak      int // 連続正解数の最大値
}

// ---- フローメソッド ----

// RunCheck は未コミット差分から問題を生成して DB に保存し、[]CheckQuestion を返す。
// 実装は core/check.go の runCheck に委譲する。
func (s *Service) RunCheck(ctx context.Context, opts CheckOptions) ([]CheckQuestion, error) {
	return s.runCheck(ctx, opts)
}

// RunRecall は SM-2 復習対象 + 未振り返りコミットから問題を生成して返す。
// TODO: core/recall.go で実装する。
func (s *Service) RunRecall(ctx context.Context, opts RecallOptions) ([]CheckQuestion, error) {
	panic("not implemented")
}

// GradeAnswer はユーザーの回答を採点し、DB に保存して結果を返す。
// 実装は core/grading.go の gradeAnswer に委譲する。
func (s *Service) GradeAnswer(ctx context.Context, req GradeAnswerRequest) (GradeAnswerResult, error) {
	return s.gradeAnswer(ctx, req)
}

// GetStats は学習統計を集計して返す。
// TODO: core/stats.go で実装する。
func (s *Service) GetStats(ctx context.Context) (Stats, error) {
	panic("not implemented")
}

// Export は保存済み問題を md ファイルにエクスポートする。
// TODO: core/export.go で実装する。
func (s *Service) Export(ctx context.Context, opts ExportOptions) error {
	panic("not implemented")
}

// RetryGrading は grade_status="failed_retryable" の回答を再採点する。
// TODO: core/grading.go で実装する。
func (s *Service) RetryGrading(ctx context.Context) error {
	panic("not implemented")
}
