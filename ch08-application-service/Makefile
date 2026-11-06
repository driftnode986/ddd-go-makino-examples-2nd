.PHONY: up down test test-integration lint

up:
	docker compose up -d --wait

down:
	docker compose down -v

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
