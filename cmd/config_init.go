package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "対話形式で初期設定を行い ~/.osarai/config.toml を生成する",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("TODO: config init")
		return nil
	},
}
