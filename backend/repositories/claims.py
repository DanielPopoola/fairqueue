from datetime import UTC, datetime

from sqlalchemy import update
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.future import select

from models import Claim, ClaimStatus


class ClaimsRepository:
	def __init__(self, db: AsyncSession):
		self.db = db

	async def get_expired_active_claims(self) -> list[Claim]:
		now = datetime.now(UTC)
		stmt = (
			select(Claim)
			.where(Claim.status.in_(['claimed', 'payment_pending']))
			.where(Claim.expires_at < now)
		)
		result = await self.db.execute(stmt)
		return result.scalars().all()

	async def get_expired_active_claims_batch(self, batch_size: int, offset: int = 0) -> list[Claim]:
		now = datetime.now(UTC)
		stmt = (
			select(Claim)
			.where(Claim.status.in_(['claimed', 'payment_pending']))
			.where(Claim.expires_at < now)
			.limit(batch_size)
			.offset(offset)
		)
		result = await self.db.execute(stmt)
		return result.scalars().all()

	async def update_status(self, claim_id: int, status: ClaimStatus) -> None:
		stmt = update(Claim).where(Claim.id == claim_id).values(status=status)
		await self.db.execute(stmt)

	async def update_status_batch(self, claim_ids: list[int], status: ClaimStatus) -> None:
		if not claim_ids:
			return
		stmt = update(Claim).where(Claim.id.in_(claim_ids)).values(status=status)
		await self.db.execute(stmt)
