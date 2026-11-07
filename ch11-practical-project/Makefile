.PHONY: up down reset seed run test test-integration lint

up:
	docker compose up -d --wait

down:
	docker compose down -v

# 会員とレッスンの初期データを入れる。何度実行してもよい。
# ドメインの検証を通すので、SQL でなく Go のコマンドで入れる。
seed:
	go run ./cmd/seed

# データを消して最初からやり直す。
reset: down up seed

run:
	go run ./cmd/api

test:
	go test ./...

test-integration:
	go test -tags=integration ./...

lint:
	golangci-lint run ./...

# sqlc の版を固定して生成する。初回はビルドに数分かかる。
.PHONY: sqlc
sqlc:
	go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate
