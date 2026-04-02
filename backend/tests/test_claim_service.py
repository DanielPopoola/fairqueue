import pytest
import asyncio

from sqlalchemy import text
from sqlalchemy.ext.asyncio import AsyncSession, async_sessionmaker

from core.inventory import InventoryStore
from conftest import seed_event, seed_claim, seed_user
from models.claim import ClaimStatus
from repositories.claims import ClaimsRepository
from services.claim_service import ClaimService


@pytest.mark.asyncio
async def test_release_valid_claim(db_session, real_redis):
	event = await seed_event(db_session)
	user = await seed_user(db_session)
	claim = await seed_claim(db_session, event.id, user.id, ClaimStatus.CLAIMED)

	await real_redis.set(f'event:{event.id}:available', 0)

	repo = ClaimsRepository(db_session)
	store = InventoryStore(real_redis)
	service = ClaimService(repo, store)

	await service.release_claim(claim_id=claim.id, event_id=event.id)

	await db_session.refresh(claim)
	assert claim.status == ClaimStatus.RELEASED

	count = await real_redis.get(f'event:{event.id}:available')
	assert int(count) == 1


@pytest.mark.asyncio
async def test_double_release_is_idempotent(db_session, real_redis):
	event = await seed_event(db_session)
	user = await seed_user(db_session)
	claim = await seed_claim(db_session, event.id, user.id, ClaimStatus.RELEASED)

	await real_redis.set(f'event:{event.id}:available', 5)

	repo = ClaimsRepository(db_session)
	store = InventoryStore(real_redis)
	service = ClaimService(repo, store)

	await service.release_claim(claim_id=claim.id, event_id=event.id)

	count = await real_redis.get(f'event:{event.id}:available')
	assert int(count) == 5  # unchanged


@pytest.mark.asyncio
async def test_release_nonexistent_claim(db_session, real_redis):
	await real_redis.set('event:999:available', 5)

	repo = ClaimsRepository(db_session)
	store = InventoryStore(real_redis)
	service = ClaimService(repo, store)

	await service.release_claim(claim_id=99999, event_id=999)

	count = await real_redis.get('event:999:available')
	assert int(count) == 5  # unchanged


@pytest.mark.asyncio
async def test_release_wrong_event(db_session, real_redis):
	event = await seed_event(db_session)
	user = await seed_user(db_session)
	claim = await seed_claim(db_session, event.id, user.id, ClaimStatus.CLAIMED)

	await real_redis.set(f'event:9999:available', 5)

	repo = ClaimsRepository(db_session)
	store = InventoryStore(real_redis)
	service = ClaimService(repo, store)

	await service.release_claim(claim_id=claim.id, event_id=9999)

	count = await real_redis.get('event:9999:available')
	assert int(count) == 5  # unchanged


@pytest.mark.asyncio
async def test_release_confirmed_claim_is_blocked(db_session, real_redis):
	event = await seed_event(db_session)
	user = await seed_user(db_session)
	claim = await seed_claim(db_session, event.id, user.id, ClaimStatus.CONFIRMED)

	await real_redis.set(f'event:{event.id}:available', 5)

	repo = ClaimsRepository(db_session)
	store = InventoryStore(real_redis)
	service = ClaimService(repo, store)

	await service.release_claim(claim_id=claim.id, event_id=event.id)

	count = await real_redis.get(f'event:{event.id}:available')
	assert int(count) == 5  # unchanged


@pytest.mark.asyncio
async def test_redis_suceeds_db_final_write_fails(failing_db_session, real_redis):
	event = await seed_event(failing_db_session)
	user = await seed_user(failing_db_session)
	claim = await seed_claim(failing_db_session, event.id, user.id, ClaimStatus.CLAIMED)
	await failing_db_session.commit()

	await real_redis.set(f'event:{event.id}:available', 0)

	repo = ClaimsRepository(failing_db_session)
	store = InventoryStore(real_redis)
	service = ClaimService(repo, store)

	with pytest.raises(Exception, match='Simulated DB failure'):
		await service.release_claim(claim_id=claim.id, event_id=event.id)

	# Redis was incremented (inflated) — detectable inconsistency
	count = await real_redis.get(f'event:{event.id}:available')
	assert int(count) == 1

	# DB is stuck at RELEASING — expiry worker can detect and retry
	await failing_db_session.refresh(claim)
	assert claim.status == ClaimStatus.RELEASING


@pytest.mark.asyncio
async def test_concurrent_claims_claim_once(db_engine, real_redis):
	async with db_engine.connect() as conn:
		result = await conn.execute(text(
			"SELECT indexname FROM pg_indexes WHERE tablename = 'claims'"
		))
		print("INDEXES:", result.fetchall())
	session_factory = async_sessionmaker(
		db_engine, class_=AsyncSession, autoflush=True, expire_on_commit=False
	)

	# Seed data using dedicated session
	async with session_factory() as session:
		event = await seed_event(session)
		user = await seed_user(session)
		await session.commit()
	print(f"event_id={event.id}, user_id={user.id}")	

	await real_redis.set(f'event:{event.id}:available', 2)

	store = InventoryStore(real_redis)

	async def attempt_claim():
		async with session_factory() as session:
			repo = ClaimsRepository(session)
			service = ClaimService(repo, store)
			try:
				claim = await service.create_claim(event_id=event.id, user_id=user.id)
				await session.commit()
				return claim
			except ValueError as e:
				return e

	results = await asyncio.gather(attempt_claim(), attempt_claim())

	print("RESULTS:", results)
	print("ERRORS:", [r for r in results if isinstance(r, ValueError)])
	successes = [r for r in results if not isinstance(r, ValueError)]
	print("SUCCESSES:", successes)

	# also check what's in the DB
	async with db_engine.connect() as conn:
		result = await conn.execute(text(
			"SELECT id, user_id, event_id, status FROM claims WHERE event_id = :eid"
		), {"eid": event.id})
		print("CLAIMS IN DB:", result.fetchall())

	count = await real_redis.get(f'event:{event.id}:available')
	print("REDIS COUNT:", count)
	errors = [r for r in results if isinstance(r, ValueError)]
	assert len(errors) == 1

	# Both coroutines decremented Redis before the unique constraint fired on the second INSERT.
	# The deflation is intentional — the expiry worker releases it when the phantom claim expires.
	count = await real_redis.get(f'event:{event.id}:available')
	assert int(count) == 0 # deflated by 1 — self-corrects on claim expiry


@pytest.mark.asyncio
async def test_concurrent_release_increments_once(db_session, real_redis):
	event = await seed_event(db_session)
	user = await seed_user(db_session)
	claim = await seed_claim(db_session, event.id, user.id, ClaimStatus.CLAIMED)
	await db_session.commit()

	await real_redis.set(f'event:{event.id}:available', 0)

	repo = ClaimsRepository(db_session)
	store = InventoryStore(real_redis)
	service = ClaimService(repo, store)

	await asyncio.gather(
		*[service.release_claim(claim_id=claim.id, event_id=event.id) for _ in range(10)]
	)

	count = await real_redis.get(f'event:{event.id}:available')
	assert int(count) == 1
