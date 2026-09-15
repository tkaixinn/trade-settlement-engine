# trade-settlement-engine

A trade settlement and reconciliation platform demonstrating idempotent event
processing, double-entry ledger accounting, and distributed systems patterns
used in trading/payments infrastructure.

## Setup

1. Start Postgres and Redis:
```
   docker compose up -d
```

2. Run the database migration:
```
   docker exec -i $(docker compose ps -q postgres) psql -U ledger -d ledger < migrations/0001_init.sql
```

3. Run the ledger service:
```
   go run cmd/ledger/main.go
```

## Architecture

- **Go ledger service** — double-entry ledger writes with idempotency
  guarantees, backed by Postgres (source of truth) and Redis (fast-path
  duplicate check cache)
- **Postgres** — accounts, ledger_entries, processed_events
- **Redis** — idempotency cache with 24h TTL