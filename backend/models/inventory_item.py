from datetime import UTC, datetime
from enum import StrEnum

from sqlalchemy import DateTime, ForeignKey, Index, Integer, String
from sqlalchemy import Enum as SQLEnum
from sqlalchemy.orm import Mapped, mapped_column

from database import Base


class ItemStatus(StrEnum):
	AVAILABLE = 'available'
	CLAIMED = 'claimed'
	CONFIRMED = 'confirmed'
	RELEASED = 'released'


class InventoryItem(Base):
	__tablename__ = 'inventory_items'

	id: Mapped[int] = mapped_column(Integer, primary_key=True)
	event_id: Mapped[int] = mapped_column(ForeignKey('events.id'), index=True)
	item_type: Mapped[str] = mapped_column(String)
	price: Mapped[int] = mapped_column(Integer)  # price in kobo (smallest unit, avoids decimals)
	status: Mapped[str] = mapped_column(SQLEnum(ItemStatus), default=ItemStatus.AVAILABLE)
	created_at: Mapped[datetime] = mapped_column(
		DateTime(timezone=True), default=lambda: datetime.now(UTC)
	)

	__table_args__ = (
		# composite: "give me available items for event 123"
		Index('ix_inventory_event_status', 'event_id', 'status'),
		# partial: only index available items — confirmed ones are never queried
		Index('ix_inventory_available', 'event_id', postgresql_where=(status == 'available')),
	)
