package core

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/KinuGra/osarai-go/internal/config"
)

// EnvStatus は環境チェックの結果。
type EnvStatus struct {
	GitInstalled bool
	GitVersion   string
	APIKeySet    bool
	APIProvider  string
	DBPath       string
}

// CheckEnv は Git インストール・API キー設定・DB パスを確認して返す。
func CheckEnv(cfg *config.Config) EnvStatus {
	status := EnvStatus{
		DBPath: dbPath(),
	}

	// Git チェック
	if out, err := exec.Command("git", "--version").Output(); err == nil {
		status.GitInstalled = true
		status.GitVersion = strings.TrimSpace(string(out))
	}

	// API キーチェック
	if _, err := cfg.APIKey(); err == nil {
		status.APIKeySet = true
	}
	status.APIProvider = cfg.LLM.Provider

	return status
}

// PrintEnvStatus は EnvStatus を標準出力に表示する。
func PrintEnvStatus(status EnvStatus) {
	fmt.Println("=== おさらいGo 環境チェック ===")
	fmt.Println()

	gitMark := "✅"
	if !status.GitInstalled {
		gitMark = "❌"
	}
	fmt.Printf("%s Git: %s\n", gitMark, status.GitVersion)

	apiMark := "✅"
	apiNote := fmt.Sprintf("(%s)", status.APIProvider)
	if !status.APIKeySet {
		apiMark = "❌"
		apiNote = "未設定 → `osarai config init` で設定してください"
	}
	fmt.Printf("%s API キー: %s\n", apiMark, apiNote)

	fmt.Printf("📁 DB パス: %s\n", status.DBPath)
}

func dbPath() string {
	home, err := homeDir()
	if err != nil {
		return "(取得失敗)"
	}
	return home + "/.osarai/data.db"
}

func homeDir() (string, error) {
	out, err := exec.Command("sh", "-c", "echo $HOME").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
