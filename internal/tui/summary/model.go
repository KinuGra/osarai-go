// Package summary はセッション結果表示画面の Bubble Tea サブモデルを提供する。
// 「N 問中 M 問正解！」と学習継続を促すメッセージを表示する。
package summary

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/KinuGra/osarai-go/internal/tui/styles"
)

// QuitMsg はサマリー画面からアプリを終了するメッセージ。
type QuitMsg struct{}

// Model はセッション結果表示画面のモデル。
type Model struct {
	total        int
	correctCount int
	maxStreak    int
}

// New はサマリー画面モデルを生成する。
func New(total, correctCount, maxStreak int) Model {
	return Model{
		total:        total,
		correctCount: correctCount,
		maxStreak:    maxStreak,
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", " ", "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) View() string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString(styles.Title.Render("📊 セッション結果"))
	sb.WriteString("\n\n")

	// 正解数
	correctLine := fmt.Sprintf("正解: %d / %d 問", m.correctCount, m.total)
	if m.correctCount == m.total {
		sb.WriteString(styles.Correct.Render("🎉  " + correctLine + "  パーフェクト！"))
	} else {
		sb.WriteString(styles.Highlight.Render("   " + correctLine))
	}
	sb.WriteString("\n")

	// 連続正解
	if m.maxStreak > 1 {
		sb.WriteString(styles.Muted.Render(
			fmt.Sprintf("   連続正解: %d 問 🔥", m.maxStreak),
		))
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	encouragement := encouragementMsg(m.correctCount, m.total)
	sb.WriteString(styles.Muted.Render(encouragement))
	sb.WriteString("\n\n")

	sb.WriteString(styles.Muted.Render("Enter で終了"))

	return sb.String()
}

// encouragementMsg は正解率に応じた励ましメッセージを返す。
func encouragementMsg(correct, total int) string {
	if total == 0 {
		return "お疲れ様でした！"
	}
	rate := float64(correct) / float64(total)
	switch {
	case rate == 1.0:
		return "完璧です！素晴らしい理解度 🌟"
	case rate >= 0.8:
		return "よく理解できています！引き続き頑張りましょう 💪"
	case rate >= 0.6:
		return "いい調子です！復習で知識を定着させましょう 📚"
	default:
		return "難しかったですね。復習で少しずつ定着させましょう 🌱"
	}
}
