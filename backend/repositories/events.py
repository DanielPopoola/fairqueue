from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy import select

from models import Event


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