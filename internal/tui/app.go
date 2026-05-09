// Package tui は Bubble Tea アプリケーションのルートモデルを提供する。
// app.go は各サブ画面モデルを切り替えるルーターとして機能し、
// ビジネスロジック（採点・SM-2 保存）の呼び出しも担当する。
package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/KinuGra/osarai-go/internal/core"
	"github.com/KinuGra/osarai-go/internal/tui/quiz"
	"github.com/KinuGra/osarai-go/internal/tui/rating"
	"github.com/KinuGra/osarai-go/internal/tui/result"
	"github.com/KinuGra/osarai-go/internal/tui/styles"
	"github.com/KinuGra/osarai-go/internal/tui/summary"
)

// ---- 内部メッセージ型（app.go 内でのみ使用）----

// gradedMsg は採点完了後に app.Update へ届く内部メッセージ。
type gradedMsg struct {
	cq         core.CheckQuestion
	gradeRes   core.GradeAnswerResult
	userAnswer string
}

// errMsg はエラー発生時に app.Update へ届く内部メッセージ。
type errMsg struct{ err error }

// ---- App モデル ----

// App は TUI のルートモデル。画面遷移ルーターとして機能する。
// core.Service を保持し、採点・SM-2 保存などのビジネスロジック呼び出しを行う。
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

	// ② ターミナルサイズ（WindowSizeMsg で更新）
	width  int
	height int

	// ① 採点中フラグ: true の間は AnsweredMsg を無視して二重採点・UNIQUE 違反を防ぐ
	grading bool
}

// NewApp は App を生成する。
// questions は Service.RunCheck() で取得済みの問題リスト。
func NewApp(service *core.Service, questions []core.CheckQuestion) *App {
	a := &App{
		service:   service,
		ctx:       context.Background(),
		questions: questions,
		width:     80, // WindowSizeMsg 到着前のデフォルト
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

	// ② ターミナルサイズを記録し、現在のサブモデルにも転送する
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		var cmd tea.Cmd
		a.currentModel, cmd = a.currentModel.Update(msg)
		return a, cmd

	// Ctrl+C はどの画面でも終了
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return a, tea.Quit
		}

	// quiz → 採点（非同期 Cmd）
	// ① 採点中の再送信は無視（Enter 連打による UNIQUE 制約違反を防ぐ）
	case quiz.AnsweredMsg:
		if a.grading {
			return a, nil
		}
		a.grading = true
		return a, a.gradeCmd(msg)

	// 採点完了 → 結果画面へ
	case gradedMsg:
		a.grading = false
		// 正解・連続正解カウントを更新
		if msg.gradeRes.Result.IsCorrect != nil && *msg.gradeRes.Result.IsCorrect {
			a.correctCount++
			a.streak++
			if a.streak > a.maxStreak {
				a.maxStreak = a.streak
			}
		} else {
			a.streak = 0
		}
		// ② width と height を渡して viewport を即時初期化
		a.currentModel = result.New(msg.cq, msg.gradeRes, msg.userAnswer, a.width, a.height)
		return a, a.currentModel.Init()

	// 結果画面 → 自己評価画面へ
	case result.ProceedMsg:
		// ② width を渡す
		a.currentModel = rating.New(msg.DBAnswerID, a.width)
		return a, a.currentModel.Init()

	// 自己評価選択 → 次の問題 or サマリーへ
	case rating.SelectedMsg:
		// TODO: ReviewLog を DB に保存（recall 実装 Issue で追加）
		a.advanceQuestion()
		return a, a.currentModel.Init()

	// エラー表示（採点失敗など）
	case errMsg:
		a.grading = false
		a.lastErr = msg.err
		return a, nil
	}

	// 現在のサブモデルに委譲
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

// gradeCmd は採点を非同期 tea.Cmd として実行する。
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
		// 送信された問題 ID に対応する CheckQuestion を探す
		var cq core.CheckQuestion
		for _, q := range a.questions {
			if q.DBQuestionID == msg.DBQuestionID {
				cq = q
				break
			}
		}
		return gradedMsg{
			cq:         cq,
			gradeRes:   res,
			userAnswer: msg.UserAnswer,
		}
	}
}

// advanceQuestion は次の問題またはサマリー画面へ遷移する。
func (a *App) advanceQuestion() {
	a.currentIdx++
	if a.currentIdx >= len(a.questions) {
		// ② width を渡す
		a.currentModel = summary.New(len(a.questions), a.correctCount, a.maxStreak, a.width)
	} else {
		// ② width を渡す
		a.currentModel = quiz.New(
			a.questions[a.currentIdx],
			a.currentIdx+1,
			len(a.questions),
			a.width,
			a.height,
		)
	}
}
