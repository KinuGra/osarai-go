package styles

import "github.com/charmbracelet/lipgloss"

var (
	// Correct は正解時のテキストスタイル（緑）。
	Correct = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)

	// Incorrect は不正解時のテキストスタイル（オレンジ）。
	Incorrect = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)

	// Title はタイトル行のスタイル。
	Title = lipgloss.NewStyle().Bold(true).Underline(true)

	// Highlight は選択中の項目を強調するスタイル。
	Highlight = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)

	// Muted は補足情報などのグレースタイル。
	Muted = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	// Border はパネル外枠のスタイル。
	Border = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(0, 1)
)
