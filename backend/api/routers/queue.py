from typing import Annotated

from fastapi import APIRouter, Depends, HTTPException

from api.schemas import JoinQueueRequest, JoinQueueResponse, QueuePositionResponse
from core import QueueService
from dependencies import get_queue_repository, get_queue_service
from repositories import QueueRepository

router = APIRouter(prefix='/queue', tags=['queue'])


@router.post('/join', response_model=JoinQueueResponse, status_code=201)
async def join_queue(
	body: JoinQueueRequest,
	queue_service: Annotated[QueueService, Depends(get_queue_service)],
	queue_repo: Annotated[QueueRepository, Depends(get_queue_repository)],
):
	position = await queue_service.join_queue(
		event_id=body.event_id,
		user_id=body.user_id,
	)
	await queue_repo.create_entry(
		event_id=body.event_id,
		user_id=body.user_id,
		queue_position=position,
	)
	return JoinQueueResponse(
		position=position,
		event_id=body.event_id,
		user_id=body.user_id,
	)


@router.get('/position', response_model=QueuePositionResponse)
async def get_position(
	event_id: int,
	user_id: int,
	queue_service: Annotated[QueueService, Depends(get_queue_service)],
	queue_repo: Annotated[QueueRepository, Depends(get_queue_repository)],
):
	position = await queue_service.get_position(
		event_id=event_id,
		user_id=user_id,
	)
	if position is None:
		raise HTTPException(status_code=404, detail='User not in queue')

	entry = await queue_repo.get_entry(event_id=event_id, user_id=user_id)

	return QueuePositionResponse(
		position=position,
		status=entry.status if entry else 'waiting',  # pyright: ignore[reportArgumentType]
		event_id=event_id,
		user_id=user_id,
	)
