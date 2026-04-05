CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email       VARCHAR(255) NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- EVENTS
-- ============================================================
CREATE TABLE events (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organizer_id     UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    name             VARCHAR(255) NOT NULL,
    total_inventory  INTEGER NOT NULL CHECK (total_inventory > 0),
    price_kobo       BIGINT NOT NULL CHECK (price_kobo > 0),
    status           VARCHAR(50) NOT NULL DEFAULT 'DRAFT',
    sale_start       TIMESTAMPTZ NOT NULL,
    sale_end         TIMESTAMPTZ NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_sale_dates CHECK (sale_end > sale_start)
);

CREATE INDEX idx_events_organizer_id ON events(organizer_id);
CREATE INDEX idx_events_status ON events(status);

-- ============================================================
-- CLAIMS
-- ============================================================
CREATE TABLE claims (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id    UUID NOT NULL REFERENCES events(id) ON DELETE RESTRICT,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    status      VARCHAR(50) NOT NULL DEFAULT 'CLAIMED',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_claims_event_id ON claims(event_id);
CREATE INDEX idx_claims_user_id ON claims(user_id);

-- Prevents two active claims for the same user/event.
-- RELEASED claims are invisible to this index so users can reclaim
-- after their previous claim expires.
CREATE UNIQUE INDEX idx_claims_active_per_user_event
    ON claims(user_id, event_id)
    WHERE status = 'CLAIMED';

-- ============================================================
-- QUEUE ENTRIES
-- ============================================================
CREATE TABLE queue_entries (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id     UUID NOT NULL REFERENCES events(id) ON DELETE RESTRICT,
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    status       VARCHAR(50) NOT NULL DEFAULT 'WAITING',
    joined_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    admitted_at  TIMESTAMPTZ,
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_queue_entries_event_id ON queue_entries(event_id);
CREATE INDEX idx_queue_entries_user_id ON queue_entries(user_id);

-- Prevents a user from being in the queue twice for the same event.
-- EXPIRED and ABANDONED entries are invisible so users can rejoin.
CREATE UNIQUE INDEX idx_queue_active_per_user_event
    ON queue_entries(user_id, event_id)
    WHERE status IN ('WAITING', 'ADMITTED');

-- ============================================================
-- PAYMENTS
-- ============================================================
CREATE TABLE payments (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    claim_id            UUID NOT NULL REFERENCES claims(id) ON DELETE RESTRICT,
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    amount_kobo         BIGINT NOT NULL CHECK (amount_kobo > 0),
    status              VARCHAR(50) NOT NULL DEFAULT 'INITIALIZING',
    reference           VARCHAR(255) UNIQUE,
    failure_reason      TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_payments_claim_id ON payments(claim_id);
CREATE INDEX idx_payments_user_id ON payments(user_id);

-- Reconciliation worker queries stale INITIALIZING and PENDING
-- payments constantly. This index makes that query fast.
CREATE INDEX idx_payments_status_updated_at
    ON payments(status, updated_at)
    WHERE status IN ('INITIALIZING', 'PENDING');