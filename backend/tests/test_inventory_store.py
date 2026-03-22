import pytest
import asyncio

@pytest.mark.asyncio
async def test_claim_success(inventory_store, real_redis):
    key = "event:1:available"
    await real_redis.set(key, 5)

    result = await inventory_store.claim(event_id=1)

    assert result is True

    value = await real_redis.get(key)
    assert int(value) == 4


@pytest.mark.asyncio
async def test_claim_last_item(inventory_store, real_redis):
    key = "event:1:available"
    await real_redis.set(key, 1)

    result = await inventory_store.claim(event_id=1)

    assert result is True

    value = await real_redis.get(key)
    assert int(value) == 0


@pytest.mark.asyncio
async def test_claim_sold_out(inventory_store, real_redis):
    key = "event:1:available"
    await real_redis.set(key, 0)

    result = await inventory_store.claim(event_id=1)
    
    assert result is False

    value = await real_redis.get(key)
    assert int(value) == 0


@pytest.mark.asyncio
async def test_claim_missing_key(inventory_store, real_redis):
    result = await inventory_store.claim(event_id=1)

    assert result is False

    value = await real_redis.get("event:1:available")
    assert value is None

@pytest.mark.asyncio
async def test_claim_concurrent(inventory_store, real_redis):
    key = "event:1:available"
    await real_redis.set(key, 1)

    async def attempt_claim():
        return await inventory_store.claim(event_id=1)

    results = await asyncio.gather(
        *[attempt_claim() for _ in range(50)]
    )

    assert(sum(results)) == 1
    value = await real_redis.get(key)
    assert int(value) == 0

