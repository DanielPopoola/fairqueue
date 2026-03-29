import asyncio
import pytest
from sqlalchemy.ext.asyncio import AsyncSession, async_sessionmaker

from conftest import seed_event, seed_claim, seed_user


@pytest.mark.asyncio
async def test_create_event(api_client):
	payload = {
		'organizer_id': 1,
		'name': 'Test Event',
		'total_inventory': 100,
		'sale_start': '2026-03-30T10:00:00',
		'sale_end': '2026-03-31T10:00:00',
		'allocation_strategy': 'fifo',
		'price_per_item': 500,
	}

	response = await api_client.post('/events/', json=payload)

	assert response.status_code == 201

	data = response.json()
	assert data['name'] == 'Test Event'
	assert data['total_inventory'] == 100


@pytest.mark.asyncio
async def test_create_claim_success(api_client, real_redis, db_engine):
	async with async_sessionmaker(db_engine, class_=AsyncSession, expire_on_commit=False)() as session:
		event = await seed_event(session)
		user = await seed_user(session)
		await session.commit()
		event_id = event.id
		user_id = user.id

	await real_redis.set(f'event:{event_id}:available', 5)

	payload = {
		'event_id': event_id,
		'user_id': user_id,
	}

	response = await api_client.post('/claims/', json=payload)
	print(response.json())
	assert response.status_code == 201

	data = response.json()
	assert data['event_id'] == event_id
	assert data['user_id'] == user_id

	# verify inventory decreased
	remaining = await real_redis.get(f'event:{event_id}:available')
	assert int(remaining) == 4


@pytest.mark.asyncio
async def test_create_claim_sold_out(api_client, real_redis, db_engine):
	async with async_sessionmaker(db_engine, class_=AsyncSession, expire_on_commit=False)() as session:
		event = await seed_event(session)
		user = await seed_user(session)
		await session.commit()
		event_id = event.id
		user_id = user.id

	await real_redis.set(f'event:{event_id}:available', 0)

	response = await api_client.post(
		'/claims/',
		json={
			'event_id': event_id,
			'user_id': user_id,
		},
	)

	assert response.status_code == 409
	assert response.json()['detail'] == 'Sold out'


@pytest.mark.asyncio
async def test_create_claim_duplicate(api_client, real_redis, db_engine):
	async with async_sessionmaker(db_engine, class_=AsyncSession, expire_on_commit=False)() as session:
		event = await seed_event(session)
		user = await seed_user(session)
		await session.commit()
		event_id = event.id
		user_id = user.id

	await real_redis.set(f'event:{event_id}:available', 5)

	payload = {'event_id': event_id, 'user_id': user_id}

	# first claim
	res1 = await api_client.post('/claims/', json=payload)
	assert res1.status_code == 201

	# second claim
	res2 = await api_client.post('/claims/', json=payload)
	assert res2.status_code == 409


@pytest.mark.asyncio
async def test_concurrent_claims_same_user(api_client, real_redis, db_engine):
    async with async_sessionmaker(db_engine, class_=AsyncSession, expire_on_commit=False)() as session:
        event = await seed_event(session)
        user = await seed_user(session)
        await session.commit()
        event_id = event.id
        user_id = user.id

    await real_redis.set(f"event:{event_id}:available", 5)

    responses = await asyncio.gather(
        api_client.post("/claims/", json={"event_id": event_id, "user_id": user_id}),
        api_client.post("/claims/", json={"event_id": event_id, "user_id": user_id}),
        return_exceptions=True,
    )

    success = [r for r in responses if not isinstance(r, Exception) and r.status_code == 201]
    assert len(success) == 1


@pytest.mark.asyncio
async def test_join_queue_success(api_client, db_engine):
    async with async_sessionmaker(db_engine, class_=AsyncSession, expire_on_commit=False)() as session:
        event = await seed_event(session)
        user = await seed_user(session)
        await session.commit()
        event_id = event.id
        user_id = user.id

    response = await api_client.post("/queue/join", json={
        "event_id": event_id,
        "user_id": user_id,
    })
    assert response.status_code == 201
    data = response.json()
    assert data["position"] == 1
    assert data["event_id"] == event_id


@pytest.mark.asyncio
async def test_get_queue_position(api_client, db_engine):
    async with async_sessionmaker(db_engine, class_=AsyncSession, expire_on_commit=False)() as session:
        event = await seed_event(session)
        user = await seed_user(session)
        await session.commit()
        event_id = event.id
        user_id = user.id

    await api_client.post("/queue/join", json={"event_id": event_id, "user_id": user_id})

    response = await api_client.get(f"/queue/position?event_id={event_id}&user_id={user_id}")
    assert response.status_code == 200
    data = response.json()
    assert data["position"] == 1


@pytest.mark.asyncio
async def test_get_queue_position_not_found(api_client):
    response = await api_client.get("/queue/position?event_id=999&user_id=999")
    assert response.status_code == 404
