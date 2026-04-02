from collections.abc import Sequence
from datetime import UTC, datetime, timedelta

from sqlalchemy import select, update
from sqlalchemy.ext.asyncio import AsyncSession

from models import Payment, PaymentStatus


class PaymentRepository:
	def __init__(self, db: AsyncSession):
		self.db = db

	async def initialize_payment(
		self, claim_id: int, payment_reference: str, price: int, status: PaymentStatus
	) -> Payment:
		payment = Payment(
			claim_id=claim_id, payment_reference=payment_reference, price=price, status=status
		)
		self.db.add(payment)
		await self.db.flush()
		return payment

	async def mark_pending(
		self,
		reference: str,
		authorization_url: str,
	) -> None:
		stmt = (
			update(Payment)
			.where(
				Payment.payment_reference == reference,
				Payment.status == PaymentStatus.INITIALIZING,
			)
			.values(
				authorization_url=authorization_url,
				status=PaymentStatus.PENDING,
			)
		)
		result = await self.db.execute(stmt)

		if result.rowcount == 0:
			raise ValueError('Payment not found or invalid state transition')

	async def get_by_claim_id(self, claim_id: int) -> Payment | None:
		stmt = select(Payment).where(Payment.claim_id == claim_id)
		result = await self.db.execute(stmt)
		return result.scalar_one_or_none()

	async def get_payment_by_reference(self, reference: str) -> Payment | None:
		stmt = select(Payment).where(Payment.payment_reference == reference)
		result = await self.db.execute(stmt)
		return result.scalar_one_or_none()

	async def update_status(self, reference: str, status: PaymentStatus) -> None:
		stmt = update(Payment).where(Payment.payment_reference == reference).values(status=status)
		await self.db.execute(stmt)

	async def get_stale_payments(self, batch_size: int) -> Sequence[Payment]:
		stmt = (
			select(Payment)
			.where(Payment.status.in_([PaymentStatus.INITIALIZING, PaymentStatus.PENDING]))
			.where(Payment.updated_at < datetime.now(UTC) - timedelta(minutes=10))
			.limit(batch_size)
		)
		result = await self.db.execute(stmt)
		return result.scalars().all()
