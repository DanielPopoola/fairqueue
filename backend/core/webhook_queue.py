import json

from redis.asyncio import Redis

WEBHOOK_QUEUE_KEY = 'queue:webhooks'


class WebhookQueue:
	def __init__(self, redis: Redis):
		self.redis = redis

	async def push(self, payload: bytes) -> None:
		await self.redis.rpush(WEBHOOK_QUEUE_KEY, json.dumps({'payload': payload.decode()}))

	async def pop(self, timeout: int = 5) -> dict | None:
		result = await self.redis.blpop(WEBHOOK_QUEUE_KEY, timeout=timeout)
		if not result:
			return None
		_, raw = result
		return json.loads(raw)
