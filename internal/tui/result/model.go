// Package result は採点結果・解説表示画面の Bubble Tea サブモデルを提供する。
// 正誤表示 + 参考書風テキスト解説をレンダリングし、
// ユーザーが Enter を押したら ProceedMsg を返して自己評価画面へ遷移させる。
package result

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

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

// Model は採点結果・解説表示画面のモデル。
type Model struct {
	cq         core.CheckQuestion
	gradeRes   core.GradeAnswerResult
	userAnswer string
}

// New は結果表示画面モデルを生成する。
func New(cq core.CheckQuestion, gradeRes core.GradeAnswerResult, userAnswer string) Model {
	return Model{
		cq:         cq,
		gradeRes:   gradeRes,
		userAnswer: userAnswer,
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
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
	return m, nil
}

func (m Model) View() string {
	r := m.gradeRes.Result
	q := m.cq.Question

	var sb strings.Builder

	// 正誤バナー
	if r.IsCorrect != nil {
		if *r.IsCorrect {
			sb.WriteString(styles.Correct.Render("✅  正解！"))
		} else {
			sb.WriteString(styles.Incorrect.Render("❌  不正解"))
			// 選択式の場合は正解を表示
			if q.Type == ai.QuestionTypeChoice {
				sb.WriteString(fmt.Sprintf("\n   正解: %s", styles.Correct.Render(q.CorrectAnswer)))
			}
		}
	} else {
		sb.WriteString(styles.Muted.Render("採点中... (retry-grading で後から採点できます)"))
	}
	sb.WriteString("\n\n")

	// あなたの回答
	sb.WriteString(styles.Muted.Render("あなたの回答: "))
	sb.WriteString(m.userAnswer)
	sb.WriteString("\n\n")

	// 解説
	if r.Explanation != "" {
		sb.WriteString(styles.Title.Render("📖 解説"))
		sb.WriteString("\n\n")
		sb.WriteString(styles.Border.Render(r.Explanation))
		sb.WriteString("\n\n")
	}

	sb.WriteString(styles.Muted.Render("Enter で次へ  Ctrl+C で終了"))

	return sb.String()
}
