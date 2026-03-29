import asyncio
import pytest
import pytest_asyncio
from httpx import AsyncClient, ASGITransport

from testcontainers.redis import RedisContainer
from testcontainers.postgres import PostgresContainer
from sqlalchemy import text
from sqlalchemy.ext.asyncio import AsyncSession, async_sessionmaker, create_async_engine
from redis.asyncio import Redis

from dependencies import get_db_session, get_redis
from main import app
from core import InventoryStore, QueueService
from database import Base
from core.inventory import InventoryStore
from models.claim import Claim, ClaimStatus
from models.event import Event, AllocationStrategy, EventStatus
from models.user import User
from repositories.claims import ClaimsRepository
from services.claim_service import ClaimService
from datetime import UTC, datetime, timedelta


@pytest.fixture(scope='session')
def redis_container():
	with RedisContainer('redis:7-alpine') as container:
		yield container


@pytest.fixture(scope='session')
def postgres_container():
	with PostgresContainer('postgres:15') as container:
		yield container


@pytest_asyncio.fixture(scope='session', loop_scope='session')
def db_url(postgres_container):
	host = postgres_container.get_container_host_ip()
	port = postgres_container.get_exposed_port(5432)
	user = postgres_container.username
	password = postgres_container.password
	db = postgres_container.dbname
	return f'postgresql+asyncpg://{user}:{password}@{host}:{port}/{db}'


@pytest_asyncio.fixture(scope='session', loop_scope='session')
async def db_engine(db_url):
	engine = create_async_engine(db_url)
	async with engine.begin() as conn:
		await conn.run_sync(Base.metadata.create_all)
		# create_all skips partial indexes — create manually
		await conn.execute(
			text("""
            CREATE UNIQUE INDEX IF NOT EXISTS uq_claims_active_user_event
            ON claims (user_id, event_id)
            WHERE status IN ('CLAIMED', 'PAYMENT_PENDING')
        """)
		)
	yield engine
	await engine.dispose()


@pytest_asyncio.fixture(loop_scope='session')
async def db_session(db_engine):
	session_factory = async_sessionmaker(
		db_engine, class_=AsyncSession, autoflush=False, expire_on_commit=False
	)
	async with session_factory() as session:
		yield session
		await session.rollback()


@pytest_asyncio.fixture(scope='session', loop_scope='session')
async def real_redis(redis_container):
	host = redis_container.get_container_host_ip()
	port = redis_container.get_exposed_port(6379)
	client = Redis(host=host, port=int(port), decode_responses=False)
	yield client
	await client.flushall()
	await client.aclose()


@pytest_asyncio.fixture(autouse=True)
async def flush_redis(real_redis):
	await real_redis.flushall()
	yield


@pytest_asyncio.fixture
async def inventory_store(real_redis):
	return InventoryStore(real_redis)


@pytest_asyncio.fixture
async def queue_service(real_redis):
	return QueueService(real_redis)


@pytest_asyncio.fixture
async def api_client(db_engine, real_redis):
	session_factory = async_sessionmaker(
		db_engine, class_=AsyncSession, autoflush=False, expire_on_commit=False
	)

	async def override_db():
		async with session_factory() as session:
			try:
				yield session
				await session.commit()
			except Exception:
				await session.rollback()
				raise

	async def override_redis():
		return real_redis

	app.dependency_overrides[get_db_session] = override_db
	app.dependency_overrides[get_redis] = override_redis

	async with AsyncClient(transport=ASGITransport(app=app), base_url='http://test') as client:
		yield client

	app.dependency_overrides.clear()



class FailOnSecondUpdateSession(AsyncSession):
	def __init__(self, *args, **kwargs):
		super().__init__(*args, **kwargs)
		self._update_count = 0

	async def execute(self, statement, *args, **kwargs):
		from sqlalchemy import Update

		if isinstance(statement, Update):
			self._update_count += 1
			if self._update_count == 2:
				raise Exception('Simulated DB failure on second write')
		return await super().execute(statement, *args, **kwargs)


@pytest_asyncio.fixture
async def failing_db_session(db_engine):
	async with FailOnSecondUpdateSession(db_engine, autoflush=True, expire_on_commit=False) as session:
		yield session
		await session.rollback()


# Seed helpers


# --- Seed helpers ---


async def seed_event(session: AsyncSession) -> Event:
	event = Event(
		organizer_id=1,
		name='Test Event',
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
	user = User(email=f'user_{datetime.now().timestamp()}@test.com', name='Test User')
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
