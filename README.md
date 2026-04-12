# FairQueue

A high-throughput inventory allocation system for high-demand live events in Nigeria. Handles the thundering herd problem — when 50,000 people try to buy 5,000 tickets at the same time — without overselling, without crashes, and without bots taking everything before real fans get a chance.

## The Problem

When a high-demand event goes on sale:

- Websites crash under the simultaneous load
- Bots claim inventory in milliseconds
- Payment failures silently lose claimed tickets
- Users have no idea where they stand

FairQueue solves this with a virtual queue that absorbs the spike, admits customers at a controlled rate, and guarantees atomic inventory allocation backed by real distributed systems correctness.

## Quick Start

```bash
git clone https://github.com/DanielPopoola/fairqueue
cd fairqueue
cp .env.example .env          # fill in Paystack keys; defaults work for local dev
docker compose up --build
```

The API is live at `http://localhost:8080`.
Swagger UI is at `http://localhost:8080/swagger/index.html`.

## How It Works

```
50,000 users hit the sale at 10:00:00 AM
         │
         ▼
  Virtual Queue (Redis ZSET)
  Assigns position instantly — cheap O(log N) operation
         │
         ▼ (admission worker, every 5s)
  Admitted in batches sized to available inventory
  Each admitted user receives a short-lived signed token
         │
         ▼
  Claim (atomic Redis Lua script + Postgres)
  Token verified → inventory decremented → claim created
         │
         ▼
  Payment (Paystack)
  Outbox record written before gateway call
  Webhook confirms → claim confirmed
  Failure → claim released → inventory restored
```

The queue absorbs the thundering herd. Only admitted users reach the claim layer. Only claimed users reach the payment layer. Each transition is a rate-limiting gate.

## Key Design Decisions

**PostgreSQL is the source of truth.** Redis holds no state that cannot be reconstructed from Postgres. If Redis is wiped, the reconciliation worker heals inventory counts and the recovery function rebuilds the queue ZSETs from Postgres on the next startup. See [TRADEOFFS.md](./TRADEOFFS.md) for the full reasoning.

**Two-layer concurrency shield.** A Redis `SET NX` lock prevents concurrent claim attempts from reaching the database simultaneously. A Postgres unique constraint on `(event_id, customer_id)` is the inviolable last line of defence. The lock is a performance optimisation; the constraint is the correctness guarantee.

**Outbox pattern for payment safety.** A `Payment` row is written in `INITIALIZING` state before the Paystack API is called. A crash between the write and the gateway call leaves a recoverable record. The reconciliation worker finds stale `INITIALIZING` records and retries the gateway call.

**100 goroutines, 1 ticket, exactly 1 claim.** The concurrency guarantee is a tested invariant, not just a design intent. See `internal/service/claims_test.go`.

## Architecture

See [ARCHITECTURE.md](./ARCHITECTURE.md) for component diagrams and flow diagrams.

## Project Structure

```
cmd/api/          Entry point, dependency wiring
internal/
  domain/         State machines and business rules (no infrastructure)
  service/        Business logic orchestration
  store/
    postgres/     PostgreSQL store implementations
    redis/        Redis store implementations
  worker/         Background workers (admission, expiry, reconciliation)
  api/            HTTP handlers, middleware, WebSocket hub
  gateway/        Paystack payment gateway
  auth/           JWT tokenizers, argon2id password hashing
  config/         Environment-based configuration
  infra/
    migrate/      Embedded SQL migrations
    retry/        Generic retry with exponential backoff
```

## Testing

```bash
# Unit tests — domain logic, no infrastructure
make test-unit

# Integration tests — real Postgres + Redis via testcontainers
make test-integration

# All tests
make test
```

Integration tests run against real infrastructure using testcontainers. No mocks except the Paystack gateway — because mocking the database tells you nothing about whether the unique constraint fires.

The test suite caught two production bugs during development:

- `MarkFailed` silently no-oping because the payment was still `INITIALIZING` not `PENDING` when a permanent gateway error occurred
- Missing unique index on `payments(claim_id)` allowing duplicate gateway calls under concurrency

## Stack

| Layer | Technology |
|---|---|
| Language | Go |
| Database | PostgreSQL 16 |
| Cache / Queue | Redis 7 |
| Payment | Paystack |
| Container | Docker + Compose |
| HTTP | chi |
| WebSocket | coder/websocket |