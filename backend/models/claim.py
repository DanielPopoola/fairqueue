from datetime import UTC, datetime
from enum import StrEnum

from sqlalchemy import DateTime, ForeignKey, Index, Integer
from sqlalchemy import Enum as SQLEnum
from sqlalchemy.orm import Mapped, mapped_column

from database import Base


class ClaimStatus(StrEnum):
	CLAIMED = 'claimed'
	PAYMENT_PENDING = 'payment_pending'
	CONFIRMED = 'confirmed'
	RELEASING = "releasing"
	RELEASED = 'released'


class Claim(Base):
	__tablename__ = 'claims'

	id: Mapped[int] = mapped_column(Integer, primary_key=True)
	event_id: Mapped[int] = mapped_column(ForeignKey('events.id'), index=True)
	item_id: Mapped[int] = mapped_column(ForeignKey('inventory_items.id'), index=True)
	user_id: Mapped[int] = mapped_column(ForeignKey('users.id'), index=True)
	status: Mapped[str] = mapped_column(SQLEnum(ClaimStatus), default=ClaimStatus.CLAIMED)
	claimed_at: Mapped[datetime] = mapped_column(
		DateTime(timezone=True), default=lambda: datetime.now(UTC)
	)
	confirmed_at: Mapped[datetime | None] = mapped_column(DateTime(timezone=True), nullable=True)
	expires_at: Mapped[datetime | None] = mapped_column(DateTime(timezone=True), nullable=True)

	__table_args__ = (
		# background job query: "find all unconfirmed claims that have expired"
		# partial: skips confirmed/released rows which are never scanned by expiry job
		Index(
			'ix_claims_active_expiry',
			'expires_at',
			postgresql_where=(status.in_(['claimed', 'payment_pending'])),
		),
		# dashboard query: "how many confirmed claims for event 123?"
		Index('ix_claims_event_status', 'event_id', 'status'),
	)
