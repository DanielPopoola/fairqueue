# FairQueue — Design Decisions & Tradeoffs

---

## Language: Go

Go's goroutines handle 20,000 long-lived WebSocket connections cheaply — the
defining traffic shape for a queue system. Python was abandoned because the original
codebase had the wrong mental model (Redis as source of truth), and porting wrong
thinking produces wrong thinking in a new language. Java was rejected because the
Spring/JVM ecosystem overhead costs more in solo-engineer time than the language
switch gains. Go produces a single compiled binary with no runtime dependencies.
**Accepted cost:** more verbose than Python. Smaller talent pool in the Nigerian
market if hiring becomes necessary.

---

## PostgreSQL as Single Source of Truth

Postgres owns all authoritative state. Redis holds nothing that cannot be
reconstructed from Postgres. The Python implementation treated Redis as a co-equal
store, creating an unsolvable dual-write problem — no atomic operation spans two
storage systems. The rule: Redis makes things faster, it cannot make things correct.
**Accepted cost:** on Redis failure, the system falls back to Postgres-only reads,
which will struggle under high load. Known degradation path, not a surprise.

---

## Inventory Count: Derived, Not Stored

No `available_count` column on events. Available inventory is always
`total_inventory - COUNT(active claims)`. Storing a count column alongside claim
rows creates two sources of truth inside Postgres itself — the dual-write problem
reproduced in a single database. This query is never on the hot path; it runs only
in the reconciliation worker (every 30s) and on Redis cache miss. An index on
`(event_id, status)` keeps it fast enough.

---

## Two-Layer Concurrency Shield

Concurrent claims are controlled at two layers. Layer 1 (Redis): `SET NX` lock
turns away concurrent attempts before they reach the database. Layer 2 (Postgres):
unique constraint on `(event_id, customer_id)` is the inviolable correctness
guarantee. Redis is a cheap doorman that reduces Postgres contention; Postgres is
the last line of defense. If Redis is unavailable, correctness is preserved but
Postgres contention increases significantly under load.

---

## Redis Cache Updated After Postgres Commit

The Redis inventory count is decremented only after a Postgres INSERT commits. Never
speculatively. If the server crashes after commit but before the Redis write, Redis
shows more tickets than exist (inflation) — healed within 30s by the reconciliation
worker. The alternative ordering (Redis first) risks deflation, which incorrectly
turns valid users away. Temporary inflation is the only acceptable failure mode.

---

## Outbox Pattern for Payment Safety

Payment records are written to Postgres in `INITIALIZING` state before Paystack is
called. Without this, a crash after Paystack confirms but before Postgres writes
produces a charge with no record. With it, the reconciliation worker finds any
`INITIALIZING` or `PENDING` record older than a threshold and polls Paystack to
determine its real state. The outbox is what makes recovery possible regardless of
when a crash occurs. **Gap surfaced by tests:** `MarkFailed` must accept
`expectedStatus` as a parameter — the permanent-error path hits the payment while
still `INITIALIZING`, not `PENDING`. Passing expected status keeps the conditional
update pattern consistent and avoids a silent no-op.

---

## Idempotency Guard on Payment Initialize

Two layers protect against duplicate payments for the same claim. The
`getExistingPayment` guard covers the sequential case (user navigates back and
clicks Pay again) by short-circuiting before the gateway call. A unique index on
`payments(claim_id)` covers the concurrent case — under a race, the second INSERT
fails with a unique violation, and `Initialize` fetches and returns the existing
record rather than erroring. **Gap surfaced by tests:** without the unique index,
five concurrent `Initialize` calls all pass the guard and all hit the gateway. Both
layers are necessary.

---

## Virtual Queue: Redis ZSET over Kafka

The waiting queue is a Redis Sorted Set with join timestamp as score. Kafka would
be justified if ingestion rate exceeded what a single Redis instance handles or if
the queue needed replay across independent consumers. At the target scale of 50,000
concurrent users per event, Redis ZSET is sufficient — O(log N) insert and rank
lookup, simple to operate. Queue entries are also written to Postgres so state can
be reconstructed after a Redis restart. Queue position is a UX feature; losing it
is recoverable. Losing payment state is not.

---

## Project Structure: Layered without Ports/Adapters

Three layers: `store → service → api`. No hexagonal architecture. With exactly one
database, one cache, and one payment provider, every interface would have exactly
one implementation — ceremony without benefit. Testcontainer-based integration tests
make infrastructure mocking unnecessary. A new reader understands the system by
reading three files, not nine. If genuine infrastructure swapping becomes necessary,
interfaces are added at that point with real requirements, not speculated in advance.

---

## Testing Strategy

Domain layer: unit tests (pure logic, no infrastructure). Service and worker layers:
integration tests against real Postgres and Redis via testcontainers. API layer: E2E
tests. The only mock in the codebase is the Paystack gateway — because real HTTP
calls have no place in a test suite and the interface is under our control. Mocking
the database tests that you called a mock correctly, not that the system works.

---

## Explicitly Rejected

| Decision | Reason |
|---|---|
| Kafka for queue | Operational overhead unjustified at target scale |
| Redis Cluster | Single instance handles target scale |
| Separate WebSocket server | Go goroutines handle it on the same server |
| `SELECT FOR UPDATE` | Redis lock shields DB more efficiently; unique constraint handles correctness |
| Kubernetes | Docker Compose on a single VPS is sufficient for a solo operator |
| Hexagonal architecture | One implementation per interface; testcontainers eliminate the need |

---

## Known Gaps

- **Redis restart recovery:** Admission worker doesn't handle `ACTIVE` events with
  wiped Redis counters. Startup recovery worker needed.
- **Worker deduplication:** Multiple server instances run duplicate workers. Leader
  election via Redis SETNX or Postgres advisory locks needed before horizontal scaling.
- **Bot prevention:** No CAPTCHA or IP rate limiting. The queue provides natural
  protection but determined operators can still abuse it.
- **Seat selection:** General admission only. Specific seats require a different
  inventory model.
- **Multi-tenancy isolation:** Key namespacing provides logical isolation only.
Under concurrent high-demand events, a single Redis instance creates resource
contention — one large event can saturate the command queue and degrade all
others. The fix when needed is Redis Cluster with hash tags, which is a key
format change, not an application change. Current key format uses event_id
as prefix and is already compatible with this migration.