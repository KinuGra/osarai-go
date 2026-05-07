package config

import (
	_ "embed" // []byte への go:embed を有効にするブランクインポート
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"

	"github.com/KinuGra/osarai-go/internal/apperror"
)

//go:embed defaults/config.default.toml
var defaultConfigBytes []byte

// Config はアプリケーション設定を保持する構造体。
// Viper から読み込んだ値をここにマッピングする。
type Config struct {
	LLM       LLMConfig       `mapstructure:"llm"`
	Questions QuestionsConfig `mapstructure:"questions"`
	Export    ExportConfig    `mapstructure:"export"`
}

// LLMConfig は LLM 関連の設定。
type LLMConfig struct {
	Provider string       `mapstructure:"provider"`
	Gemini   GeminiConfig `mapstructure:"gemini"`
}

// GeminiConfig は Gemini API の設定。
type GeminiConfig struct {
	APIKey string `mapstructure:"api_key"`
	Model  string `mapstructure:"model"`
}

// QuestionsConfig は問題生成の設定。
type QuestionsConfig struct {
	ChoiceCount  int `mapstructure:"choice_count"`
	WrittenCount int `mapstructure:"written_count"`
}

// ExportConfig はエクスポートの設定。
type ExportConfig struct {
	Dir string `mapstructure:"dir"`
}

// configDir は設定ディレクトリのパスを返す。
// ~/.osarai/
func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("ホームディレクトリの取得に失敗: %w", err)
	}
	return filepath.Join(home, ".osarai"), nil
}

// ensureConfigFile は ~/.osarai/config.toml がなければデフォルトからコピーして作成する。
func ensureConfigFile(dir string) error {
	configPath := filepath.Join(dir, "config.toml")

	// config.toml が既に存在すれば何もしない
	if _, err := os.Stat(configPath); err == nil {
		return nil
	}

	// ディレクトリがなければ作成（0700 = 所有者のみ rwx）
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("設定ディレクトリの作成に失敗: %w", err)
	}

	// デフォルト設定を書き出す（0600 = 所有者のみ rw。API キーが入るため）
	if err := os.WriteFile(configPath, defaultConfigBytes, 0600); err != nil {
		return fmt.Errorf("設定ファイルの作成に失敗: %w", err)
	}

	return nil
}

// Load は設定ファイルを読み込んで Config を返す。
//
// 優先順位（高い方が勝つ）:
//  1. 環境変数（GEMINI_API_KEY）
//  2. ~/.osarai/config.toml
//  3. go:embed のデフォルト値
func Load() (*Config, error) {
	dir, err := configDir()
	if err != nil {
		return nil, err
	}

	// config.toml がなければデフォルトから作成
	if err := ensureConfigFile(dir); err != nil {
		return nil, err
	}

	// --- Viper の設定 ---

	// デフォルト値を embedded ファイルから読み込む
	viper.SetConfigType("toml")
	if err := viper.ReadConfig(strings.NewReader(string(defaultConfigBytes))); err != nil {
		return nil, fmt.Errorf("デフォルト設定の読み込みに失敗: %w", err)
	}

	// ユーザーの config.toml で上書き
	viper.SetConfigFile(filepath.Join(dir, "config.toml"))
	_ = viper.MergeInConfig() // ファイルがなくてもデフォルトで動くので無視

	// 環境変数のバインド
	// GEMINI_API_KEY が設定されていれば、config.toml の llm.gemini.api_key より優先される
	_ = viper.BindEnv("llm.gemini.api_key", "GEMINI_API_KEY")

	// Viper の値を Config 構造体にマッピング
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("設定の読み込みに失敗: %w", err)
	}

	// ~ をホームディレクトリに展開
	if strings.HasPrefix(cfg.Export.Dir, "~") {
		home, _ := os.UserHomeDir()
		cfg.Export.Dir = filepath.Join(home, cfg.Export.Dir[1:])
	}

	return &cfg, nil
}

// APIKey は Gemini API キーを返す。未設定なら ErrAPIKeyMissing を返す。
func (c *Config) APIKey() (string, error) {
	if c.LLM.Gemini.APIKey == "" {
		return "", apperror.ErrAPIKeyMissing
	}
	return c.LLM.Gemini.APIKey, nil
}

// Save は現在の設定を ~/.osarai/config.toml に書き出す。
// config init ウィザード（D-2）で使う。
func Save() error {
	dir, err := configDir()
	if err != nil {
		return err
	}
	return viper.WriteConfigAs(filepath.Join(dir, "config.toml"))
}
