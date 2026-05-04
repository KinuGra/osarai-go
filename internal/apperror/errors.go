package apperror

import "fmt"

// ErrorCode はエラーの分類を表す。
// int ベースの独自型にすることで型安全になる（普通の int と混同しない）。
type ErrorCode int

const (
	// iota は Go の定数ジェネレーター。0, 1, 2, ... と自動で値が振られる。
	ErrCodeGitNotFound          ErrorCode = iota // git がインストールされていない
	ErrCodeRepoNotFound                          // .git が見つからない
	ErrCodeAPIKeyMissing                         // API キーが未設定
	ErrCodeRateLimit                             // API レートリミット
	ErrCodeDBLocked                              // DB ロックエラー
	ErrCodeNoQuestionsGenerated                  // AI が問題を生成できなかった
	ErrCodeNoDueReviews                          // 今日の復習対象がない
	ErrCodeGradingFailed                         // 採点 API エラー
)

// AppError はユーザー向けメッセージを持つアプリケーションエラー。
// Go の error インターフェースを満たしつつ、追加情報を持つ。
type AppError struct {
	Code    ErrorCode // エラー分類（上の const で定義）
	Message string    // ユーザー向けメッセージ（TUI や stderr に表示する用）
	Err     error     // 内部エラー（ログ・デバッグ用。ラップ元のエラー）
}

// Error() を実装することで error インターフェースを満たす。
// これにより AppError は error として扱える。
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap() を実装することで errors.Is() / errors.As() がチェーンを辿れる。
// 例: errors.Is(err, sql.ErrNoRows) で、AppError にラップされた元エラーも検出できる。
func (e *AppError) Unwrap() error {
	return e.Err
}

// UserMessage はTUI や stderr に表示するメッセージを返す。
// Error() は開発者向け（内部エラー込み）、UserMessage() はユーザー向け。
func (e *AppError) UserMessage() string {
	return e.Message
}

// New はAppErrorを生成するヘルパー。
func New(code ErrorCode, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// --- Sentinel errors（よく使うエラーを変数として事前定義）---
// 使い方: errors.Is(err, apperror.ErrAPIKeyMissing) で判定できる

var (
	ErrGitNotFound = &AppError{
		Code:    ErrCodeGitNotFound,
		Message: "git がインストールされていません。git をインストールしてください",
	}
	ErrRepoNotFound = &AppError{
		Code:    ErrCodeRepoNotFound,
		Message: "Git リポジトリが見つかりません。git init されたディレクトリで実行してください",
	}
	ErrAPIKeyMissing = &AppError{
		Code:    ErrCodeAPIKeyMissing,
		Message: "API キーが設定されていません。`osarai config init` で設定してください",
	}
	ErrRateLimit = &AppError{
		Code:    ErrCodeRateLimit,
		Message: "API のレートリミットに達しました。しばらく待ってから再試行してください",
	}
	ErrDBLocked = &AppError{
		Code:    ErrCodeDBLocked,
		Message: "データベースがロックされています。他の osarai プロセスが実行中かもしれません",
	}
	ErrNoQuestionsGenerated = &AppError{
		Code:    ErrCodeNoQuestionsGenerated,
		Message: "問題を生成できませんでした。差分が小さすぎるか、対応していないファイル形式かもしれません",
	}
	ErrNoDueReviews = &AppError{
		Code:    ErrCodeNoDueReviews,
		Message: "今日の復習対象はありません 🎉",
	}
	ErrGradingFailed = &AppError{
		Code:    ErrCodeGradingFailed,
		Message: "採点に失敗しました。`osarai retry-grading` で再採点できます",
	}
)
