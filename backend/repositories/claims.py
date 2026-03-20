from collections.abc import Sequence
from datetime import UTC, datetime

from sqlalchemy import select, update
from sqlalchemy.ext.asyncio import AsyncSession

from models import Claim, ClaimStatus


class ClaimsRepository:
	def __init__(self, db: AsyncSession):
		self.db = db

	async def create_claim(
		self,
		event_id: int,
		user_id: int,
		item_id: int | None = None,
		expires_at: datetime | None = None,
	) -> Claim:
		claim = Claim(
			event_id=event_id,
			user_id=user_id,
			item_id=item_id,
			expires_at=expires_at,
			status=ClaimStatus.CLAIMED,
		)
		self.db.add(claim)
		await self.db.flush()
		return claim

	async def get(self, claim_id: int) -> Claim | None:
		stmt = select(Claim).where(Claim.id == claim_id)
		result = await self.db.execute(stmt)
		return result.scalar_one_or_none()

	async def get_by_user_and_event(self, user_id: int, event_id: int) -> Claim | None:
		stmt = select(Claim).where(
			Claim.user_id == user_id,
			Claim.event_id == event_id,
		)
		result = await self.db.execute(stmt)
		return result.scalar_one_or_none()

	async def get_expired_active_claims(self) -> Sequence[Claim]:
		now = datetime.now(UTC)
		stmt = (
			select(Claim)
			.where(Claim.status.in_([ClaimStatus.CLAIMED, ClaimStatus.PAYMENT_PENDING]))
			.where(Claim.expires_at < now)
		)
		result = await self.db.execute(stmt)
		return result.scalars().all()

	async def get_expired_active_claims_batch(self, batch_size: int) -> Sequence[Claim]:
		stmt = (
			select(Claim)
			.where(
				Claim.status.in_(
					[ClaimStatus.CLAIMED, ClaimStatus.PAYMENT_PENDING, ClaimStatus.RELEASING]
				)
			)
			.where(Claim.expires_at < datetime.now(UTC))
			.limit(batch_size)
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
