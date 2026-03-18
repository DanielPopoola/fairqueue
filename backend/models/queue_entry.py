from datetime import UTC, datetime
from enum import StrEnum

from sqlalchemy import DateTime, ForeignKey, Index, Integer
from sqlalchemy import Enum as SQLEnum
from sqlalchemy.orm import Mapped, mapped_column

from database import Base


class QueueStatus(StrEnum):
	WAITING = 'waiting'
	ADMITTED = 'admitted'
	COMPLETED = 'completed'
	ABANDONED = 'abandoned'
	EXPIRED = 'expired'


class QueueEntry(Base):
	__tablename__ = 'queue_entries'

	id: Mapped[int] = mapped_column(Integer, primary_key=True)
	event_id: Mapped[int] = mapped_column(ForeignKey('events.id'), index=True)
	user_id: Mapped[int] = mapped_column(ForeignKey('users.id'), index=True)
	queue_position: Mapped[int] = mapped_column(Integer)
	status: Mapped[str] = mapped_column(SQLEnum(QueueStatus), default=QueueStatus.WAITING)
	joined_at: Mapped[datetime] = mapped_column(
		DateTime(timezone=True), default=lambda: datetime.now(UTC)
	)
	admitted_at: Mapped[datetime | None] = mapped_column(DateTime(timezone=True), nullable=True)
	expires_at: Mapped[datetime | None] = mapped_column(DateTime(timezone=True), nullable=True)

	__table_args__ = (
		# admission job query: "give me next N waiting users for event 123 ordered by position"
		Index('ix_queue_event_status_position', 'event_id', 'status', 'queue_position'),
		# partial: only waiting/admitted rows matter — completed ones are never queried again
		Index(
			'ix_queue_active',
			'event_id',
			'queue_position',
			postgresql_where=(status.in_(['waiting', 'admitted'])),
		),
	)
