package quiz

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/KinuGra/osarai-go/internal/ai"
	"github.com/KinuGra/osarai-go/internal/core"
)

func makeChoiceQuestion() core.CheckQuestion {
	return core.CheckQuestion{
		DBQuestionID: 1,
		DBSessionID:  1,
		Question: ai.Question{
			Title:    "テスト選択問題",
			Category: ai.QuestionCategoryLanguage,
			Type:     ai.QuestionTypeChoice,
			Body:     "テスト問題文",
			Choices:  []string{"A. 選択肢1", "B. 選択肢2", "C. 選択肢3", "D. 選択肢4"},
			CorrectAnswer: "B",
		},
	}
}

func makeWrittenQuestion() core.CheckQuestion {
	return core.CheckQuestion{
		DBQuestionID: 2,
		DBSessionID:  1,
		Question: ai.Question{
			Title:         "テスト記述問題",
			Category:      ai.QuestionCategoryLanguage,
			Type:          ai.QuestionTypeWritten,
			Body:          "テスト記述問題文",
			CorrectAnswer: "模範解答",
		},
	}
}

// TestChoiceCursorNavigation は選択肢のカーソル移動を確認する。
func TestChoiceCursorNavigation(t *testing.T) {
	m := New(makeChoiceQuestion(), 1, 5, 80, 24)

	// 初期カーソルは 0
	if m.cursor != 0 {
		t.Errorf("初期カーソル = %d, want 0", m.cursor)
	}

	// down で 1 に
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = m2.(Model)
	if m.cursor != 1 {
		t.Errorf("j 後のカーソル = %d, want 1", m.cursor)
	}

	// up で 0 に戻る
	m3, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m = m3.(Model)
	if m.cursor != 0 {
		t.Errorf("k 後のカーソル = %d, want 0", m.cursor)
	}
}

// TestChoiceAnsweredMsg は Enter で AnsweredMsg が返ることを確認する。
func TestChoiceAnsweredMsg(t *testing.T) {
	m := New(makeChoiceQuestion(), 1, 5, 80, 24)
	// カーソルを B(index=1) に移動
	m.cursor = 1

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Enter 後に cmd が nil")
	}
	msg := cmd()
	answered, ok := msg.(AnsweredMsg)
	if !ok {
		t.Fatalf("cmd() が AnsweredMsg を返さなかった: %T", msg)
	}
	if answered.UserAnswer != "B" {
		t.Errorf("UserAnswer = %q, want %q", answered.UserAnswer, "B")
	}
}

// TestWrittenTextInputForwarding は ① BlinkMsg 等が textinput に転送されることを確認する。
func TestWrittenTextInputForwarding(t *testing.T) {
	m := New(makeWrittenQuestion(), 1, 5, 80, 24)

	// BlinkMsg に似た非 KeyMsg を送る（textinput.BlinkMsg は非公開なので Blink Cmd 経由で代替）
	// ここでは textinput の Blink コマンドを実行して返ってくるメッセージを使う
	initCmd := m.Init()
	if initCmd == nil {
		t.Fatal("記述式の Init() が nil を返した（Blink コマンドが返るべき）")
	}
	blinkMsg := initCmd() // textinput.BlinkMsg が返るはず

	// blinkMsg を Update に渡す → textinput に転送されてコマンドが返るはず
	m2, cmd := m.Update(blinkMsg)
	if cmd == nil {
		t.Error("BlinkMsg 転送後、次の Blink cmd が返っていない（textinput への転送が機能していない）")
	}
	_ = m2
}

// TestWrittenAnswerSubmit は記述式で回答が送信されることを確認する。
func TestWrittenAnswerSubmit(t *testing.T) {
	m := New(makeWrittenQuestion(), 1, 5, 80, 24)

	// textinput に文字を直接セット
	m.textInput.SetValue("テスト回答")

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Enter 後に cmd が nil")
	}
	msg := cmd()
	answered, ok := msg.(AnsweredMsg)
	if !ok {
		t.Fatalf("cmd() が AnsweredMsg を返さなかった: %T", msg)
	}
	if answered.UserAnswer != "テスト回答" {
		t.Errorf("UserAnswer = %q, want %q", answered.UserAnswer, "テスト回答")
	}
}

// TestWrittenEmptyAnswerNotSubmitted は空文字は送信されないことを確認する。
func TestWrittenEmptyAnswerNotSubmitted(t *testing.T) {
	m := New(makeWrittenQuestion(), 1, 5, 80, 24)
	// textinput は空のまま

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("空文字で Enter しても AnsweredMsg が送信されるべきでない")
	}
}

// TestViewportInitialized は ② viewport が初期化されることを確認する。
func TestViewportInitialized(t *testing.T) {
	m := New(makeChoiceQuestion(), 1, 5, 120, 40)

	if !m.qvpReady {
		t.Error("width/height を渡したのに qvpReady が false")
	}
	if m.qvp.Width == 0 || m.qvp.Height == 0 {
		t.Errorf("viewport サイズが 0: %dx%d", m.qvp.Width, m.qvp.Height)
	}
}

// TestViewRenders は View() がパニックせず文字列を返すことを確認する。
func TestViewRenders(t *testing.T) {
	for _, tc := range []struct {
		name string
		cq   core.CheckQuestion
	}{
		{"選択式", makeChoiceQuestion()},
		{"記述式", makeWrittenQuestion()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := New(tc.cq, 1, 5, 80, 24)
			view := m.View()
			if view == "" {
				t.Error("View() が空文字を返した")
			}
			// 問題番号が含まれているか
			if !strings.Contains(view, "Q1/5") {
				t.Errorf("View() に問題番号が含まれていない:\n%s", view)
			}
		})
	}
}

// TestWindowSizeMsgUpdatesViewport は WindowSizeMsg で viewport が再初期化されることを確認する。
func TestWindowSizeMsgUpdatesViewport(t *testing.T) {
	m := New(makeChoiceQuestion(), 1, 5, 80, 24)
	oldH := m.qvp.Height

	m2, _ := m.Update(tea.WindowSizeMsg{Width: 200, Height: 60})
	m = m2.(Model)

	if m.width != 200 {
		t.Errorf("width が更新されていない: got %d, want 200", m.width)
	}
	if m.qvp.Height == oldH {
		t.Error("WindowSizeMsg 後、viewport 高さが変わっていない")
	}
}

// textinput.Model.SetValue は公開 API なので直接使えるが
// textinput.Blink の型が非公開のため Init() 経由でテストする。
var _ = textinput.New // import 確認
