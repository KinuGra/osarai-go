package tui

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/KinuGra/osarai-go/internal/ai"
	"github.com/KinuGra/osarai-go/internal/core"
	"github.com/KinuGra/osarai-go/internal/tui/quiz"
)

// stubService は core.Service の代わりに使うテスト用スタブ。
// GradeAnswer を呼ばれても何もしない（採点は非同期 Cmd なので直接呼ばれない）。
type stubService struct{}

func makeTestApp() *App {
	questions := []core.CheckQuestion{
		{
			DBQuestionID: 1,
			DBSessionID:  1,
			Question: ai.Question{
				Title:         "テスト問題",
				Category:      ai.QuestionCategoryLanguage,
				Type:          ai.QuestionTypeChoice,
				Body:          "テスト問題文",
				Choices:       []string{"A. 選択肢1", "B. 選択肢2", "C. 選択肢3", "D. 選択肢4"},
				CorrectAnswer: "A",
				Explanation:   "テスト解説",
			},
		},
	}
	// core.Service ゼロ値（grader/generator は nil）を使う
	svc := core.NewService()
	_ = context.Background()
	return NewApp(svc, questions)
}

// TestGradingFlag は採点中フラグが二重 AnsweredMsg を防ぐことを確認する。
func TestGradingFlag(t *testing.T) {
	app := makeTestApp()

	answeredMsg := quiz.AnsweredMsg{
		DBQuestionID: 1,
		DBSessionID:  1,
		Question:     app.questions[0].Question,
		UserAnswer:   "A",
	}

	// 1 回目の AnsweredMsg → grading=true になり gradeCmd が返るはず
	model, cmd := app.Update(answeredMsg)
	appAfter, ok := model.(*App)
	if !ok {
		t.Fatal("Update() が *App を返さなかった")
	}
	if !appAfter.grading {
		t.Error("1 回目の AnsweredMsg 後、grading フラグが true になっていない")
	}
	if cmd == nil {
		t.Error("1 回目の AnsweredMsg では gradeCmd が返るべき")
	}

	// 2 回目の AnsweredMsg（採点中）→ grading=true のまま、cmd は nil になるはず
	model2, cmd2 := appAfter.Update(answeredMsg)
	appAfter2, ok := model2.(*App)
	if !ok {
		t.Fatal("2 回目の Update() が *App を返さなかった")
	}
	if !appAfter2.grading {
		t.Error("2 回目の AnsweredMsg 後、grading フラグが true のままであるべき")
	}
	if cmd2 != nil {
		// nil を直接比較できないため、Cmd を実行して nil メッセージが返るかで判定
		msg := cmd2()
		if msg != nil {
			t.Errorf("採点中の二重 AnsweredMsg: cmd が nil でないメッセージを返した: %T", msg)
		}
	}
}

// TestGradingFlagClearedOnGraded は gradedMsg で grading フラグが解除されることを確認する。
func TestGradingFlagClearedOnGraded(t *testing.T) {
	app := makeTestApp()
	app.grading = true // 採点中の状態にセット

	isCorrect := true
	gm := gradedMsg{
		cq: app.questions[0],
		gradeRes: core.GradeAnswerResult{
			DBAnswerID: 1,
			Result: ai.GradeResult{
				IsCorrect:   &isCorrect,
				Explanation: "解説",
				GradeStatus: ai.GradeStatusLocalOnly,
			},
		},
		userAnswer: "A",
	}

	model, _ := app.Update(gm)
	appAfter, ok := model.(*App)
	if !ok {
		t.Fatal("Update() が *App を返さなかった")
	}
	if appAfter.grading {
		t.Error("gradedMsg 受信後、grading フラグが false になっていない")
	}
}

// TestGradingFlagClearedOnError は errMsg で grading フラグが解除されることを確認する。
func TestGradingFlagClearedOnError(t *testing.T) {
	app := makeTestApp()
	app.grading = true

	model, _ := app.Update(errMsg{err: nil})
	appAfter, ok := model.(*App)
	if !ok {
		t.Fatal("Update() が *App を返さなかった")
	}
	if appAfter.grading {
		t.Error("errMsg 受信後、grading フラグが false になっていない")
	}
}

// TestWindowSizeStored は WindowSizeMsg でサイズが記録されることを確認する。
func TestWindowSizeStored(t *testing.T) {
	app := makeTestApp()

	model, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	appAfter, ok := model.(*App)
	if !ok {
		t.Fatal("Update() が *App を返さなかった")
	}
	if appAfter.width != 120 || appAfter.height != 40 {
		t.Errorf("サイズが正しく記録されていない: got %dx%d, want 120x40",
			appAfter.width, appAfter.height)
	}
}
