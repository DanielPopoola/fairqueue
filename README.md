# FairQueue

> **Fair ticketing for Nigerian live entertainment. Built for the moment 50,000 fans hit a website at once.**

FairQueue is a SaaS ticketing platform built specifically for Nigerian concert and show promoters. It prevents websites from crashing under demand spikes, eliminates bot scalping, and ensures real fans get a fair shot at tickets — through an intelligent virtual queue system.

---

## The Problem

Nigerian live entertainment has a specific, brutal problem: when a Wizkid or Burna Boy concert goes on sale, demand is not spread across days. It is compressed into seconds.

- 50,000 fans hit a ticketing site at exactly 10:00 AM
- The site crashes within 2 minutes
- When it recovers, bots have claimed 60% of inventory
- Real fans are locked out
- The promoter loses revenue to the secondary market and loses fan trust

Existing platforms — Tix Africa, Paystack Tickets, Eventbrite — are not built for this load profile. They are built for trickle demand. FairQueue is built for the spike.

---

## The Solution

FairQueue puts a virtual queue in front of your ticket sale. Every fan who shows up gets a position. The system admits them at a controlled rate. Only admitted fans can attempt to purchase. The inventory layer uses atomic Redis operations so double-booking is structurally impossible.

```
Fan hits the sale page
    ↓
Joins queue → assigned position #4,821
    ↓
Real-time updates via WebSocket: #4,821 → #3,200 → #1,100 → #47
    ↓
Admitted: "It's your turn. You have 5 minutes."
    ↓
Selects ticket → completes Paystack payment
    ↓
Confirmation delivered
```

The promoter's site never sees 50,000 simultaneous requests. The queue absorbs the spike. The inventory layer handles the rest.

---

## Who This Is For

FairQueue is purpose-built for **Nigerian live entertainment**:

- Afrobeats and Amapiano concert promoters
- Comedy show organisers
- Detty December event organisers
- Any high-demand event where tickets sell out in minutes

**Out of scope for MVP:** tech conferences, product drops, appointment booking, university hostel allocation. These do not face the demand concentration problem FairQueue is designed to solve. They are better served by simpler tools.

---

## Architecture

The architecture separates three distinct problems that existing platforms conflate:

```
┌─────────────────────────────────────────────┐
│  Edge Layer (Cloudflare)                    │
│  DDoS protection, rate limiting per IP      │
└──────────────┬──────────────────────────────┘
               ↓
┌─────────────────────────────────────────────┐
│  Queue Service (Redis ZSET)                 │
│  Assigns positions, controls admission rate │
│  WebSocket broadcasts live updates          │
└──────────────┬──────────────────────────────┘
               ↓
┌─────────────────────────────────────────────┐
│  Inventory Coordinator (Redis + Lua)        │
│  Atomic claim/release, 10-minute expiry     │
│  Structurally prevents double-booking       │
└──────────────┬──────────────────────────────┘
               ↓
┌─────────────────────────────────────────────┐
│  Payment Service (Paystack)                 │
│  Webhook handling, idempotent operations    │
│  Automatic inventory release on failure     │
└──────────────┬──────────────────────────────┘
               ↓
┌─────────────────────────────────────────────┐
│  PostgreSQL                                 │
│  Permanent records, full audit trail        │
└─────────────────────────────────────────────┘
```

**Key design decisions:**
- Queue and inventory are separate systems — the queue handles the herd, Redis handles atomicity
- Inventory uses a counter + Lua script, not individual item keys — fast, atomic, simple
- On failure, the system fails inflated (detectable) not deflated (silent) — by design
- Optimistic locking (`UPDATE WHERE status = CLAIMED`) prevents concurrent release races

**Tech stack:**

| Layer | Technology |
|-------|-----------|
| API | Python 3.12 + FastAPI |
| Queue & coordination | Redis 7+ with Lua scripting |
| Database | PostgreSQL 15+ |
| Payments | Paystack |
| Real-time | WebSockets (FastAPI native) |
| Deployment | Docker Compose |
| Monitoring | Prometheus + Grafana |

---

## Quick Start

**Prerequisites:** Docker and Docker Compose.

```bash
# Clone the repository
git clone https://github.com/DanielPopoola/fairqueue.git
cd fairqueue/backend

# Set up environment variables
cp .env.example .env
# Add your Paystack secret key and database credentials to .env

# Start all services
docker-compose up -d

# Run database migrations
docker-compose exec api alembic upgrade head

# Verify everything is running
curl http://localhost:8000/info
```

**Create your first event:**

```bash
curl -X POST http://localhost:8000/events \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Wizkid Live in Lagos",
    "organizer_id": 1,
    "total_inventory": 5000,
    "price_per_item": 25000,
    "sale_start": "2024-06-01T10:00:00Z",
    "sale_end": "2024-06-01T23:59:00Z",
    "allocation_strategy": "fifo"
  }'
```

---

## API Reference

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/events` | Create a new event |
| `POST` | `/claims` | Claim a ticket for an event |
| `POST` | `/queue/join` | Join the queue for an event |
| `GET` | `/queue/position` | Get current queue position |

Full API docs available at `http://localhost:8000/docs` when running locally.

---

## Development

```bash
# Install dependencies
pip install uv
uv pip install --system -e .

# Run tests
pytest

# Run with hot reload
uvicorn main:app --reload
```

**Running tests against real infrastructure (recommended):**

Tests use `testcontainers` to spin up real Redis and PostgreSQL instances. No mocks for infrastructure — the tests prove actual behaviour under concurrency.

```bash
# Full test suite including race condition tests
pytest tests/ -v
```

---

## Roadmap

### MVP (current)
- [x] Redis-based atomic inventory with Lua scripts
- [x] FIFO queue service
- [x] Claim expiry background worker
- [x] Event activation worker
- [x] Paystack payment integration
- [ ] WebSocket live queue position updates
- [ ] Promoter dashboard

### Phase 2
- [ ] Ticket tiers with separate inventory pools (VIP, Regular, Early Bird)
- [ ] Waitlist management — auto-assign released inventory
- [ ] Abandoned cart recovery
- [ ] SMS notifications via Termii

### Future
- [ ] Named tickets tied to buyer ID (anti-scalping)
- [ ] QR code generation and gate scanning
- [ ] Bot detection

---

## Why Not Just Use Eventbrite / Tix Africa?

They crash. Under the demand profile of a Nigerian concert sale — tens of thousands of fans at the same second — standard ticketing platforms are not architected to survive. FairQueue's queue layer exists precisely to absorb that spike before it reaches the database.

Additionally: FairQueue charges 3% per transaction. Most alternatives charge 5–10%.

---

## License

MIT — see [LICENSE](LICENSE) for details.

---

**Built in Lagos, Nigeria 🇳🇬**