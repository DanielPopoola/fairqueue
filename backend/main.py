from contextlib import asynccontextmanager

from fastapi import FastAPI
from redis.asyncio import Redis

from api.routers import claims, events, queue, webhoooks
from config import settings
from database import Database
from dependencies import deps


@asynccontextmanager
async def lifespan(app: FastAPI):

	deps.db = Database(settings.DATABASE_URL)
	deps.redis_client = Redis.from_url(settings.REDIS_URL)
	yield
	await deps.redis_client.aclose()
	await deps.db.engine.dispose()


app = FastAPI(title=settings.APP_NAME, lifespan=lifespan)

app.include_router(events.router)
app.include_router(claims.router)
app.include_router(queue.router)
app.include_router(webhoooks.router)


@app.get('/info')
async def info():
	return {'app_name': settings.APP_NAME, 'description': 'The API for FairQueue'}
