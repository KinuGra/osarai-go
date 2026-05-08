package cmd

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/KinuGra/osarai-go/internal/ai"
	"github.com/KinuGra/osarai-go/internal/config"
	"github.com/KinuGra/osarai-go/internal/core"
	"github.com/KinuGra/osarai-go/internal/db"
	"github.com/KinuGra/osarai-go/internal/tui"
)

var checkStaged bool

var checkCmd = &cobra.Command{
	Use:   "check [<file>]",
	Short: "未コミット差分から AI 問題を生成してクイズを開始する",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runCheck,
}

func init() {
	checkCmd.Flags().BoolVar(&checkStaged, "staged", false, "ステージ済みの差分のみ対象にする")
}

func runCheck(_ *cobra.Command, args []string) error {
	ctx := context.Background()

	// 1. 設定読み込み
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("設定の読み込みに失敗しました: %w", err)
	}

	apiKey, err := cfg.APIKey()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		fmt.Fprintln(os.Stderr, "  → osarai config init で API キーを設定してください")
		return nil
	}

	// 2. DB を開く
	store, err := db.Open()
	if err != nil {
		return fmt.Errorf("データベースの初期化に失敗しました: %w", err)
	}
	defer store.Close() //nolint:errcheck

	// 3. AI プロバイダーと Service を構築
	provider, err := ai.NewGeminiProvider(apiKey, cfg.LLM.Gemini.Model)
	if err != nil {
		return fmt.Errorf("AI プロバイダーの初期化に失敗しました: %w", err)
	}

	svc := core.NewService(
		core.WithStore(store),
		core.WithGenerator(ai.NewGenerator(provider)),
		// Grader は MVP フェーズでは注入しない（ローカル採点モックを使用）
	)

	// 4. オプション構築
	opts := core.CheckOptions{
		Staged: checkStaged,
	}
	if len(args) > 0 {
		opts.FilePath = args[0]
	}

	// 5. 問題生成（TUI 外で行い、ロード中メッセージを表示）
	fmt.Println("🔍 差分を取得して問題を生成中...")
	questions, err := svc.RunCheck(ctx, opts)
	if err != nil {
		return fmt.Errorf("%w", err)
	}
	if len(questions) == 0 {
		fmt.Println("問題を生成できませんでした。差分が小さすぎるかもしれません。")
		return nil
	}

	fmt.Printf("✅ %d 問生成しました。クイズを開始します...\n\n", len(questions))

	// 6. TUI を起動
	app := tui.NewApp(svc, questions)
	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI の実行に失敗しました: %w", err)
	}

	return nil
}
