import asyncio
import logging

import httpx
from redis.asyncio import Redis

from config import settings
from core.inventory import InventoryStore
from database import Database
from repositories import ClaimsRepository, EventRepository
from repositories.payments import PaymentRepository, PaymentStatus
from services.claim_service import ClaimService
from services.payment_service import PaymentService

logger = logging.getLogger(__name__)

PAYSTACK_VERIFY_URL = 'https://api.paystack.co/transaction/verify'


async def reconcile_stale_payments(db: Database, inventory_store: InventoryStore):
	async with db.managed_session() as session:
		claims_repo = ClaimsRepository(session)
		events_repo = EventRepository(session)
		payments_repo = PaymentRepository(session)

		claim_service = ClaimService(claims_repo, inventory_store)
		payment_service = PaymentService(
			claims_repo=claims_repo,
			events_repo=events_repo,
			payments_repo=payments_repo,
			paystack_secret=settings.PAYSTACK_SECRET,
		)

		while True:
			stale_payments = await payments_repo.get_stale_payments(
				settings.PAYMENT_RECONCILIATION_WORKER_BATCH
			)

			if not stale_payments:
				break

			for payment in stale_payments:
				try:
					result = await verify_transaction(payment.payment_reference)

					if not result.get('status'):
						logger.warning(f'Verify failed: {payment.payment_reference}')
						continue

					paystack_status = result['data']['status']

					if paystack_status == 'success':
						await payment_service.process_successful_payment(payment.payment_reference)

					elif paystack_status in ('failed', 'abandoned'):
						await payments_repo.update_status(
							payment.payment_reference, PaymentStatus.FAILED
						)

						claim = await claims_repo.get(payment.claim_id)
						if claim:
							await claim_service.release_claim(claim.id, claim.event_id)

						logger.info(f'Payment failed: {payment.payment_reference}')

					else:
						logger.debug(f'Unhandled Paystack status: {paystack_status}')

				except Exception as e:
					logger.exception(f'Reconciliation error: {payment.payment_reference} - {e}')


async def verify_transaction(reference: str) -> dict:
	async with httpx.AsyncClient(timeout=10.0) as client:
		response = await client.get(
			f'{PAYSTACK_VERIFY_URL}/{reference}',
			headers={'Authorization': f'Bearer {settings.PAYSTACK_SECRET}'},
		)
		response.raise_for_status()
		return response.json()


async def reconcile_stale_payments_worker(db: Database, inventory_store: InventoryStore):
	while True:
		try:
			await reconcile_stale_payments(db, inventory_store)
		except Exception as e:
			logger.error(f'[ReconcileStalePaymentsWorker] Error: {e}')
		await asyncio.sleep(settings.PAYMENT_RECONCILIATION_WORKER_INTERVAL)


if __name__ == '__main__':

	async def main():
		redis = Redis.from_url(settings.REDIS_URL)
		db = Database(settings.DATABASE_URL)
		inventory_store = InventoryStore(redis)
		await reconcile_stale_payments_worker(db, inventory_store)

	asyncio.run(main())
