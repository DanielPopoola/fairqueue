from typing import Annotated

from fastapi import APIRouter, Depends, HTTPException

from api.schemas import CreateEventRequest, EventResponse
from dependencies import get_events_service
from services import EventService

router = APIRouter(prefix='/events', tags=['events'])


@router.post('/', response_model=EventResponse, status_code=201)
async def create_event(
	body: CreateEventRequest, service: Annotated[EventService, Depends(get_events_service)]
):
	try:
		result = await service.create_event(
			organizer_id=body.organizer_id,
			name=body.name,
			total_inventory=body.total_inventory,
			sale_start=body.sale_start,
			sale_end=body.sale_end,
			allocation_strategy=body.allocation_strategy,
			price_per_item=body.price_per_item,
		)
		return EventResponse.model_validate(result)
	except ValueError as e:
		raise HTTPException(status_code=409, detail=str(e)) from e
