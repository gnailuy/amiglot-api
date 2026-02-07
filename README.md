# Amiglot API

Backend service for Amiglot — find language learning partners.

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
curl http://localhost:8080/healthz
```
