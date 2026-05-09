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
	Short:         "おさらいGo — AI 生成コードの\"わかったつもり\"をなくし、知識として定着させるターミナル TUI アプリ",
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
