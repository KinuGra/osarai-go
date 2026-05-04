# Git Development Rules

本プロジェクトでは **Issueドリブン開発 + シンプルなGit Flow** を採用する。

---

# 1. Development Flow（Issueドリブン開発）

すべての作業は **Issue を起点に開始する。**

## フロー

1. Issue を作成（Project の `+ Add item` → `Create new issue`）
2. 必要に応じて Sub Issue を作成
3. 自分を Assignee に設定
4. 作業開始時に `In Progress` に変更
5. Issue の右サイドバー **Development** → **`Create a branch`** からブランチを作成
   - Branch name に命名規則に沿った名前を入力（例: `feat/18-add-login-api`）
   - source branch が `main` であることを確認
6. 表示されるコマンドでローカルに checkout して作業
   ```bash
   git fetch origin
   git checkout <branch-name>
   ```
7. main に対して Pull Request を作成
8. レビュー後に main に merge

## ドキュメントについて

- 実装の方針・メモ・仕様の補足などは **Issue のコメントや本文に残しておくと良い**
- ちゃんとしたドキュメントが必要な場合は `docs/` にまとめる

---

# 2. Git Flow

本プロジェクトでは **シンプルな Git Flow** を採用する。

## 基本ブランチ

```
main : 本番 / 最新安定コード
```

### ルール

- **すべての作業ブランチは main から作成**
- 作業完了後 **main に Pull Request を作成**
- レビュー後 main に merge

---

# 3. Branch Naming Rules（ブランチ命名規則）

```
<prefix>/<issue-number>-<description>
```

例

```
feat/18-add-login-api
fix/28-auth-error
refactor/35-clean-user-service
docs/40-update-readme
```

---

# 4. Commit Message Rules

フォーマット

```
<prefix>: <message>
```

例（英語）

```
feat: add login api
fix: auth middleware bug
refactor: clean user service
docs: update readme
```

例（日本語）

```
feat: ログインAPIを追加
fix: 認証ミドルウェアのバグを修正
refactor: ユーザーサービスを整理
docs: READMEを更新
```

---

## 作業途中コミット

作業途中の場合は `wip` を使用

```
feat: (wip) implement login api
feat: (wip) ログインAPIを実装中
```

---

## Issue番号を含める場合

```
fix: (#28) auth middleware bug
feat: (#18) add login api
fix: (#28) 認証ミドルウェアのバグを修正
feat: (#18) ログインAPIを追加
```

Issue番号の記載は任意。

---

# 5. Sub Issue の切り方

1つの Issue が大きすぎる場合や、複数の作業に分割できる場合は **Sub Issue を作成する。**

## 切るタイミングの目安

- 1つの Issue に複数の独立した作業が含まれる場合
- 別の人が並行して作業できる単位に分けられる場合
- PR が大きくなりすぎると感じた場合

## Sub Issue の作り方

親 Issue に対して GitHub の Sub Issues 機能で紐付ける。タイトルは親 Issue の内容が伝わるように書く。

例：親 Issue「ユーザー認証機能を実装する」に対して

```
ログインAPIを実装する
ログアウトAPIを実装する
認証ミドルウェアを追加する
```

各 Sub Issue に対して個別にブランチを切り、PR を作成する。

---

# 6. Issue Title Rules（Issue タイトル規則）

Issue のタイトルは **日本語、または分かりやすい英語** で記載する。

- 何をするのか・何が問題なのかが一目で分かるように書く
- 曖昧なタイトル（例: `fix bug`、`作業`）は避ける

例

```
ログインAPIを実装する
認証ミドルウェアでエラーが発生する
ユーザー一覧画面のレイアウトを修正する
Add login API
Fix auth middleware error
```

---

# 6. Pull Request

## 基本ルール

- **main に対して作成**
- レビュー後に merge

## PR タイトル

**日本語、または分かりやすい英語** で記載する。対応する Issue の内容が伝わるタイトルにする。

例

```
ログインAPIを追加
認証ミドルウェアのバグを修正
Add login API
Fix auth middleware error
```

## 関連 Issue の紐付け（必須）

PR の本文に以下のキーワードを記載することで、merge 時に Issue が自動でクローズされる。

```
Closes #<issue番号>
```

例

```
Closes #18
Closes #28
```

複数 Issue に対応する場合

```
Closes #18
Closes #35
```

---

# 6. Summary

開発ルールまとめ

- Issueドリブン開発
- main ベースのブランチ運用
- Issue番号付きブランチ
- prefix付きコミットメッセージ
- PRでレビューしてからmainへmerge
