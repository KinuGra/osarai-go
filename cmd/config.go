package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/KinuGra/osarai-go/internal/config"
	"github.com/KinuGra/osarai-go/internal/core"
)

var configList bool

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "アプリケーション設定を管理する",
	RunE: func(cmd *cobra.Command, _ []string) error {
		if configList {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("設定の読み込みに失敗しました: %w", err)
			}
			status := core.CheckEnv(cfg)
			core.PrintEnvStatus(status)
			return nil
		}
		return cmd.Help()
	},
}

func init() {
	configCmd.Flags().BoolVar(&configList, "list", false, "現在の設定と動作環境を表示する")
	configCmd.AddCommand(configInitCmd)
}
