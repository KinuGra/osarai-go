package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/KinuGra/osarai-go/internal/config"
	"github.com/KinuGra/osarai-go/internal/core"
	"github.com/KinuGra/osarai-go/internal/db"
)

var exportDir string

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "保存済み問題を Markdown ファイルにエクスポートする",
	Args:  cobra.NoArgs,
	RunE:  runExport,
}

func init() {
	exportCmd.Flags().StringVar(&exportDir, "dir", "", "エクスポート先ディレクトリ（省略時は ~/.osarai/exports）")
}

func runExport(_ *cobra.Command, _ []string) error {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("設定の読み込みに失敗しました: %w", err)
	}

	store, err := db.Open()
	if err != nil {
		return fmt.Errorf("データベースの初期化に失敗しました: %w", err)
	}
	defer store.Close() //nolint:errcheck

	svc := core.NewService(core.WithStore(store))

	outputDir := exportDir
	if outputDir == "" {
		outputDir = cfg.Export.Dir
	}

	opts := core.ExportOptions{OutputDir: outputDir}
	if err := svc.Export(ctx, opts); err != nil {
		return fmt.Errorf("%w", err)
	}

	fmt.Printf("✅ エクスポートが完了しました: %s\n", outputDir)
	return nil
}
