# AGENT.md — Amiglot API

Instructions for AI agents working on this repo.

## Documentation

Read `.aidoc/INDEX.md` first. It has a discovery table and reading chains — follow the chain that matches your task.

The frontend repo ([amiglot-ui](https://github.com/gnailuy/amiglot-ui)) has its own `.aidoc/` with complementary docs. Cross-repo references in `.aidoc/INDEX.md` link to the counterpart docs.

## Development workflow

- Branch from `main`: `feature/<short>` for features, `fix/<short>` for fixes.
- PRs target `main`. Include both docs and implementation in the same PR.
- CI must pass before merge: `go test` (80% coverage minimum), `golangci-lint`.

## Documentation as development

When adding or changing features, update `.aidoc/` docs as part of the same PR. Follow the doc-manager skill and DocGuidelines:

- Docs capture the *why* and *constraints*. Code is the *how*.
- Create or update the relevant design doc in `.aidoc/designs/`.
- Update `.aidoc/INDEX.md`: add table entries and update reading chains.
- Keep docs ~100 lines. Split by subdomain rather than writing monoliths.

## Domain placeholder

Use `example.com` as the placeholder domain in all code, docs, and tests.

## Internationalization

All user-facing strings must be translated. English is the absolute fallback. Error messages use structured codes with localized text via `locales/*.json`.
