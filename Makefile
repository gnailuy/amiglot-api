.PHONY: run tidy sqlc migrate-up migrate-down fmt lint test

run:
	go run ./cmd/server

tidy:
	go mod tidy

fmt:
	gofmt -w .
	goimports -w .

lint:
	golangci-lint run

test:
	go test ./...

sqlc:
	sqlc generate

migrate-up:
	goose -dir db/migrations postgres "$$DATABASE_URL" up

migrate-down:
	goose -dir db/migrations postgres "$$DATABASE_URL" down
