from sqlalchemy import select, update
from sqlalchemy.ext.asyncio import AsyncSession

from models import Payment, PaymentStatus


class PaymentRepository:
	def __init__(self, db: AsyncSession):
		self.db = db

	async def create_payment(
		self, claim_id: int, payment_reference: str, price: int, status: PaymentStatus
	) -> Payment:
		payment = Payment(
			claim_id=claim_id, payment_reference=payment_reference, price=price, status=status
		)
		self.db.add(payment)
		await self.db.flush()
		return payment

	async def get_payment_by_reference(self, reference: str) -> Payment | None:
		stmt = select(Payment).where(Payment.payment_reference == reference)
		result = await self.db.execute(stmt)
		return result.scalar_one_or_none()

	async def update_status(self, reference: str, status: PaymentStatus) -> None:
		stmt = update(Payment).where(Payment.payment_reference == reference).values(status=status)
		await self.db.execute(stmt)
