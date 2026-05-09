// Package tui は Bubble Tea アプリケーションのルートモデルを提供する。
// app.go は各サブ画面モデルを切り替えるルーターとして機能し、
// ビジネスロジック（採点・SM-2 保存）の呼び出しも担当する。
package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/KinuGra/osarai-go/internal/core"
	"github.com/KinuGra/osarai-go/internal/tui/components/confirm"
	"github.com/KinuGra/osarai-go/internal/tui/quiz"
	"github.com/KinuGra/osarai-go/internal/tui/rating"
	"github.com/KinuGra/osarai-go/internal/tui/result"
	"github.com/KinuGra/osarai-go/internal/tui/styles"
	"github.com/KinuGra/osarai-go/internal/tui/summary"
)

// ---- 内部メッセージ型 ----

type gradedMsg struct {
	cq         core.CheckQuestion
	gradeRes   core.GradeAnswerResult
	userAnswer string
}

type ratingDoneMsg struct{}

type errMsg struct{ err error }

// ---- App モデル ----

type App struct {
	service      *core.Service
	ctx          context.Context
	questions    []core.CheckQuestion
	currentIdx   int
	correctCount int
	streak       int
	maxStreak    int
	currentModel tea.Model
	lastErr      error

	width  int
	height int

	grading bool // 採点中フラグ（二重採点防止）

	// 直前の採点結果（confirm ダイアログ用）
	lastGradedMsg *gradedMsg
}

func NewApp(service *core.Service, questions []core.CheckQuestion) *App {
	a := &App{
		service:   service,
		ctx:       context.Background(),
		questions: questions,
		width:     80,
		height:    24,
	}
	a.currentModel = quiz.New(questions[0], 1, len(questions), a.width, a.height)
	return a
}

func (a *App) Init() tea.Cmd {
	return a.currentModel.Init()
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		var cmd tea.Cmd
		a.currentModel, cmd = a.currentModel.Update(msg)
		return a, cmd

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return a, tea.Quit
		}

	// quiz → 採点
	case quiz.AnsweredMsg:
		if a.grading {
			return a, nil
		}
		a.grading = true
		cq := a.questions[a.currentIdx]
		if cq.IsRecallReview {
			return a, a.gradeRecallCmd(msg, cq)
		}
		return a, a.gradeCmd(msg)

	// 採点完了 → 結果画面
	case gradedMsg:
		a.grading = false
		if msg.gradeRes.Result.IsCorrect != nil && *msg.gradeRes.Result.IsCorrect {
			a.correctCount++
			a.streak++
			if a.streak > a.maxStreak {
				a.maxStreak = a.streak
			}
		} else {
			a.streak = 0
		}
		a.lastGradedMsg = &msg
		a.currentModel = result.New(msg.cq, msg.gradeRes, msg.userAnswer, a.width, a.height)
		return a, a.currentModel.Init()

	// 結果画面 → 自己評価画面
	case result.ProceedMsg:
		a.currentModel = rating.New(msg.DBAnswerID, a.width)
		return a, a.currentModel.Init()

	// 自己評価選択 → SM-2 保存 → confirm ダイアログ
	case rating.SelectedMsg:
		return a, a.saveRatingCmd(msg)

	// SM-2 保存完了 → confirm ダイアログ（md 保存確認）
	case ratingDoneMsg:
		a.currentModel = confirm.New("md に保存しますか？", "保存する", "あとで", a.width)
		return a, a.currentModel.Init()

	// confirm 選択 → （保存 or スキップ）→ 次の問題
	case confirm.ConfirmedMsg:
		if msg.Yes && a.lastGradedMsg != nil {
			go func() {
				_, _ = a.service.ExportAnswer(
					a.ctx,
					a.lastGradedMsg.gradeRes.DBAnswerID,
					"",
				)
			}()
		}
		a.advanceQuestion()
		return a, a.currentModel.Init()

	case errMsg:
		a.grading = false
		a.lastErr = msg.err
		return a, nil
	}

	var cmd tea.Cmd
	a.currentModel, cmd = a.currentModel.Update(msg)
	return a, cmd
}

func (a *App) View() string {
	if a.lastErr != nil {
		return styles.Incorrect.Render("エラー: "+a.lastErr.Error()) +
			"\n\n" + styles.Muted.Render("Ctrl+C で終了")
	}
	return a.currentModel.View()
}

// gradeCmd は通常の採点（新規回答を DB に保存）を非同期で実行する。
func (a *App) gradeCmd(msg quiz.AnsweredMsg) tea.Cmd {
	return func() tea.Msg {
		res, err := a.service.GradeAnswer(a.ctx, core.GradeAnswerRequest{
			DBQuestionID: msg.DBQuestionID,
			Question:     msg.Question,
			UserAnswer:   msg.UserAnswer,
			DiffContext:  msg.DiffContext,
		})
		if err != nil {
			return errMsg{err: fmt.Errorf("採点に失敗しました: %w", err)}
		}
		var cq core.CheckQuestion
		for _, q := range a.questions {
			if q.DBQuestionID == msg.DBQuestionID {
				cq = q
				break
			}
		}
		return gradedMsg{cq: cq, gradeRes: res, userAnswer: msg.UserAnswer}
	}
}

// gradeRecallCmd は SM-2 復習アイテムのローカル採点を非同期で実行する。
// 新しい DB 回答は作らず、既存の解説を使う。
func (a *App) gradeRecallCmd(msg quiz.AnsweredMsg, cq core.CheckQuestion) tea.Cmd {
	return func() tea.Msg {
		res, err := a.service.GradeAnswerForRecall(a.ctx, core.GradeAnswerRequest{
			DBQuestionID:     msg.DBQuestionID,
			Question:         msg.Question,
			UserAnswer:       msg.UserAnswer,
			DiffContext:      msg.DiffContext,
			ExistingAnswerID: cq.ExistingAnswerID,
			ExistingResult:   cq.ExistingResult,
		})
		if err != nil {
			return errMsg{err: fmt.Errorf("採点に失敗しました: %w", err)}
		}
		return gradedMsg{cq: cq, gradeRes: res, userAnswer: msg.UserAnswer}
	}
}

// saveRatingCmd は SM-2 自己評価を DB に保存する非同期コマンド。
func (a *App) saveRatingCmd(msg rating.SelectedMsg) tea.Cmd {
	return func() tea.Msg {
		_ = a.service.SaveRating(a.ctx, core.SaveRatingRequest{
			DBAnswerID: msg.DBAnswerID,
			Rating:     msg.Rating,
		})
		return ratingDoneMsg{}
	}
}

// advanceQuestion は次の問題またはサマリー画面へ遷移する。
func (a *App) advanceQuestion() {
	a.currentIdx++
	if a.currentIdx >= len(a.questions) {
		a.currentModel = summary.New(len(a.questions), a.correctCount, a.maxStreak, a.width)
	} else {
		a.currentModel = quiz.New(
			a.questions[a.currentIdx],
			a.currentIdx+1,
			len(a.questions),
			a.width,
			a.height,
		)
	}
}
