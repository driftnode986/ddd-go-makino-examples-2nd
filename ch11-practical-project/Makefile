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
