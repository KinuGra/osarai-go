package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var configList bool

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "アプリケーション設定を管理する",
	RunE: func(cmd *cobra.Command, args []string) error {
		if configList {
			fmt.Println("TODO: config --list")
			return nil
		}
		return cmd.Help()
	},
}

func init() {
	configCmd.Flags().BoolVar(&configList, "list", false, "現在の設定を一覧表示する")
	configCmd.AddCommand(configInitCmd)
}
