import asyncio

from config import settings
from core.inventory import InventoryStore
from database import Database
from repositories.claims import ClaimsRepository, ClaimStatus


async def process_expired_claims(db: Database, inventory_store: InventoryStore):
	async with db.managed_session() as session:
		repo = ClaimsRepository(session)
		offset = 0

		while True:
			batch = await repo.get_expired_active_claims_batch(
				settings.CLAIM_EXPIRY_WORKER_BATCH
			)
			if not batch:
				break

			claim_ids = [c.id for c in batch]
			claims_to_release = [(c.id, c.event_id) for c in batch]

			# PostgreSQL first (source of truth)
			await repo.update_status_batch(claim_ids, ClaimStatus.RELEASED)

			# Redis second (recoverable if this fails)
			await inventory_store.release_batch(claims_to_release)

			offset += settings.CLAIM_EXPIRY_WORKER_BATCH


async def claim_expiry_worker(db: Database, inventory_store: InventoryStore):
	while True:
		try:
			await process_expired_claims(db, inventory_store)
		except Exception as e:
			print(f'[ClaimExpiryWorker] Error: {e}')
		await asyncio.sleep(settings.CLAIM_EXPIRY_WORKER_INTERVAL)


if __name__ == '__main__':
	from dependencies import get_redis

	async def main():
		redis = await get_redis()
		db = Database(settings.DATABASE_URL)
		inventory_store = InventoryStore(redis)
		await claim_expiry_worker(db, inventory_store)

	asyncio.run(main())
