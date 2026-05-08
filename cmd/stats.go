package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "学習統計（正解率・連続日数など）を表示する",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("TODO: stats")
		return nil
	},
}
