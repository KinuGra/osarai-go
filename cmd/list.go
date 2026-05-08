package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var listSaved bool

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "保存済みの問題一覧を表示する",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("TODO: list")
		return nil
	},
}

func init() {
	listCmd.Flags().BoolVar(&listSaved, "saved", false, "保存済みの問題のみ表示する")
}
