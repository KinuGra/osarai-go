package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/KinuGra/osarai-go/internal/ai"
	"github.com/KinuGra/osarai-go/internal/core"
)

// ---- 画面遷移メッセージ型 ----

// NavigateToQuizMsg はクイズ画面へ遷移するメッセージ。
type NavigateToQuizMsg struct {
	Questions []ai.Question
}

// NavigateToResultMsg は結果表示画面へ遷移するメッセージ。
type NavigateToResultMsg struct {
	Result ai.GradeResult
}

// NavigateToRatingMsg は SM-2 評価入力画面へ遷移するメッセージ。
type NavigateToRatingMsg struct {
	Result ai.GradeResult
}

// NavigateToSummaryMsg はセッションサマリー画面へ遷移するメッセージ。
type NavigateToSummaryMsg struct{}

// QuitMsg はアプリケーションを終了するメッセージ。
type QuitMsg struct{}

// ---- App モデル ----

// App は TUI のルートモデル。画面遷移ルーターとして機能する。
// core.Service を保持し、各サブ画面モデルに DI する。
type App struct {
	service      *core.Service
	currentModel tea.Model
}

// NewApp は App を生成する。
func NewApp(service *core.Service) *App {
	return &App{
		service:      service,
		currentModel: newPlaceholderModel(),
	}
}

func (a *App) Init() tea.Cmd {
	return a.currentModel.Init()
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return a, tea.Quit
		}

	case NavigateToQuizMsg:
		// TODO: クイズ画面モデルに差し替える
		_ = msg.Questions
		return a, nil

	case NavigateToResultMsg:
		// TODO: 結果表示画面モデルに差し替える
		_ = msg.Result
		return a, nil

	case NavigateToRatingMsg:
		// TODO: SM-2 評価入力画面モデルに差し替える
		_ = msg.Result
		return a, nil

	case NavigateToSummaryMsg:
		// TODO: サマリー画面モデルに差し替える
		return a, nil

	case QuitMsg:
		return a, tea.Quit
	}

	var cmd tea.Cmd
	a.currentModel, cmd = a.currentModel.Update(msg)
	return a, cmd
}

func (a *App) View() string {
	return a.currentModel.View()
}

// ---- プレースホルダーモデル（後続 Issue で各画面に置き換える） ----

type placeholderModel struct{}

func newPlaceholderModel() placeholderModel { return placeholderModel{} }

func (p placeholderModel) Init() tea.Cmd                           { return nil }
func (p placeholderModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return p, nil }
func (p placeholderModel) View() string {
	return "おさらいGo へようこそ。\n\nCtrl+C で終了"
}
