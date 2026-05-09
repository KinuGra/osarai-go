package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// DiffScope は差分の取得範囲を表す。
type DiffScope string

const (
	DiffAll    DiffScope = "all"    // 全ての変更（staged + unstaged、git diff HEAD）
	DiffStaged DiffScope = "staged" // ステージ済みのみ（git diff --staged）
	DiffCommit DiffScope = "commit" // 特定コミットの差分（git show <hash>）
)

// DiffResult は diff の取得結果。
type DiffResult struct {
	Scope DiffScope
	Body  string // diff の内容
	Hash  string // commit scope の場合のコミットハッシュ
}

// GetDiff は指定スコープの diff を取得する。
// scope が DiffCommit の場合は commitHash を指定する。
// 差分が空の場合は errNoDiff を返す。
func GetDiff(repo *Repo, scope DiffScope, commitHash string) (*DiffResult, error) {
	var args []string
	switch scope {
	case DiffAll:
		// HEAD との差分（staged + unstaged を両方含む）
		args = []string{"diff", "HEAD"}
	case DiffStaged:
		args = []string{"diff", "--staged"}
	case DiffCommit:
		if commitHash == "" {
			return nil, fmt.Errorf("コミットハッシュが指定されていません")
		}
		// git show でコミット単体の diff を取得
		args = []string{"show", commitHash}
	default:
		return nil, fmt.Errorf("不明なスコープ: %s", scope)
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = repo.Path
	out, err := cmd.Output()
	if err != nil {
		// git diff は差分ゼロでも exit 0 なので、ここに来るのは本当のエラー
		return nil, fmt.Errorf("git diff 取得失敗: %w", err)
	}

	body := string(out)
	if strings.TrimSpace(body) == "" {
		return nil, fmt.Errorf("差分がありません（%s）。変更を加えてから実行してください", scope)
	}

	return &DiffResult{Scope: scope, Body: body, Hash: commitHash}, nil
}
