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

var (
	recallRepo   string
	recallNew    bool
	recallReview bool
)

var recallCmd = &cobra.Command{
	Use:   "recall",
	Short: "保存済みコミットの問題を SM-2 スケジュールで復習する",
	Args:  cobra.NoArgs,
	RunE:  runRecall,
}

func init() {
	recallCmd.Flags().StringVar(&recallRepo, "repo", "", "特定リポジトリに絞り込む")
	recallCmd.Flags().BoolVar(&recallNew, "new", false, "未振り返りコミットのみ対象にする")
	recallCmd.Flags().BoolVar(&recallReview, "review", false, "SM-2 復習分のみ対象にする")
}

func runRecall(_ *cobra.Command, _ []string) error {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("設定の読み込みに失敗しました: %w", err)
	}

	apiKey, err := cfg.APIKey()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		fmt.Fprintln(os.Stderr, "  → osarai config init で API キーを設定してください")
		return err
	}

	store, err := db.Open()
	if err != nil {
		return fmt.Errorf("データベースの初期化に失敗しました: %w", err)
	}
	defer store.Close() //nolint:errcheck

	provider, err := ai.NewGeminiProvider(apiKey, cfg.LLM.Gemini.Model)
	if err != nil {
		return fmt.Errorf("AI プロバイダーの初期化に失敗しました: %w", err)
	}

	svc := core.NewService(
		core.WithStore(store),
		core.WithGenerator(ai.NewGenerator(provider)),
		core.WithGrader(ai.NewGrader(provider)),
	)

	// failed_retryable バナー
	if n, err := svc.CountRetryable(); err == nil && n > 0 {
		fmt.Fprintf(os.Stderr, "⚠️  採点失敗が %d 件あります。`osarai retry-grading` で再採点できます。\n\n", n)
	}

	opts := core.RecallOptions{
		RepoName:   recallRepo,
		NewOnly:    recallNew,
		ReviewOnly: recallReview,
	}

	fmt.Println("🔍 復習対象を取得中...")
	questions, err := svc.RunRecall(ctx, opts)
	if err != nil {
		return fmt.Errorf("%w", err)
	}
	if len(questions) == 0 {
		fmt.Println("今日の復習対象はありません 🎉")
		return nil
	}

	fmt.Printf("✅ %d 問取得しました。復習を開始します...\n\n", len(questions))

	app := tui.NewApp(svc, questions)
	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI の実行に失敗しました: %w", err)
	}

	return nil
}
