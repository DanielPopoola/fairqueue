from contextlib import asynccontextmanager
from fastapi import FastAPI
from database import Database
from dependencies import deps
from config import settings
from redis.asyncio import Redis
from api.routers import events, claims, queue


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


@app.get('/info')
async def info():
	return {'app_name': settings.APP_NAME, 'description': 'The API for FairQueue'}
