package ai

import "context"

// GradeRequest は採点リクエスト。
// 問題・ユーザーの回答・元の diff をセットにして Grader に渡す。
type GradeRequest struct {
	Question   Question // 元の問題
	UserAnswer string   // ユーザーの回答
	DiffBody   string   // 元の diff（文脈として AI に渡す）
}

// GradeResult は AI による採点結果。
type GradeResult struct {
	IsCorrect   bool   `json:"is_correct"`  // 正解かどうか
	Score       int    `json:"score"`       // 0〜100（記述式の部分点用）
	Explanation string `json:"explanation"` // 参考書風の解説（AI が生成）
	RawJSON     string // AI の生レスポンス（デバッグ用）
}

// Grader はユーザーの回答を採点する構造体。
type Grader struct {
	provider LLMProvider
}

// NewGrader は Grader を生成する。
func NewGrader(provider LLMProvider) *Grader {
	return &Grader{provider: provider}
}

// GradeChoice は選択式問題を採点する。
// TODO: 機能 B の担当者が実装する。
func (g *Grader) GradeChoice(ctx context.Context, req GradeRequest) (*GradeResult, error) {
	panic("not implemented")
}

// GradeWritten は記述式問題を採点する。
// TODO: 機能 B の担当者が実装する。
func (g *Grader) GradeWritten(ctx context.Context, req GradeRequest) (*GradeResult, error) {
	panic("not implemented")
}
