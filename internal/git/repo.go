package git

import (
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/KinuGra/osarai-go/internal/apperror"
)

// Repo は Git リポジトリの情報を保持する構造体。
type Repo struct {
	Path string // リポジトリのルートパス（.git があるディレクトリ）
	Name string // リポジトリ名（ディレクトリ名）
}

// DetectRepo はカレントディレクトリから .git を探してリポジトリを検出する。
// git が未インストールなら ErrGitNotFound、.git が見つからなければ ErrRepoNotFound を返す。
func DetectRepo() (*Repo, error) {
	if _, err := exec.LookPath("git"); err != nil {
		return nil, apperror.ErrGitNotFound
	}

	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return nil, apperror.ErrRepoNotFound
	}

	path := strings.TrimSpace(string(out))
	name := filepath.Base(path)
	return &Repo{Path: path, Name: name}, nil
}
