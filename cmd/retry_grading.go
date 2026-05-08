package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var retryGradingRepo string

var retryGradingCmd = &cobra.Command{
	Use:   "retry-grading",
	Short: "採点失敗（failed_retryable）の回答を AI で再採点する",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("TODO: retry-grading")
		return nil
	},
}

func init() {
	retryGradingCmd.Flags().StringVar(&retryGradingRepo, "repo", "", "特定リポジトリに絞り込む")
}
