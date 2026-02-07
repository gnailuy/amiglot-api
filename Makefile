.PHONY: run tidy sqlc migrate-up migrate-down

run:
	go run ./cmd/server

tidy:
	go mod tidy

sqlc:
	sqlc generate

migrate-up:
	goose -dir db/migrations postgres "$$DATABASE_URL" up

migrate-down:
	goose -dir db/migrations postgres "$$DATABASE_URL" down
