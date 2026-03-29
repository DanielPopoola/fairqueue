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
		"""Atomically claim one ticket. Returns True if successful."""
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

		result = await self.redis.eval(lua_script, 1, key)  # pyright: ignore[reportGeneralTypeIssues]
		return bool(result)

	async def release(self, event_id: int, claim_id: int) -> bool:
		"""
		Idempotently release one ticket back to inventory.
		Returns True if actually released, False if already released.
		"""
		counter_key = f'event:{event_id}:available'
		released_set_key = f'event:{event_id}:released'

		lua_script = """
		local counter_key = KEYS[1]
		local released_set_key = KEYS[2]
		local claim_id = ARGV[1]

		-- Check if already released
		local already_released = redis.call('SISMEMBER', released_set_key, claim_id)
		if already_released == 1 then
			return 0 -- Already released, idempotent
		end

		-- Release it
		redis.call('INCR', counter_key)
		redis.call('SADD', released_set_key, claim_id)
		return 1 -- Successfully released
		"""

		result = await self.redis.eval(lua_script, 2, counter_key, released_set_key, claim_id)
		return bool(result)

	async def release_batch(self, claims: list[tuple[int, int]]) -> int:
		"""
		Idempotently release multiple tickets.
		Returns count of actually released tickets (skips already-released).
		"""
		lua_script = """
        local released_count = 0
        for i = 1, #ARGV, 2 do
            local claim_id = ARGV[i]
            local event_id = ARGV[i+1]
            
            local counter_key = "event:" .. event_id .. ":available"
            local released_set_key = "event:" .. event_id .. ":released"
            
            -- Check if already released
            local already_released = redis.call('SISMEMBER', released_set_key, claim_id)
            if already_released == 0 then
                -- Not released yet, release now
                redis.call('INCR', counter_key)
                redis.call('SADD', released_set_key, claim_id)
                released_count = released_count + 1
            end
        end
        return released_count
        """

		# Flatten: [(claim_id, event_id), ...] → [claim_id, event_id, claim_id, event_id, ...]
		argv = [str(x) for pair in claims for x in pair]

		num_released = await self.redis.eval(lua_script, 0, *argv)
		return int(num_released)

	async def available_count(self, event_id: int) -> int:
		key = f'event:{event_id}:available'
		value = await self.redis.get(key)
		if not value:
			return 0
		return int(value.decode('utf-8'))
