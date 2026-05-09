package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/KinuGra/osarai-go/internal/ai"
	"github.com/KinuGra/osarai-go/internal/config"
	"github.com/KinuGra/osarai-go/internal/core"
	"github.com/KinuGra/osarai-go/internal/db"
)

var retryGradingRepo string

var retryGradingCmd = &cobra.Command{
	Use:   "retry-grading",
	Short: "採点失敗（failed_retryable）の回答を AI で再採点する",
	Args:  cobra.NoArgs,
	RunE:  runRetryGrading,
}

func init() {
	retryGradingCmd.Flags().StringVar(&retryGradingRepo, "repo", "", "特定リポジトリに絞り込む")
}

func runRetryGrading(_ *cobra.Command, _ []string) error {
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
		core.WithGrader(ai.NewGrader(provider)),
	)

	fmt.Println("🔄 再採点を実行中...")
	n, err := svc.RetryGrading(ctx)
	if err != nil {
		return fmt.Errorf("再採点に失敗しました: %w", err)
	}

	if n == 0 {
		fmt.Println("再採点対象はありませんでした。")
	} else {
		fmt.Printf("✅ %d 件の再採点が完了しました。\n", n)
	}

	return nil
}
