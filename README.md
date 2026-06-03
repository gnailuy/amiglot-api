# Amiglot API

Backend service for Amiglot — a site to find language learning partners.

## Features

- Magic-link authentication (dev mode: local link generation)
- User profiles with languages, levels, availability, and handles
- Discovery and matching with language-pair scoring
- Connection requests with pre-accept messaging
- In-app messaging between matched partners
- Internationalization (11 locales)

## Documentation

Project documentation lives in `.aidoc/` and follows AI-native conventions. Start with [`.aidoc/INDEX.md`](.aidoc/INDEX.md) for architecture, designs, and workflows.

The frontend lives in a separate repo: [amiglot-ui](https://github.com/gnailuy/amiglot-ui). Both repos cross-reference each other's `.aidoc/` docs.

## Stack

- Go 1.24, [Huma](https://github.com/danielgtaylor/huma) (HTTP framework)
- PostgreSQL with [pgx](https://github.com/jackc/pgx) + [sqlc](https://sqlc.dev)
- Migrations: [goose](https://github.com/pressly/goose)

## Setup

```bash
cp .env.example .env.local
```

Key variables: `PORT` (default `6176`), `DATABASE_URL`, `ENV` (`dev` for magic-link dev mode), `MAGIC_LINK_BASE_URL`.

## Database

```bash
docker network create amiglot-dev-net

docker run -d --name amiglot-dev-db --rm \
  --network amiglot-dev-net \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_DB=amiglot_dev \
  -p 5432:5432 \
  postgres:16

make migrate-up
```

## Build & Run

```bash
# Local
go build -o bin/amiglot-api ./cmd/server
make run

# Docker
docker build -t amiglot-api:dev .
docker run --rm -d --name amiglot-dev-api \
  --network amiglot-dev-net \
  -p 6176:6176 \
  --env-file .env.local \
  amiglot-api:dev
```

Health check: `curl http://localhost:6176/api/v1/healthz`

## Tests

```bash
make test
```

CI enforces minimum 80% coverage.
