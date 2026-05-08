package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "保存済み問題を Markdown ファイルにエクスポートする",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("TODO: export")
		return nil
	},
}
