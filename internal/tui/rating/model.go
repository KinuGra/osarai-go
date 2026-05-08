// Package rating は SM-2 自己評価入力画面の Bubble Tea サブモデルを提供する。
// Again / Hard / Good / Easy の 4 段階をカーソル選択で入力する。
package rating

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/KinuGra/osarai-go/internal/tui/styles"
)

// SelectedMsg はユーザーが自己評価を選択したときに返す。
// app.go がこれを受け取り、次の問題または summary 画面へ遷移する。
type SelectedMsg struct {
	DBAnswerID int64
	Rating     string // "again" | "hard" | "good" | "easy"
}

type ratingOption struct {
	label       string
	value       string
	description string
}

var options = []ratingOption{
	{"Again", "again", "全く覚えていなかった"},
	{"Hard", "hard", "難しかったが思い出せた"},
	{"Good", "good", "少し考えて答えられた"},
	{"Easy", "easy", "すぐに答えられた"},
}

// Model は自己評価入力画面のモデル。
type Model struct {
	dbAnswerID int64
	cursor     int
	width      int
}

// New は自己評価画面モデルを生成する。
func New(dbAnswerID int64, width int) Model {
	return Model{
		dbAnswerID: dbAnswerID,
		cursor:     2, // デフォルトは "Good"
		width:      width,
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(options)-1 {
				m.cursor++
			}
		case "enter", " ":
			selected := options[m.cursor]
			return m, func() tea.Msg {
				return SelectedMsg{
					DBAnswerID: m.dbAnswerID,
					Rating:     selected.value,
				}
			}
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) View() string {
	var sb strings.Builder

	sb.WriteString(styles.Title.Render("理解度は？"))
	sb.WriteString("\n\n")

	for i, opt := range options {
		prefix := "  "
		label := opt.label
		desc := opt.description

		if i == m.cursor {
			prefix = styles.Highlight.Render(" ❯")
			label = styles.Highlight.Render(opt.label)
			desc = styles.Highlight.Render(opt.description)
		} else {
			desc = styles.Muted.Render(opt.description)
		}

		sb.WriteString(fmt.Sprintf("%s %-6s  %s\n", prefix, label, desc))
	}

	sb.WriteString("\n")
	sb.WriteString(styles.Muted.Render("↑↓ で選択  Enter で確定  Ctrl+C で終了"))

	return sb.String()
}
