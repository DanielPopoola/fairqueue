-- Migration: 000001_initial_schema.up.sql

-- ============================================================
-- ORGANIZERS
-- ============================================================
CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION pg_stat_statements;

CREATE TABLE IF NOT EXISTS organizers (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          VARCHAR(255) NOT NULL,
    email         CITEXT NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- CUSTOMERS
-- ============================================================
CREATE TABLE IF NOT EXISTS customers (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email      CITEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- EVENTS
-- ============================================================
CREATE TABLE IF NOT EXISTS events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organizer_id    UUID NOT NULL REFERENCES organizers(id) ON DELETE RESTRICT,
    name            VARCHAR(255) NOT NULL,
    total_inventory INTEGER NOT NULL CHECK (total_inventory > 0),
    price_kobo      BIGINT NOT NULL CHECK (price_kobo > 0),
    status          VARCHAR(50) NOT NULL DEFAULT 'DRAFT',
    sale_start      TIMESTAMPTZ NOT NULL,
    sale_end        TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_sale_dates CHECK (sale_end > sale_start)
);

CREATE INDEX IF NOT EXISTS idx_events_organizer_id ON events(organizer_id);
CREATE INDEX IF NOT EXISTS idx_events_status ON events(status);

-- ============================================================
-- CLAIMS
-- ============================================================
CREATE TABLE IF NOT EXISTS claims (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id    UUID NOT NULL REFERENCES events(id) ON DELETE RESTRICT,
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
    status      VARCHAR(50) NOT NULL DEFAULT 'CLAIMED',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_claims_event_id ON claims(event_id);
CREATE INDEX IF NOT EXISTS idx_claims_customer_id ON claims(customer_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_claims_active_per_customer_event
    ON claims(customer_id, event_id)
    WHERE status = 'CLAIMED';

-- ============================================================
-- QUEUE ENTRIES
-- ============================================================
CREATE TABLE IF NOT EXISTS queue_entries (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id    UUID NOT NULL REFERENCES events(id) ON DELETE RESTRICT,
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
    status      VARCHAR(50) NOT NULL DEFAULT 'WAITING',
    joined_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    admitted_at TIMESTAMPTZ,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_queue_entries_event_id ON queue_entries(event_id);
CREATE INDEX IF NOT EXISTS idx_queue_entries_customer_id ON queue_entries(customer_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_queue_active_per_customer_event
    ON queue_entries(customer_id, event_id)
    WHERE status IN ('WAITING', 'ADMITTED');

-- ============================================================
-- PAYMENTS
-- ============================================================
CREATE TABLE IF NOT EXISTS payments (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    claim_id            UUID NOT NULL REFERENCES claims(id) ON DELETE RESTRICT,
    customer_id         UUID NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
    amount_kobo         BIGINT NOT NULL CHECK (amount_kobo > 0),
    status              VARCHAR(50) NOT NULL DEFAULT 'INITIALIZING',
    reference           VARCHAR(255) UNIQUE,
    authorization_url   VARCHAR(500),
    failure_reason      TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_payments_one_per_claim ON payments(claim_id);
CREATE INDEX IF NOT EXISTS idx_payments_customer_id ON payments(customer_id);

CREATE INDEX IF NOT EXISTS idx_payments_status_updated_at
    ON payments(status, updated_at)
    WHERE status IN ('INITIALIZING', 'PENDING');