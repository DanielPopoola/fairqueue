# Architecture

## Component Overview

```mermaid
graph TD
    Client["Client<br/>(Browser / App)"]
    API["API Layer<br/>chi router + handlers"]
    WS["WebSocket Hub<br/>live position updates"]
    Auth["Auth<br/>JWT + argon2id OTP"]

    CS["ClaimService"]
    QS["QueueService"]
    PS["PaymentService"]
    ES["EventService"]

    AW["Admission Worker<br/>every 5s"]
    EW["Expiry Worker<br/>every 30s"]
    RW["Reconciliation Worker<br/>every 30s"]

    PG[("PostgreSQL<br/>source of truth")]
    RD[("Redis<br/>performance layer")]
    GW["Paystack Gateway"]

    Client -->|HTTP| API
    Client -->|WS| WS
    API --> Auth
    API --> CS
    API --> QS
    API --> PS
    API --> ES

    AW -->|admit batch| QS
    AW -->|push token| WS
    EW -->|release claims| CS
    RW -->|heal divergence| RD
    RW -->|reconcile payments| PS

    CS --> PG
    CS --> RD
    QS --> PG
    QS --> RD
    PS --> PG
    PS --> GW
    ES --> PG
```

## Request Flow: Customer Buys a Ticket

```mermaid
sequenceDiagram
    actor Customer
    participant API
    participant Queue as Queue Service
    participant Redis
    participant Postgres
    participant Admission as Admission Worker
    participant Claim as Claim Service
    participant Payment as Payment Service
    participant Paystack

    Customer->>API: POST /events/{id}/queue
    API->>Queue: Join(customerID, eventID)
    Queue->>Postgres: INSERT queue_entry (WAITING)
    Queue->>Redis: ZADD waiting:{eventID} score=joinedAt
    API-->>Customer: 201 { position: 1547 }

    Note over Admission: Every 5 seconds
    Admission->>Redis: ZPOPMIN waiting:{eventID} batch
    Admission->>Redis: ZADD admitted:{eventID}
    Admission->>Postgres: UPDATE queue_entries SET status=ADMITTED
    Admission->>Customer: WebSocket push { type: admitted, token: ... }

    Customer->>API: POST /events/{id}/claims { admission_token }
    API->>Claim: Claim(token, eventID)
    Claim->>Redis: SET NX lock (concurrency shield layer 1)
    Claim->>Redis: DECRBY inventory (atomic Lua script)
    Claim->>Postgres: INSERT claims (unique constraint = layer 2)
    API-->>Customer: 201 { claim_id, expires_at }

    Customer->>API: POST /claims/{id}/payments
    API->>Payment: Initialize(claimID, customerID)
    Payment->>Postgres: INSERT payments (status=INITIALIZING)
    Payment->>Paystack: InitializeTransaction
    Payment->>Postgres: UPDATE payments (status=PENDING)
    API-->>Customer: 201 { authorization_url }

    Customer->>Paystack: Complete payment on hosted page
    Paystack->>API: POST /webhooks/paystack (charge.success)
    API->>Payment: HandleWebhook(rawBody, signature)
    Payment->>Postgres: UPDATE payments (PENDING→CONFIRMED)
    Payment->>Postgres: UPDATE claims (CLAIMED→CONFIRMED)
    API-->>Paystack: 200 OK
```

## Inventory Consistency Model

Redis is a performance layer over Postgres. It is never the source of truth.

```mermaid
flowchart LR
    subgraph Write Path
        direction TB
        W1["Postgres INSERT claim<br/>(atomic, durable)"]
        W2["Redis DECRBY inventory<br/>(fast, best-effort)"]
        W1 --> W2
    end

    subgraph Failure Recovery
        direction TB
        F1["Redis wiped / crash"]
        F2["Reconciliation Worker<br/>derives count from Postgres<br/>every 30s"]
        F3["Startup Recovery<br/>rebuilds ZSETs from Postgres"]
        F1 --> F2
        F1 --> F3
    end

    subgraph Read Path
        direction TB
        R1["Redis read (fast)"]
        R2{"cache miss?"}
        R3["Postgres read (fallback)"]
        R1 --> R2
        R2 -->|yes| R3
    end
```

## Payment State Machine

```mermaid
stateDiagram-v2
    [*] --> INITIALIZING: INSERT before gateway call
    INITIALIZING --> PENDING: Paystack responds OK
    INITIALIZING --> FAILED: Permanent gateway error
    PENDING --> CONFIRMED: charge.success webhook
    PENDING --> FAILED: charge.failed webhook
    CONFIRMED --> [*]
    FAILED --> [*]

    note right of INITIALIZING
        Outbox record exists before
        any external call is made.
        Crash here = reconciliation
        worker retries the gateway call.
    end note
```

## Claim State Machine

```mermaid
stateDiagram-v2
    [*] --> CLAIMED: Customer claims with admission token
    CLAIMED --> CONFIRMED: Payment confirmed
    CLAIMED --> RELEASED: Payment failed or explicit release
    CLAIMED --> RELEASED: Expiry worker (10min TTL)
    CONFIRMED --> [*]
    RELEASED --> [*]

    note right of RELEASED
        Inventory restored in Redis.
        Next person in queue can claim.
    end note
```

## Concurrency Shield

Two independent layers prevent double-booking. Both must fail for an oversell to occur.

```mermaid
flowchart TD
    Request["Claim Request"]

    L1{"Redis SET NX lock<br/>acquired?"}
    Reject1["Return ErrAlreadyClaimed<br/>(concurrent request)"]

    L2["Atomic Lua script<br/>DECRBY inventory"]
    SoldOut["Return ErrEventSoldOut"]

    L3{"Postgres INSERT<br/>unique constraint passes?"}
    Reject3["Rollback Redis decrement<br/>Return ErrAlreadyClaimed"]

    Success["Claim created ✓"]

    Request --> L1
    L1 -->|no| Reject1
    L1 -->|yes| L2
    L2 -->|count ≤ 0| SoldOut
    L2 -->|decremented| L3
    L3 -->|violation| Reject3
    L3 -->|inserted| Success
```

## Layer Dependencies

Each layer depends only on the layers below it. No upward dependencies.

```
┌─────────────────────────────────┐
│  API (handlers, middleware)     │  ← HTTP boundary
├─────────────────────────────────┤
│  Workers (scheduler, 3 workers) │  ← background processing
├─────────────────────────────────┤
│  Services (4 services)          │  ← business logic
├─────────────────────────────────┤
│  Stores (postgres, redis)       │  ← infrastructure
├─────────────────────────────────┤
│  Domain (state machines)        │  ← pure business rules
└─────────────────────────────────┘
```