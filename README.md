# FairQueue

A virtual queue and inventory allocation system for high-demand live events in Nigeria. Built to handle the moment 50,000 people try to buy 5,000 tickets at exactly the same time — without overselling, without crashes, and without bots.

## The Problem

When a high-demand event goes on sale, three things happen simultaneously:

- The website crashes under the sudden spike in traffic
- Bots grab the inventory in milliseconds before real fans get a chance
- Payment failures silently lose tickets that were already claimed

FairQueue solves all three. It absorbs the traffic spike into a virtual queue, admits customers at a controlled rate, and guarantees that inventory allocation is atomic — two people can never get the same ticket.

## Quick Start

```bash
git clone https://github.com/DanielPopoola/fairqueue
cd fairqueue
cp .env.example .env    # fill in your Paystack keys; everything else works out of the box
docker compose up --build
```

API is available at `http://localhost:8080`. Swagger UI is at `http://localhost:8080/swagger/index.html`.

## How It Works

A customer's journey through the system has four stages:

**1. Queue** — When the sale opens, everyone hits `POST /events/{id}/queue` at once. This is a cheap Redis write (O(log N)), so it absorbs any traffic volume without touching the database. Each customer gets a queue position.

**2. Admission** — A background worker runs every 5 seconds. It pops the next batch of customers from the waiting queue, moves them to the admitted set, and pushes them a signed admission token via WebSocket. Customers who miss the push can poll `GET /events/{id}/queue/position` to retrieve their token.

**3. Claim** — An admitted customer presents their token to `POST /events/{id}/claims`. The system atomically checks and decrements the Redis inventory counter. If that succeeds, it inserts a claim row in Postgres. A unique constraint on `(event_id, customer_id)` is the last line of defence against any race condition.

**4. Payment** — The customer calls `POST /claims/{id}/payments`, which writes a payment record to Postgres *before* calling Paystack. This means a crash can never produce a charge with no record. The reconciliation worker finds and heals any payments stuck in intermediate states.

## Key Design Decisions

**PostgreSQL is the only source of truth.** Redis holds nothing that cannot be reconstructed from Postgres. If Redis is wiped, the startup recovery function rebuilds the queue and the reconciliation worker heals the inventory count. Redis makes things fast; Postgres makes things correct.

**Two-layer concurrency shield.** A Redis `SET NX` lock stops concurrent claim attempts before they reach the database. A Postgres unique constraint on `(event_id, customer_id)` is the inviolable correctness guarantee that holds even if the lock is unavailable. Both layers must fail for an oversell to occur.

**Outbox pattern for payment safety.** A `Payment` row is always written in `INITIALIZING` state before the Paystack API is called. A crash at any point leaves a recoverable record. The reconciliation worker finds stale `INITIALIZING` records and retries the gateway call.

**Postgres-first writes.** The Redis inventory counter is only decremented *after* the Postgres insert commits. If the server crashes between the commit and the Redis write, Redis shows more tickets than exist — the reconciliation worker heals this within 30 seconds. The alternative (Redis first) risks permanently locking out valid users. Temporary inflation is the only acceptable failure mode.

For the full reasoning behind every decision, see [TRADEOFFS.md](./TRADEOFFS.md).

## Architecture

See [ARCHITECTURE.md](./ARCHITECTURE.md) for a component diagram showing how all the pieces fit together.

## Project Structure

```
cmd/api/            Entry point and dependency wiring
internal/
  domain/           State machines and domain errors — no infrastructure dependencies
  service/          Business logic: claims, queue, payments, events
  store/
    postgres/       PostgreSQL store implementations
    redis/          Redis store implementations (inventory, queue, lock)
  worker/           Background workers: admission, expiry, reconciliation, recovery
  api/              HTTP handlers, middleware, WebSocket hub
  gateway/paystack/ Paystack payment gateway adapter
  auth/             JWT tokenizers (organizer + customer), argon2id password hashing
  config/           Environment-based configuration loading and validation
  infra/
    migrate/        Embedded SQL migrations
    retry/          Generic retry with exponential backoff
```

## Running Tests

```bash
# Domain logic only — fast, no infrastructure required
make test-unit

# Service and worker tests — spins up real Postgres and Redis via testcontainers
make test-integration

# Full end-to-end flow tests
make test-e2e

# Everything
make test
```

Integration tests run against real infrastructure. The only mock in the codebase is the Paystack gateway — because mocking the database tells you nothing about whether the unique constraint fires.

## Stack

| Layer | Technology |
|---|---|
| Language | Go |
| Database | PostgreSQL 16 |
| Cache / Queue | Redis 7 |
| Payment | Paystack |
| HTTP router | Chi |
| WebSocket | coder/websocket |
| Container | Docker + Compose |