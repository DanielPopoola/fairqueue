from datetime import datetime, UTC
from enum import StrEnum


from sqlalchemy import DateTime, ForeignKey, Integer, String
from sqlalchemy import Enum as SQLEnum
from sqlalchemy.orm import Mapped, mapped_column

from database import Base


class PaymentStatus(StrEnum):
	PENDING = 'pending'
	CONFIRMED = 'confirmed'
	FAILED = 'failed'
	


class Payment(Base):
	__tablename__ = 'payments'

	id: Mapped[int] = mapped_column(Integer, primary_key=True)
	claim_id: Mapped[int] = mapped_column(ForeignKey('claims.id'), index=True)
	payment_reference: Mapped[str] = mapped_column(String, unique=True, index=True)
	price: Mapped[int] = mapped_column(Integer)  # price in kobo (smallest unit, avoids decimals)
	status: Mapped[PaymentStatus] = mapped_column(SQLEnum(PaymentStatus, name="payment_status"), default=PaymentStatus.PENDING)
	created_at: Mapped[datetime] = mapped_column(
		DateTime(timezone=True), default=lambda: datetime.now(UTC)
	)
	updated_at: Mapped[datetime] = mapped_column(
		DateTime(timezone=True), default=lambda: datetime.now(UTC), onupdate=lambda: datetime.now(UTC)
    )
