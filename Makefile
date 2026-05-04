.PHONY: build test lint        # ファイル名と衝突しないように宣言
                                # .PHONY がないと「test」というファイルがある時に
                                # make test が「すでに最新」と判断してスキップされる

build:                          # make build → バイナリを生成
	go build -o osarai ./main.go
	#  -o osarai : 出力ファイル名を「osarai」に指定
	#  ./main.go : エントリポイント

test:                           # make test → テスト実行
	go test ./...
	#  ./... : 全パッケージを再帰的にテスト

lint:                           # make lint → リンター実行
	golangci-lint run
	#  .golangci-lint.yml の設定に従ってチェック
