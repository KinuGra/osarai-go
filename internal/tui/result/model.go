// Package result は採点結果・解説表示画面の Bubble Tea サブモデルを提供する。
// 正誤バナー・ユーザー回答を固定ヘッダーとして表示し、
// 参考書風テキスト解説は bubbles/viewport でスクロール表示する。
// ユーザーが Enter を押したら ProceedMsg を返して自己評価画面へ遷移させる。
package result

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/KinuGra/osarai-go/internal/ai"
	"github.com/KinuGra/osarai-go/internal/core"
	"github.com/KinuGra/osarai-go/internal/tui/styles"
)

// ProceedMsg はユーザーが結果を確認して次へ進む操作をしたときに返す。
// app.go がこれを受け取り、自己評価画面（rating）へ遷移する。
type ProceedMsg struct {
	DBAnswerID  int64
	GradeResult ai.GradeResult
}

// ヘッダー・フッターの固定行数。
// viewport の高さはターミナル高さからこれらを差し引いて決定する。
const (
	headerLines = 8 // バナー + ユーザー回答 + 解説タイトル + 空行
	footerLines = 3 // スクロール % + ヒント行 + 余白
)

// Model は採点結果・解説表示画面のモデル。
type Model struct {
	cq         core.CheckQuestion
	gradeRes   core.GradeAnswerResult
	userAnswer string

	width    int
	height   int
	viewport viewport.Model
	ready    bool // viewport 初期化済みフラグ
}

// New は結果表示画面モデルを生成する。
// width・height が正値であれば viewport を即時初期化する。
func New(cq core.CheckQuestion, gradeRes core.GradeAnswerResult, userAnswer string, width, height int) Model {
	m := Model{
		cq:         cq,
		gradeRes:   gradeRes,
		userAnswer: userAnswer,
		width:      width,
		height:     height,
	}
	if width > 0 && height > 0 {
		m.initViewport()
	}
	return m
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// ② ターミナルリサイズ時に viewport を再設定する
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.initViewport()
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "enter", " ", "n":
			return m, func() tea.Msg {
				return ProceedMsg{
					DBAnswerID:  m.gradeRes.DBAnswerID,
					GradeResult: m.gradeRes.Result,
				}
			}
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	}

	// viewport にスクロール操作を転送する（↑↓ / j/k / pgup/pgdn 等）
	if m.ready {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) View() string {
	var sb strings.Builder

	// ---- 固定ヘッダー ----
	sb.WriteString(m.renderBanner())
	sb.WriteString("\n\n")
	sb.WriteString(styles.Muted.Render("あなたの回答: "))
	sb.WriteString(m.userAnswer)
	sb.WriteString("\n\n")
	sb.WriteString(styles.Title.Render("📖 解説"))
	sb.WriteString("\n")

	// ---- スクロール領域（viewport）----
	if !m.ready {
		// WindowSizeMsg 到着前の一時表示（幅制限つき）
		r := m.gradeRes.Result
		if r.Explanation != "" {
			fallbackW := m.width - 4
			if fallbackW < 20 {
				fallbackW = 76
			}
			sb.WriteString("\n")
			sb.WriteString(lipgloss.NewStyle().Width(fallbackW).Render(r.Explanation))
			sb.WriteString("\n")
		}
	} else {
		sb.WriteString(m.viewport.View())
		sb.WriteString("\n")
		// スクロール位置をパーセントで表示
		if m.viewport.TotalLineCount() > m.viewport.Height {
			pct := int(m.viewport.ScrollPercent() * 100)
			sb.WriteString(styles.Muted.Render(fmt.Sprintf("─── %d%% ───", pct)))
			sb.WriteString("\n")
		}
	}

	// ---- 固定フッター ----
	sb.WriteString("\n")
	sb.WriteString(styles.Muted.Render("↑↓/j/k でスクロール  Enter で次へ  Ctrl+C で終了"))

	return sb.String()
}

// renderBanner は正誤バナーを返す。
func (m Model) renderBanner() string {
	r := m.gradeRes.Result
	q := m.cq.Question

	if r.IsCorrect == nil {
		return styles.Muted.Render("採点中... (retry-grading で後から採点できます)")
	}
	if *r.IsCorrect {
		return styles.Correct.Render("✅  正解！")
	}

	var sb strings.Builder
	sb.WriteString(styles.Incorrect.Render("❌  不正解"))
	if q.Type == ai.QuestionTypeChoice {
		sb.WriteString(fmt.Sprintf("\n   正解: %s", styles.Correct.Render(q.CorrectAnswer)))
	}
	return sb.String()
}

// initViewport は viewport を現在の width/height で初期化（または再初期化）する。
func (m *Model) initViewport() {
	vpW := m.width - 2
	if vpW < 20 {
		vpW = 20
	}
	vpH := m.height - headerLines - footerLines
	if vpH < 3 {
		vpH = 3
	}

	m.viewport = viewport.New(vpW, vpH)
	m.viewport.SetContent(m.renderExplanation(vpW))
	m.ready = true
}

// renderExplanation は解説テキストを viewport 用にレンダリングする。
func (m Model) renderExplanation(width int) string {
	r := m.gradeRes.Result
	if r.Explanation == "" {
		return styles.Muted.Render("（解説なし）")
	}
	// ② 長い行を viewport 幅で折り返す
	return lipgloss.NewStyle().Width(width).Render(r.Explanation)
}

// スクロールヒント用に viewport のスクロール率を返すメソッドはそのまま
// viewport.ScrollPercent() を使う（上記 View() 参照）。
