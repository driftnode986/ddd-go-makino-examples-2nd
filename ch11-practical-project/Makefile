.PHONY: up down reset seed run test test-integration lint

up:
	docker compose up -d --wait

down:
	docker compose down -v

# 会員とレッスンの初期データを入れる。何度流してもよい。
seed:
	docker compose exec -T postgres psql -v ON_ERROR_STOP=1 \
		-U dev -d trainingbooking < scripts/seed.sql

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
