import pytest
import asyncio

from sqlalchemy.ext.asyncio import AsyncSession

from core.inventory import InventoryStore
from models.claim import Claim, ClaimStatus
from models.event import Event, AllocationStrategy, EventStatus
from models.user import User
from repositories.claims import ClaimsRepository
from services.claim_service import ClaimService
from datetime import UTC, datetime, timedelta


# --- Seed helpers ---

async def seed_event(session: AsyncSession) -> Event:
    event = Event(
        organizer_id=1,
        name="Test Event",
        total_inventory=100,
        sale_start=datetime.now(UTC),
        sale_end=datetime.now(UTC) + timedelta(hours=2),
        allocation_strategy=AllocationStrategy.FIFO,
        price_per_item=5000,
        status=EventStatus.ACTIVE,
    )
    session.add(event)
    await session.flush()
    return event


async def seed_user(session: AsyncSession) -> User:
    user = User(email=f"user_{datetime.now().timestamp()}@test.com", name="Test User")
    session.add(user)
    await session.flush()
    return user


async def seed_claim(session: AsyncSession, event_id: int, user_id: int, status: ClaimStatus) -> Claim:
    claim = Claim(
        event_id=event_id,
        user_id=user_id,
        status=status,
        expires_at=datetime.now(UTC) + timedelta(minutes=10),
    )
    session.add(claim)
    await session.flush()
    return claim

# --- Tests ---

@pytest.mark.asyncio
async def test_release_valid_claim(db_session, real_redis):
    event = await seed_event(db_session)
    user = await seed_user(db_session)
    claim = await seed_claim(db_session, event.id, user.id, ClaimStatus.CLAIMED)

    await real_redis.set(f"event:{event.id}:available", 0)

    repo = ClaimsRepository(db_session)
    store = InventoryStore(real_redis)
    service = ClaimService(repo, store)

    await service.release_claim(claim_id=claim.id, event_id=event.id)

    await db_session.refresh(claim)
    assert claim.status == ClaimStatus.RELEASED

    count = await real_redis.get(f"event:{event.id}:available")
    assert int(count) == 1


@pytest.mark.asyncio
async def test_double_release_is_idempotent(db_session, real_redis):
    event = await seed_event(db_session)
    user = await seed_user(db_session)
    claim = await seed_claim(db_session, event.id, user.id, ClaimStatus.RELEASED)

    await real_redis.set(f"event:{event.id}:available", 5)

    repo = ClaimsRepository(db_session)
    store = InventoryStore(real_redis)
    service = ClaimService(repo, store)

    await service.release_claim(claim_id=claim.id, event_id=event.id)

    count = await real_redis.get(f"event:{event.id}:available")
    assert int(count) == 5  # unchanged

@pytest.mark.asyncio
async def test_release_nonexistent_claim(db_session, real_redis):
    await real_redis.set("event:999:available", 5)

    repo = ClaimsRepository(db_session)
    store = InventoryStore(real_redis)
    service = ClaimService(repo, store)

    await service.release_claim(claim_id=99999, event_id=999)

    count = await real_redis.get("event:999:available")
    assert int(count) == 5  # unchanged

@pytest.mark.asyncio
async def test_release_wrong_event(db_session, real_redis):
    event = await seed_event(db_session)
    user = await seed_user(db_session)
    claim = await seed_claim(db_session, event.id, user.id, ClaimStatus.CLAIMED)

    await real_redis.set(f"event:9999:available", 5)

    repo = ClaimsRepository(db_session)
    store = InventoryStore(real_redis)
    service = ClaimService(repo, store)

    await service.release_claim(claim_id=claim.id, event_id=9999)

    count = await real_redis.get("event:9999:available")
    assert int(count) == 5  # unchanged

@pytest.mark.asyncio
async def test_release_confirmed_claim_is_blocked(db_session, real_redis):
    event = await seed_event(db_session)
    user = await seed_user(db_session)
    claim = await seed_claim(db_session, event.id, user.id, ClaimStatus.CONFIRMED)

    await real_redis.set(f"event:{event.id}:available", 5)

    repo = ClaimsRepository(db_session)
    store = InventoryStore(real_redis)
    service = ClaimService(repo, store)

    await service.release_claim(claim_id=claim.id, event_id=event.id)

    count = await real_redis.get(f"event:{event.id}:available")
    assert int(count) == 5  # unchanged


@pytest.mark.asyncio
async def test_redis_suceeds_db_final_write_fails(failing_db_session, real_redis):
    event = await seed_event(failing_db_session)
    user = await seed_user(failing_db_session)
    claim = await seed_claim(failing_db_session, event.id, user.id, ClaimStatus.CLAIMED)
    await failing_db_session.commit()

    await real_redis.set(f"event:{event.id}:available", 0)

    repo = ClaimsRepository(failing_db_session)
    store = InventoryStore(real_redis)
    service = ClaimService(repo, store)

    with pytest.raises(Exception, match="Simulated DB failure"):
        await service.release_claim(claim_id=claim.id, event_id=event.id)

    # Redis was incremented (inflated) — detectable inconsistency
    count = await real_redis.get(f"event:{event.id}:available")
    assert int(count) == 1

    # DB is stuck at RELEASING — expiry worker can detect and retry
    await failing_db_session.refresh(claim)
    assert claim.status == ClaimStatus.RELEASING

@pytest.mark.asyncio
async def test_concurrent_release_increments_once(db_session, real_redis):
    event = await seed_event(db_session)
    user = await seed_user(db_session)
    claim = await seed_claim(db_session, event.id, user.id, ClaimStatus.CLAIMED)
    await db_session.commit()

    await real_redis.set(f"event:{event.id}:available", 0)

    repo = ClaimsRepository(db_session)
    store = InventoryStore(real_redis)
    service = ClaimService(repo, store)

    await asyncio.gather(*[
        service.release_claim(claim_id=claim.id, event_id=event.id)
        for _ in range(10)
    ])

    count = await real_redis.get(f"event:{event.id}:available")
    assert int(count) == 1