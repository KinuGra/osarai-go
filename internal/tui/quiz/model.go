// Package quiz は出題画面の Bubble Tea サブモデルを提供する。
// 選択式（カーソル上下 + Enter）と記述式（テキスト入力 + Enter）を切り替えて表示する。
// 循環インポートを防ぐため、このパッケージは tui パッケージを import しない。
// 代わりに AnsweredMsg を返し、親の app.go が採点・画面遷移を担う。
package quiz

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/KinuGra/osarai-go/internal/ai"
	"github.com/KinuGra/osarai-go/internal/core"
	"github.com/KinuGra/osarai-go/internal/tui/styles"
)

// AnsweredMsg はユーザーが回答を確定したときに返すメッセージ。
// app.go がこれを受け取り、採点 → 結果画面への遷移を行う。
type AnsweredMsg struct {
	DBQuestionID int64
	DBSessionID  int64
	Question     ai.Question
	UserAnswer   string
	DiffContext  string
}

// Model は出題画面のモデル。
type Model struct {
	cq        core.CheckQuestion // 現在の問題
	cursor    int                // 選択式: 選択肢のカーソル位置（0-based）
	textInput textinput.Model    // 記述式: テキスト入力

	// 問題番号表示用
	current int // 1-based
	total   int

	width int // ターミナル幅（0 = 未取得）
}

// New は出題画面モデルを生成する。
func New(cq core.CheckQuestion, current, total, width int) Model {
	ti := textinput.New()
	ti.Placeholder = "ここに回答を入力..."
	ti.CharLimit = 500

	if cq.Question.Type == ai.QuestionTypeWritten {
		ti.Focus()
	}

	return Model{
		cq:        cq,
		cursor:    0,
		textInput: ti,
		current:   current,
		total:     total,
		width:     width,
	}
}

func (m Model) Init() tea.Cmd {
	if m.cq.Question.Type == ai.QuestionTypeWritten {
		return textinput.Blink
	}
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil
	case tea.KeyMsg:
		switch m.cq.Question.Type {
		case ai.QuestionTypeChoice:
			return m.updateChoice(msg)
		case ai.QuestionTypeWritten:
			return m.updateWritten(msg)
		}
	}
	// ① KeyMsg 以外のメッセージ（BlinkMsg・FocusMsg 等）を記述式の textinput に転送する。
	// textinput はカーソル点滅などで自身に BlinkMsg を送り続けるため、
	// これを転送しないと入力が一切反応しなくなる。
	if m.cq.Question.Type == ai.QuestionTypeWritten {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}
	return m, nil
}

// updateChoice は選択式問題のキー操作を処理する。
func (m Model) updateChoice(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	choices := m.cq.Question.Choices
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(choices)-1 {
			m.cursor++
		}
	case "enter", " ":
		if len(choices) == 0 {
			return m, nil
		}
		// 選択肢のラベル（"A"〜"D"）を抽出して回答とする
		answer := choiceLabel(m.cursor)
		return m, func() tea.Msg {
			return AnsweredMsg{
				DBQuestionID: m.cq.DBQuestionID,
				DBSessionID:  m.cq.DBSessionID,
				Question:     m.cq.Question,
				UserAnswer:   answer,
				DiffContext:  m.cq.DiffContext,
			}
		}
	}
	return m, nil
}

// updateWritten は記述式問題のキー操作を処理する。
func (m Model) updateWritten(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		answer := strings.TrimSpace(m.textInput.Value())
		if answer == "" {
			return m, nil // 空文字は送信しない
		}
		return m, func() tea.Msg {
			return AnsweredMsg{
				DBQuestionID: m.cq.DBQuestionID,
				DBSessionID:  m.cq.DBSessionID,
				Question:     m.cq.Question,
				UserAnswer:   answer,
				DiffContext:  m.cq.DiffContext,
			}
		}
	case "ctrl+c":
		return m, tea.Quit
	}
	// テキスト入力コンポーネントに委譲
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	q := m.cq.Question

	// ② コンテンツ幅を terminal 幅に合わせる
	cw := m.contentWidth()

	var sb strings.Builder

	// ヘッダー: "Q2/5 [language] 選択式"
	typeLabel := "選択式"
	if q.Type == ai.QuestionTypeWritten {
		typeLabel = "記述式"
	}
	header := fmt.Sprintf("Q%d/%d  %s  %s",
		m.current, m.total,
		styles.Muted.Render("["+string(q.Category)+"]"),
		styles.Muted.Render(typeLabel),
	)
	sb.WriteString(styles.Title.Render(header))
	sb.WriteString("\n\n")

	// 問題文（長い場合に折り返す）
	sb.WriteString(lipgloss.NewStyle().Width(cw).Render(q.Body))
	sb.WriteString("\n\n")

	// 選択肢 or テキスト入力
	switch q.Type {
	case ai.QuestionTypeChoice:
		for i, choice := range q.Choices {
			prefix := "  "
			line := lipgloss.NewStyle().Width(cw - 4).Render(choice)
			if i == m.cursor {
				prefix = styles.Highlight.Render(" ❯")
				line = styles.Highlight.Render(lipgloss.NewStyle().Width(cw - 4).Render(choice))
			}
			sb.WriteString(fmt.Sprintf("%s %s\n", prefix, line))
		}
		sb.WriteString("\n")
		sb.WriteString(styles.Muted.Render("↑↓ で移動  Enter で決定  Ctrl+C で終了"))

	case ai.QuestionTypeWritten:
		sb.WriteString(styles.Border.Width(cw - 4).Render(m.textInput.View()))
		sb.WriteString("\n")
		sb.WriteString(styles.Muted.Render("Enter で送信  Ctrl+C で終了"))
	}

	return sb.String()
}

// contentWidth は表示コンテンツの最大幅を返す。
// 幅未取得時はデフォルト値を返す。
func (m Model) contentWidth() int {
	if m.width <= 8 {
		return 76 // デフォルト
	}
	return m.width - 4
}

// choiceLabel は 0-based インデックスを "A"〜"D" に変換する。
func choiceLabel(idx int) string {
	labels := []string{"A", "B", "C", "D", "E"}
	if idx < 0 || idx >= len(labels) {
		return ""
	}
	return labels[idx]
}
