# trade-settlement-engine

A trade settlement and reconciliation platform demonstrating event-driven
architecture, price-time priority order matching, idempotent event
processing, and double-entry ledger accounting — patterns used in
trading/payments infrastructure.

## Architecture

Two independent Go services connected by Kafka:

- **Matching service** — maintains an in-memory order book and matches
  incoming orders using price-time priority (best price first, FIFO within
  a price level). Supports limit and market orders, partial fills, and
  resting orders. Order submission is serialized through a single-goroutine
  event loop (orders are sent in over a channel and processed one at a
  time), avoiding lock contention on the book. On every fill, publishes a
  fill event to Kafka. Exposes a gRPC `SubmitOrder` endpoint.
- **Ledger service** — consumes fill events from Kafka and applies them to
  Postgres as double-entry bookkeeping (every fill becomes two offsetting
  ledger entries: a debit on one account, a matching credit on the other).
  Idempotency is enforced with a two-tier check — a Redis cache for the
  fast path, and a `processed_events` table in Postgres as the source of
  truth — so a redelivered or retried Kafka message is never applied
  twice. Exposes gRPC `GetBalance` and `GetLedgerHistory` endpoints, and a
  Prometheus `/metrics` endpoint for observability.

```
┌─────────────────┐  SubmitOrder (gRPC)   ┌──────────────────┐
│     Client      │──────────────────────▶│ Matching Service │
└─────────────────┘                       └────────┬─────────┘
                                                   │ 
                                                   │ publishes fill events
                                                   ▼
                                               ┌─────────────┐
                                               │    Kafka    │
                                               │ fill-events │
                                               └──────┬──────┘
                                                      │ consumes
                                                      ▼
                                              ┌─────────────────┐
         GetBalance / GetLedgerHistory        │  Ledger Service │
         (gRPC) ◀──────────────────────────── │                 │──▶ /metrics (Prometheus)
                                              └───────┬─────────┘
                                 Redis (idempotency)  │  Postgres
                                 fast-path cache  ◀───┴──▶ (accounts,
                                                            ledger_entries,
                                                            processed_events)
```

**Order flow:** Client submits an order via gRPC to the matching service →
order is queued on a channel and matched against the resting book (price-time
priority, partial fills supported) → each resulting fill is published to the
`fill-events` Kafka topic.

**Settlement flow:** The ledger service consumes `fill-events` → for each
fill, checks Redis then Postgres to confirm the event hasn't already been
processed → if new, writes two balanced ledger entries (trader side,
clearing side) inside a single Postgres transaction and marks the event as
processed → caches the result in Redis with a 24h TTL for fast future
duplicate checks.

## Running it locally

Requires Go 1.21+ and Docker.

1. Start Postgres, Redis, and Kafka:
```
docker compose up -d
```

2. Run the database migration:
```
docker exec -i $(docker compose ps -q postgres) psql -U ledger -d ledger < migrations/0001_init.sql
```

3. Run the ledger service (gRPC server on `:50051`, Kafka consumer, metrics on `:9101`):
```
go run cmd/ledger/main.go
```

4. In a separate terminal, run the matching service (gRPC server on `:50052`):
```
go run cmd/matching/main.go
```

## Manual testing

Using [grpcurl](https://github.com/fullstorydev/grpcurl):

```
# Submit an order to the matching service
grpcurl -plaintext -import-path proto/matching -proto matching.proto \
  -d '{"account_id": "<uuid>", "side": "buy", "order_type": "limit", "price": 100.5, "quantity": 10}' \
  localhost:50052 matching.MatchingService/SubmitOrder

# Check an account's balance on the ledger service
grpcurl -plaintext -import-path proto -proto ledger.proto \
  -d '{"account_id": "<uuid>"}' \
  localhost:50051 ledger.LedgerService/GetBalance

# Full ledger history for an account
grpcurl -plaintext -import-path proto -proto ledger.proto \
  -d '{"account_id": "<uuid>"}' \
  localhost:50051 ledger.LedgerService/GetLedgerHistory
```

## Observability

The ledger service exposes Prometheus metrics on a separate port:

```
curl http://localhost:9101/metrics
```

Tracked metrics:
- `ledger_events_processed_total` — counter of fill events successfully recorded
- `ledger_events_duplicate_total` — counter of duplicate events skipped by the idempotency check
- `ledger_record_event_duration_seconds` — histogram of time spent recording each event, from idempotency check through transaction commit

No Prometheus server is configured to scrape this endpoint yet (no
`prometheus.yml` / service in `docker-compose.yml`) — the service is
instrumented and the metrics are live and correct, but there's no running
collector or dashboard on top of them yet.

## Design decisions & trade-offs

- **Single-goroutine matching, not per-order locking.** The matching engine
  serializes all order submissions through one goroutine via a channel
  (`Engine.Submit` sends a request, blocks for the result). This avoids
  lock contention and race conditions on the shared order book without
  needing explicit mutexes around the matching logic itself, at the cost
  of matching throughput being bound to a single core.
- **At-least-once Kafka delivery, idempotency handled at the consumer.**
  Rather than relying on exactly-once delivery guarantees from Kafka, the
  ledger service assumes a fill event may be delivered more than once
  (retry, consumer restart, rebalance) and makes duplicate application
  impossible via the Redis + Postgres idempotency check, keyed on a unique
  event ID generated at publish time.
- **Services are decoupled via Kafka, not direct gRPC calls between them.**
  The matching service has no knowledge of the ledger service; it only
  publishes to a topic. This means settlement can lag behind matching
  (eventual consistency) and the ledger service can be restarted or
  rebuilt without the matching service needing to know.
- **No persistence in the matching service.** The order book and matching
  state live in memory only; a restart loses all resting orders. Only the
  ledger (the system of record for settled trades) is durable, in
  Postgres.
- **No distributed transaction across matching and settlement.** A fill
  that's generated but fails to publish to Kafka (e.g. broker unreachable)
  is currently just logged as an error and the fill is lost — there's no
  outbox pattern or retry queue yet. Acceptable for this project's current
  scope, but a production system would want at-least-once delivery
  guaranteed from the matching side too, not just idempotency on the
  consuming side.

## Tech stack

Go, gRPC, Apache Kafka (KRaft mode, via `segmentio/kafka-go`), PostgreSQL
(via `pgx`), Redis (via `go-redis`), Prometheus (`client_golang`), Docker
Compose.