from redis.asyncio import Redis

from config import settings

redis_client: Redis | None = None  # type: ignore


async def get_redis() -> Redis:
	global redis_client
	if redis_client is None:
		redis_client = Redis.from_url(settings.REDIS_URL)
	return redis_client
