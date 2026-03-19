import pytest
import asyncio

@pytest.mark.asyncio
async def test_claim_success(inventory_store, redis_client):
    key = "event:1:available"
    await redis_client.set(key, 5)

    result = await inventory_store.claim(event_id=1)

    assert result is True

    value = await redis_client.get(key)
    assert int(value) == 4


@pytest.mark.asyncio
async def test_claim_last_item(inventory_store, redis_client):
    key = "event:1:available"
    await redis_client.set(key, 1)

    result = await inventory_store.claim(event_id=1)

    assert result is True

    value = await redis_client.get(key)
    assert int(value) == 0


@pytest.mark.asyncio
async def test_claim_sold_out(inventory_store, redis_client):
    key = "event:1:available"
    await redis_client.set(key, 0)

    result = await inventory_store.claim(event_id=1)
    
    assert result is False

    value = await redis_client.get(key)
    assert int(value) == 0


@pytest.mark.asyncio
async def test_claim_missing_key(inventory_store, redis_client):
    result = await inventory_store.claim(event_id=1)

    assert result is False

    value = await redis_client.get("event:1:available")
    assert value is None

@pytest.mark.asyncio
async def test_claim_concurrent(inventory_store, redis_client):
    key = "event:1:available"
    await redis_client.set(key, 1)

    async def attempt_claim():
        return await inventory_store.claim(event_id=1)

    results = await asyncio.gather(
        *[attempt_claim() for _ in range(50)]
    )

    assert(sum(results)) == 1
    value = await redis_client.get(key)
    assert int(value) == 0

@pytest.mark.asyncio
async def test_release_increments_counter(inventory_store, redis_client):
    key = "event:1:available"
    await redis_client.set(key, 0)

    await inventory_store.release(event_id=1)

    value = await redis_client.get(key)
    assert int(value) == 1


@pytest.mark.asyncio
async def test_release_called_twice_increments_twice(inventory_store, redis_client):
    key = "event:1:available"
    await redis_client.set(key, 0)

    await inventory_store.release(event_id=1)
    await inventory_store.release(event_id=1)

    value = await redis_client.get(key)
    assert int(value) == 2