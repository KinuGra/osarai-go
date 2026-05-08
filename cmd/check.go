package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var checkStaged bool

var checkCmd = &cobra.Command{
	Use:   "check [<file>]",
	Short: "未コミット差分から AI 問題を生成してクイズを開始する",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("TODO: check")
		return nil
	},
}

func init() {
	checkCmd.Flags().BoolVar(&checkStaged, "staged", false, "ステージ済みの差分のみ対象にする")
}
