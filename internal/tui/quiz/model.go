// Package quiz は出題画面の Bubble Tea サブモデルを提供する。
// 選択式（カーソル上下 + Enter）と記述式（テキスト入力 + Enter）を切り替えて表示する。
// 循環インポートを防ぐため、このパッケージは tui パッケージを import しない。
// 代わりに AnsweredMsg を返し、親の app.go が採点・画面遷移を担う。
package quiz

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
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

// 固定ヘッダー・フッターの行数定数。
// viewport の高さはターミナル高からこれらを差し引いて決定する。
const (
	// quizHeaderH: タイトル行(1) + 空行(1) = 2
	quizHeaderH = 2
	// quizChoiceH: 選択肢4行 + 空行(1) + ヒント(1) = 6
	quizChoiceH = 6
	// quizWrittenH: 入力ボーダー(3) + ヒント(1) = 4
	quizWrittenH = 4
	// viewport と下部コンテンツの間の空行
	quizSepLines = 1
)

// Model は出題画面のモデル。
type Model struct {
	cq        core.CheckQuestion
	cursor    int
	textInput textinput.Model

	current int
	total   int
	width   int
	height  int

	// ② 問題文の viewport（長い問題でも選択肢が画面外に押し出されなくなる）
	qvp      viewport.Model
	qvpReady bool
}

// New は出題画面モデルを生成する。
func New(cq core.CheckQuestion, current, total, width, height int) Model {
	ti := textinput.New()
	ti.Placeholder = "ここに回答を入力..."
	ti.CharLimit = 500

	if cq.Question.Type == ai.QuestionTypeWritten {
		ti.Focus()
	}

	m := Model{
		cq:        cq,
		cursor:    0,
		textInput: ti,
		current:   current,
		total:     total,
		width:     width,
		height:    height,
	}
	if width > 0 && height > 0 {
		m.initQVP()
	}
	return m
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
		m.height = msg.Height
		m.initQVP()
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
	// 選択肢カーソル操作
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(choices)-1 {
			m.cursor++
		}
	// ② 問題文のスクロール（Ctrl+U/D、または pgup/pgdn）
	case "ctrl+u", "pgup":
		if m.qvpReady {
			m.qvp.HalfPageUp()
		}
	case "ctrl+d", "pgdn":
		if m.qvpReady {
			m.qvp.HalfPageDown()
		}
	case "enter", " ":
		if len(choices) == 0 {
			return m, nil
		}
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
			return m, nil
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
	// ② 記述式では ↑↓ で問題文をスクロール（textinput は ↑↓ を使わない）
	case "up", "k":
		if m.qvpReady {
			m.qvp.ScrollUp(1)
		}
	case "down":
		if m.qvpReady {
			m.qvp.ScrollDown(1)
		}
	case "ctrl+u", "pgup":
		if m.qvpReady {
			m.qvp.HalfPageUp()
		}
	case "ctrl+d", "pgdn":
		if m.qvpReady {
			m.qvp.HalfPageDown()
		}
	case "ctrl+c":
		return m, tea.Quit
	}
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	q := m.cq.Question
	cw := m.contentWidth()

	var sb strings.Builder

	// ── 固定ヘッダー ──
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

	// ── 問題文（viewport or フォールバック）──
	if m.qvpReady {
		sb.WriteString(m.qvp.View())
	} else {
		sb.WriteString(lipgloss.NewStyle().Width(cw).Render(q.Body))
	}
	sb.WriteString("\n\n")

	// ── 選択肢 or テキスト入力 ──
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
		hint := "↑↓/j/k で選択  Ctrl+U/D で問題スクロール  Enter で決定"
		sb.WriteString(styles.Muted.Render(hint))

	case ai.QuestionTypeWritten:
		sb.WriteString(styles.Border.Width(cw - 4).Render(m.textInput.View()))
		sb.WriteString("\n")
		hint := "Enter で送信  ↑↓ で問題スクロール  Ctrl+C で終了"
		sb.WriteString(styles.Muted.Render(hint))
	}

	return sb.String()
}

// initQVP は問題文表示用 viewport を初期化（または再初期化）する。
// viewport の高さは word-wrap 後の実際の行数と端末の余白から小さい方を採用する。
// 問題文が短い場合は viewport がぴったりのサイズになり、選択肢・入力欄が直下に続く。
func (m *Model) initQVP() {
	vpW := m.width - 2
	if vpW < 20 {
		vpW = 20
	}

	var fixedH int
	if m.cq.Question.Type == ai.QuestionTypeChoice {
		fixedH = quizHeaderH + quizSepLines + quizChoiceH
	} else {
		fixedH = quizHeaderH + quizSepLines + quizWrittenH
	}

	maxVpH := m.height - fixedH
	if maxVpH < 3 {
		maxVpH = 3
	}

	// word-wrap 後の実際の行数を求め、それを超えない範囲で viewport を確保する
	content := lipgloss.NewStyle().Width(vpW).Render(m.cq.Question.Body)
	actualLines := strings.Count(content, "\n") + 1
	vpH := actualLines
	if vpH > maxVpH {
		vpH = maxVpH
	}

	m.qvp = viewport.New(vpW, vpH)
	m.qvp.SetContent(content)
	m.qvpReady = true
}

// contentWidth はフォールバック用のコンテンツ最大幅を返す。
func (m Model) contentWidth() int {
	if m.width <= 8 {
		return 76
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
