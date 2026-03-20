from datetime import datetime

from models import AllocationStrategy, Event
from repositories import EventRepository


class EventService:
	def __init__(self, event_repo: EventRepository):
		self.event_repo = event_repo

	async def create_event(
		self,
		organizer_id: int,
		name: str,
		total_inventory: int,
		sale_start: datetime,
		sale_end: datetime,
		allocation_strategy: AllocationStrategy,
		price_per_item: int,
	) -> Event:
		return await self.event_repo.create_event(
			organizer_id,
			name,
			total_inventory,
			sale_start,
			sale_end,
			allocation_strategy,
			price_per_item,
		)
