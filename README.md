# Amiglot API

Backend service for Amiglot — find language learning partners.

## Stack
- HTTP framework: [Huma](https://github.com/danielgtaylor/huma)
- DB driver: [pgx](https://github.com/jackc/pgx)
- Query layer: [sqlc](https://sqlc.dev)
- Migrations: [goose](https://github.com/pressly/goose)

## Environment
Copy `.env.example` to `.env.local` and adjust as needed:

```bash
cp .env.example .env.local
```

## Run locally

```bash
go run ./cmd/server
```

Health check:

```bash
curl http://localhost:6174/healthz
```

## Database

Generate sqlc code:

```bash
make sqlc
```

Run migrations:

```bash
make migrate-up
```
