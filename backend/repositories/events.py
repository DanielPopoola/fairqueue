from collections.abc import Sequence
from datetime import UTC, datetime

from sqlalchemy import select, update
from sqlalchemy.ext.asyncio import AsyncSession

from models import Event, EventStatus


class EventRepository:
	def __init__(self, db: AsyncSession):
		self.db = db

	async def create_event(
		self,
		organizer_id: int,
		name: str,
		total_inventory: int,
		sale_start,
		sale_end,
		allocation_strategy,
		price_per_item: int,
	) -> Event:
		event = Event(
			organizer_id=organizer_id,
			name=name,
			total_inventory=total_inventory,
			sale_start=sale_start,
			sale_end=sale_end,
			allocation_strategy=allocation_strategy,
			price_per_item=price_per_item,
		)
		self.db.add(event)
		await self.db.flush()
		return event

	async def get_event(self, event_id: int) -> Event | None:
		stmt = select(Event).where(Event.id == event_id)
		result = await self.db.execute(stmt)
		return result.scalar_one_or_none()

	async def get_events_ready_to_activate(self) -> Sequence[Event]:
		stmt = (
			select(Event)
			.where(Event.status.in_([EventStatus.UPCOMING]))
			.where(Event.sale_start <= datetime.now(UTC))
		)
		result = await self.db.execute(stmt)
		return result.scalars().all()

	async def update_status(self, claim_id: int, status: EventStatus) -> None:
		stmt = update(Event).where(Event.id == claim_id).values(status=status)
		await self.db.execute(stmt)
