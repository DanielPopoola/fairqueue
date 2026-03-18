from datetime import UTC, datetime
from enum import StrEnum

from sqlalchemy import DateTime, Integer, String
from sqlalchemy import Enum as SQLEnum
from sqlalchemy.orm import Mapped, mapped_column

from database import Base


class AllocationStrategy(StrEnum):
	FIFO = 'fifo'
	LOTTERY = 'lottery'
	WEIGHTED = 'weighted'


class EventStatus(StrEnum):
	ACTIVE = 'active'
	UPCOMING = 'upcoming'
	ENDED = 'ended'
	SOLDOUT = 'soldout'


class Event(Base):
	__tablename__ = 'events'

	id: Mapped[int] = mapped_column(Integer, primary_key=True, index=True)
	organizer_id: Mapped[int] = mapped_column(Integer, index=True)
	name: Mapped[str] = mapped_column(String)
	total_inventory: Mapped[int] = mapped_column(Integer)
	sale_start: Mapped[datetime] = mapped_column(DateTime(timezone=True))
	sale_end: Mapped[datetime] = mapped_column(DateTime(timezone=True))
	allocation_strategy: Mapped[str] = mapped_column(
		SQLEnum(AllocationStrategy), default=AllocationStrategy.FIFO
	)
	price_per_item: Mapped[int] = mapped_column(Integer)
	status: Mapped[str] = mapped_column(SQLEnum(EventStatus), default=EventStatus.UPCOMING)
	created_at: Mapped[datetime] = mapped_column(
		DateTime(timezone=True), default=lambda: datetime.now(UTC)
	)
