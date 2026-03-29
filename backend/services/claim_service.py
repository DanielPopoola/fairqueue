from datetime import UTC, datetime, timedelta

from sqlalchemy.exc import IntegrityError

from config import settings
from core.inventory import InventoryStore
from models import Claim, ClaimStatus
from repositories import ClaimsRepository


class ClaimService:
	def __init__(self, claims_repo: ClaimsRepository, inventory_store: InventoryStore):
		self.claims_repo = claims_repo
		self.inventory_store = inventory_store

	async def create_claim(self, event_id: int, user_id: int) -> Claim:
		existing = await self.claims_repo.get_by_user_and_event(user_id=user_id, event_id=event_id)
		if existing and existing.status in (ClaimStatus.CLAIMED, ClaimStatus.PAYMENT_PENDING):
			raise ValueError('User already has active claim')

		success = await self.inventory_store.claim(event_id)
		if not success:
			raise ValueError('Sold out')

		expires_at = datetime.now(UTC) + timedelta(seconds=settings.CLAIM_TTL_SECONDS)

		try:
			claim = await self.claims_repo.create_claim(
				event_id=event_id, user_id=user_id, expires_at=expires_at
			)
		except IntegrityError as e:
			if 'uq_claims_active_user_event' in str(e.orig):
				await self.inventory_store.release(event_id)
				raise ValueError('User already has active claim') from e
			raise

		return claim

	async def release_claim(self, claim_id: int, event_id: int) -> None:
		claim = await self.claims_repo.get(claim_id)
		if not claim:
			return

		if claim.event_id != event_id:
			return

		if claim.status in (
			ClaimStatus.RELEASED,
			ClaimStatus.CONFIRMED,
		):
			return

		acquired = await self.claims_repo.try_mark_releasing(claim_id)
		if not acquired:
			return

		await self.inventory_store.release(event_id)

		await self.claims_repo.update_status(claim_id, ClaimStatus.RELEASED)
