import asyncio
import logging

from redis.asyncio import Redis

from config import settings
from core.inventory import InventoryStore
from database import Database
from models import EventStatus
from repositories import EventRepository

logger = logging.getLogger(__name__)

async def activate_pending_events(db: Database, inventory_store: InventoryStore):
	async with db.managed_session() as session:
		repo = EventRepository(session)
		events = await repo.get_events_ready_to_activate()

		for event in events:
			await inventory_store.initialize_event(event.id, event.total_inventory)
			await repo.update_status(event.id, EventStatus.ACTIVE)


async def event_activation_worker(db: Database, inventory_store: InventoryStore):
	while True:
		try:
			await activate_pending_events(db, inventory_store)
		except Exception as e:
			logger.debug(f'[EventActivationWorker] Error: {e}')
		await asyncio.sleep(settings.EVENT_ACTIVATION_WORKER_INTERVAL)


if __name__ == '__main__':

	async def main():
		redis = Redis.from_url(settings.REDIS_URL)
		db = Database(settings.DATABASE_URL)
		inventory_store = InventoryStore(redis)
		await event_activation_worker(db, inventory_store)

	asyncio.run(main())
