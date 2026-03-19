import pytest
import pytest_asyncio
from testcontainers.redis import RedisContainer
from redis.asyncio import Redis

from core.inventory import InventoryStore

@pytest.fixture(scope="session")
def redis_container():
    with RedisContainer("redis:7-alpine") as container:
        yield container

@pytest_asyncio.fixture
async def redis_client(redis_container):
    host = redis_container.get_container_host_ip()
    port = redis_container.get_exposed_port(6379)

    client = Redis(host=host, port=port, decode_responses=False)

    yield client

    await client.aclose()

@pytest_asyncio.fixture(autouse=True)
async def flush_redis(redis_client):
    await redis_client.flushall()
    yield


@pytest_asyncio.fixture
async def inventory_store(redis_client):
    return InventoryStore(redis_client)