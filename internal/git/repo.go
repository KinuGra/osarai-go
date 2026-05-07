package git

// Repo はGitリポジトリの情報を保持する構造体。
type Repo struct {
	Path string // リポジトリのルートパス（.git があるディレクトリ）
	Name string // リポジトリ名（ディレクトリ名）
}

// DetectRepo はカレントディレクトリから .git を探してリポジトリを検出する。
// 見つからない場合は apperror.ErrRepoNotFound を返す。
// TODO: 機能 A の担当者が実装する。
func DetectRepo() (*Repo, error) {
	panic("not implemented")
}

