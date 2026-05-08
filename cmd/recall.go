package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	recallRepo   string
	recallNew    bool
	recallReview bool
)

var recallCmd = &cobra.Command{
	Use:   "recall",
	Short: "保存済みコミットの問題を SM-2 スケジュールで復習する",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("TODO: recall")
		return nil
	},
}

func init() {
	recallCmd.Flags().StringVar(&recallRepo, "repo", "", "特定リポジトリに絞り込む")
	recallCmd.Flags().BoolVar(&recallNew, "new", false, "未振り返りコミットのみ対象にする")
	recallCmd.Flags().BoolVar(&recallReview, "review", false, "SM-2 復習分のみ対象にする")
}
