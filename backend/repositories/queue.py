from datetime import UTC, datetime

from sqlalchemy import select, update
from sqlalchemy.ext.asyncio import AsyncSession

from models import QueueEntry, QueueStatus


class QueueRepository:
	def __init__(self, db: AsyncSession):
		self.db = db

	async def create_entry(
		self,
		event_id: int,
		user_id: int,
		queue_position: int,
	) -> QueueEntry:
		entry = QueueEntry(
			event_id=event_id,
			user_id=user_id,
			queue_position=queue_position,
		)
		self.db.add(entry)
		await self.db.flush()
		return entry

	async def get_entry(self, event_id: int, user_id: int) -> QueueEntry | None:
		stmt = select(QueueEntry).where(
			QueueEntry.event_id == event_id,
			QueueEntry.user_id == user_id,
		)
		result = await self.db.execute(stmt)
		return result.scalar_one_or_none()

	async def mark_admitted(self, entry_id: int) -> None:
		stmt = (
			update(QueueEntry)
			.where(QueueEntry.id == entry_id)
			.values(
				status=QueueStatus.ADMITTED,
				admitted_at=datetime.now(UTC),
			)
		)
		await self.db.execute(stmt)

	async def mark_completed(self, entry_id: int) -> None:
		stmt = update(QueueEntry).where(QueueEntry.id == entry_id).values(status=QueueStatus.COMPLETED)
		await self.db.execute(stmt)

	async def mark_abandoned(self, entry_id: int) -> None:
		stmt = update(QueueEntry).where(QueueEntry.id == entry_id).values(status=QueueStatus.ABANDONED)
		await self.db.execute(stmt)

	async def mark_expired(self, entry_id: int) -> None:
		stmt = update(QueueEntry).where(QueueEntry.id == entry_id).values(status=QueueStatus.EXPIRED)
		await self.db.execute(stmt)
