from collections.abc import AsyncGenerator
from typing import Annotated

from fastapi import Depends
from redis.asyncio import Redis
from sqlalchemy.ext.asyncio import AsyncSession

from core import InventoryStore, QueueService, WebhookQueue
from database import Database
from repositories import ClaimsRepository, EventRepository, QueueRepository
from services import ClaimService, EventService


class AppDependencies:
	db: Database | None = None
	redis_client: Redis | None = None


deps = AppDependencies()


async def get_db_session() -> AsyncGenerator[AsyncSession, None]:
	if deps.db is None:
		raise RuntimeError('Database is not initialized')

	async with deps.db.managed_session() as session:
		yield session


async def get_redis() -> Redis:
	if deps.redis_client is None:
		raise RuntimeError('Cache is not initialized')
	return deps.redis_client


async def get_inventory_store(cache: Annotated[Redis, Depends(get_redis)]):
	return InventoryStore(cache)


async def get_queue_service(cache: Annotated[Redis, Depends(get_redis)]):
	return QueueService(cache)


async def get_webhook_queue(queue: Annotated[Redis, Depends(get_redis)]):
	return WebhookQueue(queue)


async def get_claims_repository(
	session: Annotated[AsyncSession, Depends(get_db_session)],
) -> ClaimsRepository:
	return ClaimsRepository(session)


async def get_events_repository(
	session: Annotated[AsyncSession, Depends(get_db_session)],
) -> EventRepository:
	return EventRepository(session)


async def get_queue_repository(
	session: Annotated[AsyncSession, Depends(get_db_session)],
) -> QueueRepository:
	return QueueRepository(session)


async def get_claims_service(
	repository: Annotated[ClaimsRepository, Depends(get_claims_repository)],
	inventory_store: Annotated[InventoryStore, Depends(get_inventory_store)],
) -> ClaimService:
	return ClaimService(repository, inventory_store)


async def get_events_service(
	repository: Annotated[EventRepository, Depends(get_events_repository)],
) -> EventService:
	return EventService(repository)
