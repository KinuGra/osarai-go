# osarai-go

AI生成コードの"わかったつもり"をなくし、知識として定着させるターミナルTUIアプリ

## 前提条件

- Go 1.25+
- Git

## セットアップ

```

git clone https://github.com/KinuGra/osarai-go.git

cd osarai-go

go mod tidy

```

## 開発コマンド

```

# ビルド

make build

# テスト

make test

# リンター（要インストール）

brew install golangci-lint

make lint

```

## ディレクトリ構成

```markdown
osarai-go/

├── main.go

├── cmd/ # コマンド層（Cobra）

├── internal/

│ ├── core/ # ビジネスロジック

│ ├── ai/ # LLMプロバイダー

│ ├── git/ # Git操作

│ ├── db/ # SQLite

│ ├── tui/ # Bubble Tea

│ ├── config/ # 設定管理

│ └── apperror/ # エラー型

├── Makefile

└── go.mod
```
