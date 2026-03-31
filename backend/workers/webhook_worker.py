import asyncio
import logging

from redis.asyncio import Redis

from config import settings
from core import WebhookQueue
from database import Database
from repositories import ClaimsRepository, EventRepository, PaymentRepository
from services import PaymentService

logger = logging.getLogger(__name__)


async def webhook_worker(db: Database, webhook_queue: WebhookQueue, paystack_secret: str):
	logger.info('[WebhookWorker] Started')
	while True:
		try:
			job = await webhook_queue.pop(timeout=5)
			if not job:
				await asyncio.sleep(0.1)
				continue

			async with db.managed_session() as session:
				service = PaymentService(
					claims_repo=ClaimsRepository(session),
					events_repo=EventRepository(session),
					payments_repo=PaymentRepository(session),
					paystack_secret=paystack_secret,
				)
				await service.handle_webhook(payload=job['payload'].encode())

		except Exception as e:
			logger.error(f'[WebhookWorker] Error processing webhook: {e}')


if __name__ == '__main__':

	async def main():
		redis = Redis.from_url(settings.REDIS_URL)
		db = Database(settings.DATABASE_URL)
		queue = WebhookQueue(redis)
		await webhook_worker(db, queue, settings.PAYSTACK_SECRET_KEY)

	asyncio.run(main())
