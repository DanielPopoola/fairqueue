import hashlib
import hmac
from typing import Annotated

from fastapi import APIRouter, Depends, Header, HTTPException, Request

from config import settings
from core import WebhookQueue
from dependencies import get_webhook_queue

router = APIRouter(prefix='/webhooks', tags=['webhooks'])


@router.post('/paystack')
async def paystack_webhook(
	request: Request,
	queue: Annotated[WebhookQueue, Depends(get_webhook_queue)],
	x_paystack_signature: str = Header(...),
):
	payload = await request.body()

	expected = hmac.new(
		settings.PAYSTACK_SECRET.encode(),
		payload,
		hashlib.sha512,
	).hexdigest()

	if not hmac.compare_digest(expected, x_paystack_signature):
		raise HTTPException(status_code=400, detail='Invalid signature')

	await queue.push(payload)

	return {'status': 'ok'}
