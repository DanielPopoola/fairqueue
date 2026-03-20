from redis.asyncio import Redis
from sqlalchemy.util.typing import final


class InventoryStore:
	@final
	def __init__(self, redis: Redis):
		self.redis = redis

	async def initialize_event(self, event_id: int, total_inventory: int) -> None:
		key = f'event:{event_id}:available'
		await self.redis.set(key, total_inventory, nx=True)

	async def claim(self, event_id: int) -> bool:
		key = f'event:{event_id}:available'

		lua_script = """
        local counter_key = KEYS[1]
        local count = tonumber(redis.call('GET', counter_key))
        if count and count > 0 then
            redis.call('DECR', counter_key)
            return 1
        else
            return 0
        end
        """

		result = await self.redis.eval(lua_script, 1, key) # pyright: ignore[reportGeneralTypeIssues]
		return bool(result)

	async def release(self, event_id: int) -> None:
		key = f'event:{event_id}:available'
		await self.redis.incr(key)

	async def release_batch(self, claims: list[tuple[int, int]]) -> int:
		lua_script = """
        for i = 1, #ARGV, 2 do
            local claim_id = ARGV[i]
            local event_id = ARGV[i+1]
            local key = "event:" .. event_id .. ":available"
            redis.call("INCR", key)
        end
        return #ARGV / 2
        """
		# Flatten ARGV
		argv = [str(x) for pair in claims for x in pair]
		num_released = await self.redis.eval(lua_script, 0, *argv) # pyright: ignore[reportGeneralTypeIssues]
		return num_released # pyright: ignore[reportReturnType]

	async def available_count(self, event_id: int) -> int:
		key = f'event:{event_id}:available'
		value = await self.redis.get(key)
		if not value:
			return 0
		return int(value.decode('utf-8'))
