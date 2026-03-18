import logging

from fastapi import FastAPI

from .config import settings

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


app = FastAPI(title=settings.APP_NAME)


@app.get('/info')
async def info():
	return {'app_name': settings.APP_NAME, 'description': 'The API for FairQueue'}
