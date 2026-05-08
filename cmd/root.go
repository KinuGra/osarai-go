package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:           "osarai",
	Short:         "おさらいGo — git diff から AI 問題を生成し、SM-2 で復習管理する TUI ツール",
	SilenceErrors: true,
}

// Execute はルートコマンドを実行する。main から呼ぶ。
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.AddCommand(
		checkCmd,
		recallCmd,
		statsCmd,
		configCmd,
		exportCmd,
		listCmd,
		retryGradingCmd,
	)
}

func initConfig() {
	viper.SetEnvPrefix("OSARAI")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
}
