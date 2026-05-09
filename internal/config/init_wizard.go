package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// RunInitWizard は対話式セットアップウィザードを実行する。
// API キーを標準入力から受け取り、~/.osarai/config.toml に保存する。
func RunInitWizard() error {
	dir, err := configDir()
	if err != nil {
		return err
	}

	fmt.Println("=== おさらいGo セットアップウィザード ===")
	fmt.Println()

	// 既存 config.toml の読み込み（なければデフォルトから作成）
	if err := ensureConfigFile(dir); err != nil {
		return err
	}
	viper.SetConfigFile(filepath.Join(dir, "config.toml"))
	_ = viper.ReadInConfig()

	// Gemini API キーの入力
	fmt.Println("Gemini API キーを入力してください。")
	fmt.Println("（Google AI Studio: https://aistudio.google.com/ から取得できます）")
	fmt.Println()

	currentKey := viper.GetString("llm.gemini.api_key")
	if currentKey != "" {
		fmt.Printf("現在の API キー: %s...（上書きする場合は新しいキーを入力）\n", maskKey(currentKey))
		fmt.Print("新しいキー（変更しない場合は Enter）: ")
	} else {
		fmt.Print("Gemini API キー: ")
	}

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("入力の読み取りに失敗: %w", err)
	}
	apiKey := strings.TrimSpace(input)

	if apiKey == "" && currentKey != "" {
		fmt.Println()
		fmt.Println("✅ API キーはそのままです。")
	} else if apiKey == "" {
		return fmt.Errorf("API キーが入力されていません")
	} else {
		viper.Set("llm.gemini.api_key", apiKey)
		fmt.Println()
		fmt.Println("✅ API キーを設定しました。")
	}

	// config.toml に書き出し
	cfgPath := filepath.Join(dir, "config.toml")
	if err := viper.WriteConfigAs(cfgPath); err != nil {
		return fmt.Errorf("設定ファイルの書き込みに失敗: %w", err)
	}

	// パーミッションを 600 に制限（API キーが入るため）
	if err := os.Chmod(cfgPath, 0600); err != nil {
		fmt.Fprintf(os.Stderr, "warning: config.toml のパーミッション設定に失敗: %v\n", err)
	}

	fmt.Println()
	fmt.Printf("📁 設定ファイル: %s\n", cfgPath)
	fmt.Println()
	fmt.Println("セットアップ完了！`osarai check` で理解度チェックを開始できます。")

	return nil
}

// maskKey は API キーの先頭 8 文字以外をマスクする。
func maskKey(key string) string {
	runes := []rune(key)
	if len(runes) <= 8 {
		return strings.Repeat("*", len(runes))
	}
	return string(runes[:8]) + strings.Repeat("*", len(runes)-8)
}
