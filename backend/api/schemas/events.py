from datetime import datetime

from pydantic import BaseModel, field_validator

from models import AllocationStrategy


class CreateEventRequest(BaseModel):
	name: str
	organizer_id: int
	total_inventory: int
	sale_start: datetime
	sale_end: datetime
	allocation_strategy: AllocationStrategy
	price_per_item: int  # in naira

	@field_validator('price_per_item')
	@classmethod
	def convert_to_kobo(cls, v: int) -> int:
		return v * 100


class EventResponse(BaseModel):
	id: int
	name: str
	total_inventory: int
	sale_start: datetime
	sale_end: datetime
	status: str

	model_config = {'from_attributes': True}
