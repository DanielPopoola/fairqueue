import pytest
import asyncio


@pytest.mark.asyncio
async def test_earlie_arrival_gets_lower_position(queue_service):
	pos_a = await queue_service.join_queue(event_id=1, user_id=1)
	pos_b = await queue_service.join_queue(event_id=1, user_id=2)

	assert pos_a < pos_b


@pytest.mark.asyncio
async def test_same_user_joining_twice_keeps_original_position(queue_service):
	pos_first = await queue_service.join_queue(event_id=1, user_id=1)
	await queue_service.join_queue(event_id=1, user_id=2)
	pos_second_join = await queue_service.join_queue(event_id=1, user_id=1)

	assert pos_first == pos_second_join


@pytest.mark.asyncio
async def test_position_updates_after_admission(queue_service):
	await queue_service.join_queue(event_id=1, user_id=1)
	await queue_service.join_queue(event_id=1, user_id=2)
	await queue_service.join_queue(event_id=1, user_id=3)

	await queue_service.admit_next(event_id=1, count=2)

	position = await queue_service.get_position(event_id=1, user_id=3)
	assert position == 1


@pytest.mark.asyncio
async def test_concurrent_admission_admits_exactly_once(queue_service):
	for user_id in range(5):
		await queue_service.join_queue(event_id=1, user_id=user_id)

	results = await asyncio.gather(
		queue_service.admit_next(event_id=1, count=5),
		queue_service.admit_next(event_id=1, count=5),
	)

	all_admitted = results[0] + results[1]
	assert len(all_admitted) == 5
	assert len(set(all_admitted)) == 5
