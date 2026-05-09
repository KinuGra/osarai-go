package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/KinuGra/osarai-go/internal/core"
	"github.com/KinuGra/osarai-go/internal/db"
)

var listSaved bool

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "保存済みの問題一覧を表示する",
	Args:  cobra.NoArgs,
	RunE:  runList,
}

func init() {
	listCmd.Flags().BoolVar(&listSaved, "saved", false, "保存済みの問題のみ表示する")
}

func runList(_ *cobra.Command, _ []string) error {
	store, err := db.Open()
	if err != nil {
		return fmt.Errorf("データベースの初期化に失敗しました: %w", err)
	}
	defer store.Close() //nolint:errcheck

	svc := core.NewService(core.WithStore(store))

	items, err := svc.ListSaved()
	if err != nil {
		return fmt.Errorf("一覧の取得に失敗しました: %w", err)
	}

	if len(items) == 0 {
		fmt.Println("保存済みの問題はありません。")
		fmt.Println("クイズ後に「保存する」を選ぶと一覧に表示されます。")
		return nil
	}

	fmt.Printf("=== 保存済み問題一覧 (%d 件) ===\n\n", len(items))
	for i, item := range items {
		exportNote := ""
		if item.ExportPath != "" {
			exportNote = fmt.Sprintf(" → %s", item.ExportPath)
		}
		fmt.Printf("%d. [%s] %s (%s)%s\n",
			i+1, item.Category, item.Title, item.CreatedAt, exportNote)
	}

	return nil
}
