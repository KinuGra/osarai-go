// Package confirm は「保存しますか？」などの二択確認ダイアログを提供する。
package confirm

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/KinuGra/osarai-go/internal/tui/styles"
)

// ConfirmedMsg はユーザーが選択を確定したときに返すメッセージ。
type ConfirmedMsg struct {
	Yes bool
}

// Model は確認ダイアログのモデル。
type Model struct {
	prompt  string
	yesText string
	noText  string
	cursor  int // 0=Yes, 1=No
	width   int
}

// New は確認ダイアログを生成する。
func New(prompt, yesText, noText string, width int) Model {
	return Model{
		prompt:  prompt,
		yesText: yesText,
		noText:  noText,
		cursor:  0,
		width:   width,
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
			if m.cursor < 1 {
				m.cursor++
			}
		case "enter", " ":
			yes := m.cursor == 0
			return m, func() tea.Msg { return ConfirmedMsg{Yes: yes} }
		case "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) View() string {
	var sb strings.Builder

	sb.WriteString(styles.Title.Render("💡 " + m.prompt))
	sb.WriteString("\n\n")

	items := []string{m.yesText, m.noText}
	for i, item := range items {
		prefix := "  "
		label := item
		if i == m.cursor {
			prefix = styles.Highlight.Render(" ❯")
			label = styles.Highlight.Render(item)
		}
		sb.WriteString(prefix + " " + label + "\n")
	}

	sb.WriteString("\n")
	sb.WriteString(styles.Muted.Render("↑↓ で選択  Enter で確定  Ctrl+C で終了"))

	return sb.String()
}
