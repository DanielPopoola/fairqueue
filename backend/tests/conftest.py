import asyncio
import pytest
import pytest_asyncio
from testcontainers.redis import RedisContainer
from testcontainers.postgres import PostgresContainer
from sqlalchemy.ext.asyncio import AsyncSession, async_sessionmaker, create_async_engine
from redis.asyncio import Redis

from core.inventory import InventoryStore
from database import Base



@pytest.fixture(scope="session")
def redis_container():
    with RedisContainer("redis:7-alpine") as container:
        yield container

@pytest.fixture(scope="session")
def postgres_container():
    with PostgresContainer("postgres:15") as container:
        yield container

@pytest_asyncio.fixture(scope="session", loop_scope="session")
def db_url(postgres_container):
    host = postgres_container.get_container_host_ip()
    port = postgres_container.get_exposed_port(5432)
    user = postgres_container.username
    password = postgres_container.password
    db = postgres_container.dbname
    return f"postgresql+asyncpg://{user}:{password}@{host}:{port}/{db}"

@pytest_asyncio.fixture(scope="session", loop_scope="session")
async def db_engine(db_url):
    engine = create_async_engine(db_url)
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)
    yield engine
    await engine.dispose()


@pytest_asyncio.fixture(loop_scope="session")
async def db_session(db_engine):
    session_factory = async_sessionmaker(
        db_engine, class_=AsyncSession, autoflush=False, expire_on_commit=False
    )
    async with session_factory() as session:
        yield session
        await session.rollback()


@pytest_asyncio.fixture(scope="session", loop_scope="session")
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


class FailOnSecondUpdateSession(AsyncSession):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self._update_count = 0

    async def execute(self, statement, *args, **kwargs):
        from sqlalchemy import Update
        if isinstance(statement, Update):
            self._update_count += 1
            if self._update_count == 2:
                raise Exception("Simulated DB failure on second write")
        return await super().execute(statement, *args, **kwargs)
    

@pytest_asyncio.fixture
async def failing_db_session(db_engine):
    async with FailOnSecondUpdateSession(db_engine, autoflush=True, expire_on_commit=False) as session:
        yield session
        await session.rollback()
