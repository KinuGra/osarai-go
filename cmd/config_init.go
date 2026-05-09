package cmd

import (
	"github.com/spf13/cobra"

	"github.com/KinuGra/osarai-go/internal/config"
)

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "対話形式で初期設定を行い ~/.osarai/config.toml を生成する",
	Args:  cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		return config.RunInitWizard()
	},
}
