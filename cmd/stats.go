package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/KinuGra/osarai-go/internal/core"
	"github.com/KinuGra/osarai-go/internal/db"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "学習統計（正解率・復習対象数など）を表示する",
	Args:  cobra.NoArgs,
	RunE:  runStats,
}

func runStats(_ *cobra.Command, _ []string) error {
	ctx := context.Background()

	store, err := db.Open()
	if err != nil {
		return fmt.Errorf("データベースの初期化に失敗しました: %w", err)
	}
	defer store.Close() //nolint:errcheck

	svc := core.NewService(core.WithStore(store))

	stats, err := svc.GetStats(ctx)
	if err != nil {
		return fmt.Errorf("統計の取得に失敗しました: %w", err)
	}

	fmt.Println("=== 学習統計 ===")
	fmt.Println()
	fmt.Printf("📝 総問題数    : %d 問\n", stats.TotalQuestions)

	if stats.TotalQuestions > 0 {
		rate := float64(stats.CorrectAnswers) / float64(stats.TotalQuestions) * 100
		fmt.Printf("✅ 正解数      : %d 問 (%.1f%%)\n", stats.CorrectAnswers, rate)
	} else {
		fmt.Printf("✅ 正解数      : 0 問\n")
	}

	fmt.Printf("🔁 今日の復習  : %d 件\n", stats.DueReviews)
	fmt.Printf("⚠️  再採点対象  : %d 件\n", stats.RetryableCount)

	if stats.RetryableCount > 0 {
		fmt.Println()
		fmt.Println("  → `osarai retry-grading` で再採点できます")
	}

	return nil
}
