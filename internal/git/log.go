package git

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// CommitInfo はコミット履歴の 1 エントリ。
type CommitInfo struct {
	Hash        string
	Message     string
	AuthorName  string
	AuthorEmail string
	CommittedAt time.Time
}

// GetAuthorEmail はリポジトリの git config user.email を返す。
func GetAuthorEmail(repo *Repo) (string, error) {
	out, err := exec.Command("git", "-C", repo.Path, "config", "user.email").Output()
	if err != nil {
		return "", fmt.Errorf("git config user.email の取得に失敗: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// GetCommitLog は指定した author のコミット履歴を最新 limit 件取得する。
// authorEmail が空の場合は全 author のコミットを取得する。
func GetCommitLog(repo *Repo, authorEmail string, limit int) ([]CommitInfo, error) {
	// フォーマット: %H (ハッシュ), %s (メッセージ), %an (author名), %ae (authorメール), %aI (ISO8601日時)
	// 各フィールドをユニットセパレータで区切り、コミット間は改行で区切る
	format := "%H\x1f%s\x1f%an\x1f%ae\x1f%aI"

	args := []string{
		"-C", repo.Path,
		"log",
		fmt.Sprintf("--max-count=%d", limit),
		fmt.Sprintf("--format=%s", format),
	}
	if authorEmail != "" {
		args = append(args, fmt.Sprintf("--author=%s", authorEmail))
	}

	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("git log の取得に失敗: %w", err)
	}

	body := strings.TrimSpace(string(out))
	if body == "" {
		return nil, nil
	}

	var commits []CommitInfo
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\x1f", 5)
		if len(parts) != 5 {
			continue
		}
		t, err := time.Parse(time.RFC3339, parts[4])
		if err != nil {
			continue
		}
		commits = append(commits, CommitInfo{
			Hash:        parts[0],
			Message:     parts[1],
			AuthorName:  parts[2],
			AuthorEmail: parts[3],
			CommittedAt: t.UTC(),
		})
	}
	return commits, nil
}
