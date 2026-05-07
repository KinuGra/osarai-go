package git

// DiffScope は差分の取得範囲を表す。
type DiffScope string

const (
	DiffAll    DiffScope = "all"    // 全ての変更（git diff HEAD）
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
// TODO: 機能 A の担当者が実装する。
func GetDiff(repo *Repo, scope DiffScope, commitHash string) (*DiffResult, error) {
	panic("not implemented")
}
