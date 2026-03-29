from datetime import UTC, datetime

from redis.asyncio import Redis


class QueueService:
	def __init__(self, redis: Redis):
		self.redis = redis

	async def join_queue(self, event_id: int, user_id: int) -> int:
		key = f'queue:event:{event_id}'
		score = int(datetime.now(UTC).timestamp() * 1000)
		await self.redis.zadd(key, {user_id: score}, nx=True)  # pyright: ignore[reportArgumentType]
		position = await self.redis.zrank(key, user_id)
		return position + 1

	async def admit_next(self, event_id: int, count: int) -> list[str]:
		key = f'queue:event:{event_id}'
		lua_script = """
        local key = KEYS[1]
        local count = tonumber(ARGV[1])
        local users = redis.call('ZRANGE', key, 0, count - 1)
        if #users > 0 then
            redis.call('ZREM', key, unpack(users))
        end
        return users
        """
		result = await self.redis.eval(lua_script, 1, key, count)  # pyright: ignore[reportGeneralTypeIssues]
		return [user.decode('utf-8') for user in result]  # pyright: ignore[reportAttributeAccessIssue]

	async def get_position(self, event_id: int, user_id: int) -> int | None:
		key = f'queue:event:{event_id}'
		position = await self.redis.zrank(key, user_id)
		if position is None:
			return None
		return position + 1
