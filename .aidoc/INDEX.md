---
domain: Conventions
status: Active
entry_points: []
dependencies: []
---

# Amiglot API — Documentation Index

Discovery index for all project documentation. See the reading chains below for guided paths.

## Architecture

| Document | Description |
|----------|-------------|
| [architecture/guidelines.md](architecture/guidelines.md) | Backend architecture, coding standards, layer separation |

## Designs

| Document | Description |
|----------|-------------|
| [designs/database-schema.md](designs/database-schema.md) | Database schema — tables, constraints, migration notes |
| [designs/api-contract.md](designs/api-contract.md) | API endpoint contract, auth, validation, monitoring |
| [designs/discovery-matching.md](designs/discovery-matching.md) | Discovery endpoint design and matching rules |
| [designs/discovery-matching-query.md](designs/discovery-matching-query.md) | SQL CTE query and index strategy for matching |
| [designs/connection-handshake.md](designs/connection-handshake.md) | Connection state machine and handshake endpoints |
| [designs/in-app-messaging.md](designs/in-app-messaging.md) | Match-scoped messaging endpoints and polling strategy |

## Workflows

| Document | Description |
|----------|-------------|
| [workflows/unit-test-plan.md](workflows/unit-test-plan.md) | Unit test baseline and coverage priorities |
| [workflows/e2e-test-plan.md](workflows/e2e-test-plan.md) | End-to-end test plan with seed data and test groups |

## Cross-Repo References (amiglot-ui)

Amiglot API and UI are closely connected. The UI repo (`gnailuy/amiglot-ui`) has its own `.aidoc/` with complementary docs:

| API Doc | UI Counterpart | Relationship |
|---------|---------------|---------------|
| [API Contract](designs/api-contract.md) | [Technical Specification](https://github.com/gnailuy/amiglot-ui/blob/main/.aidoc/designs/technical-specification.md) | Shared endpoint contract — API defines the server side, UI defines the client side |
| [Discovery Matching](designs/discovery-matching.md) | [Discovery Dashboard](https://github.com/gnailuy/amiglot-ui/blob/main/.aidoc/designs/discovery-dashboard.md) | API matching rules ↔ UI dashboard that consumes them |
| [Connection Handshake](designs/connection-handshake.md) | [Connection Handshake](https://github.com/gnailuy/amiglot-ui/blob/main/.aidoc/designs/connection-handshake.md) | API state machine ↔ UI flows and components |
| [In-App Messaging](designs/in-app-messaging.md) | [In-App Messaging](https://github.com/gnailuy/amiglot-ui/blob/main/.aidoc/designs/in-app-messaging.md) | API messaging endpoints ↔ UI conversations hub and chat |
| [E2E Test Plan](workflows/e2e-test-plan.md) | [E2E Test Plan](https://github.com/gnailuy/amiglot-ui/blob/main/.aidoc/workflows/e2e-test-plan.md) | Server-side test scenarios ↔ Playwright browser tests |
| [Architecture Guidelines](architecture/guidelines.md) | [Architecture Guidelines](https://github.com/gnailuy/amiglot-ui/blob/main/.aidoc/architecture/guidelines.md) | Backend conventions ↔ Frontend conventions |
| — | [Product Definition](https://github.com/gnailuy/amiglot-ui/blob/main/.aidoc/designs/product-definition.md) | Product scope, personas, and V1 requirements (UI repo is the source of truth) |

## Reading Chains

### New Developer
1. [Architecture Guidelines](architecture/guidelines.md) — understand layers and conventions
2. [Database Schema](designs/database-schema.md) — data model
3. [API Contract](designs/api-contract.md) — endpoints and validation
4. [Unit Test Plan](workflows/unit-test-plan.md) — testing approach

### Feature Work (Discovery & Matching)
1. [Discovery Matching](designs/discovery-matching.md) — endpoint and matching rules
2. [Matching Query](designs/discovery-matching-query.md) — SQL CTE and indexes
3. [E2E Test Plan](workflows/e2e-test-plan.md) — test scenarios

### Feature Work (Connection Handshake)
1. [Connection Handshake](designs/connection-handshake.md) — state machine and endpoints
2. [Database Schema](designs/database-schema.md) — match_requests/matches/messages tables
3. [E2E Test Plan](workflows/e2e-test-plan.md) — connection test groups

### Feature Work (In-App Messaging)
1. [In-App Messaging](designs/in-app-messaging.md) — messaging endpoints and polling strategy
2. [Connection Handshake](designs/connection-handshake.md) — match creation and message re-association
3. [Database Schema](designs/database-schema.md) — messages and matches tables
4. [E2E Test Plan](workflows/e2e-test-plan.md) — messaging test scenarios

### Cross-Repo: Full-Stack Feature Understanding
1. [Product Definition (UI)](https://github.com/gnailuy/amiglot-ui/blob/main/.aidoc/designs/product-definition.md) — what Amiglot is
2. [Architecture Guidelines](architecture/guidelines.md) — API conventions
3. [Architecture Guidelines (UI)](https://github.com/gnailuy/amiglot-ui/blob/main/.aidoc/architecture/guidelines.md) — Frontend conventions
4. [API Contract](designs/api-contract.md) + [Technical Specification (UI)](https://github.com/gnailuy/amiglot-ui/blob/main/.aidoc/designs/technical-specification.md) — shared contract
5. [Discovery Matching](designs/discovery-matching.md) + [Discovery Dashboard (UI)](https://github.com/gnailuy/amiglot-ui/blob/main/.aidoc/designs/discovery-dashboard.md) — full-stack discovery
6. [Connection Handshake](designs/connection-handshake.md) + [Connection Handshake (UI)](https://github.com/gnailuy/amiglot-ui/blob/main/.aidoc/designs/connection-handshake.md) — full-stack connection
7. [In-App Messaging](designs/in-app-messaging.md) + [In-App Messaging (UI)](https://github.com/gnailuy/amiglot-ui/blob/main/.aidoc/designs/in-app-messaging.md) — full-stack messaging
