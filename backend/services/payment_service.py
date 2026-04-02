import json
import logging
import uuid

import httpx

from models import ClaimStatus, PaymentStatus
from repositories import ClaimsRepository, EventRepository, PaymentRepository

logger = logging.getLogger(__name__)


class PaymentService:
	PAYSTACK_INITIALIZE_URL = 'https://api.paystack.co/transaction/initialize'

	def __init__(
		self,
		claims_repo: ClaimsRepository,
		events_repo: EventRepository,
		payments_repo: PaymentRepository,
		paystack_secret: str,
	):
		self.claims_repo = claims_repo
		self.events_repo = events_repo
		self.payments_repo = payments_repo
		self.paystack_secret = paystack_secret
		self.client = httpx.AsyncClient(timeout=10.0)

	async def initialize_payment(self, claim_id: int, user_email: str) -> str:
		claim = await self._get_valid_claim(claim_id)

		existing_payment = await self.payments_repo.get_by_claim_id(claim_id)
		if existing_payment:
			return existing_payment.authorization_url

		event = await self._get_event(claim.event_id)

		reference = str(uuid.uuid4())

		await self.payments_repo.initialize_payment(
			claim_id=claim_id,
			payment_reference=reference,
			price=event.price_per_item,
			status=PaymentStatus.INITIALIZING,
		)

		response_data = await self._initialize_paystack_payment(
			reference=reference,
			email=user_email,
			amount=event.price_per_item,
		)
		authorization_url = response_data['authorization_url']

		await self.payments_repo.mark_pending(
			reference=reference,
			authorization_url=authorization_url,
		)

		await self.claims_repo.update_status(claim_id, ClaimStatus.PAYMENT_PENDING)

		return authorization_url

	async def handle_webhook(self, payload: bytes) -> None:
		data = self._parse_payload(payload)

		if data.get('event') != 'charge.success':
			return

		payment_data = data.get('data', {})
		reference = payment_data.get('reference')

		await self.process_successful_payment(reference)

		return
	
	async def process_successful_payment(self, reference: str) -> None:
		payment = await self.payments_repo.get_payment_by_reference(reference)
		if not payment:
			return

		if payment.status == PaymentStatus.CONFIRMED:
			return

		claim = await self.claims_repo.get(payment.claim_id)
		if not claim:
			return

		if claim.status == ClaimStatus.PAYMENT_PENDING:
			await self.payments_repo.update_status(reference, PaymentStatus.CONFIRMED)
			await self.claims_repo.update_status(claim.id, ClaimStatus.CONFIRMED)
			return

		if claim.status == ClaimStatus.RELEASED:
			await self.payments_repo.update_status(reference, PaymentStatus.FAILED)
			logger.error(f"MANUAL REFUND NEEDED: reference={reference}")

	async def _get_valid_claim(self, claim_id: int):
		claim = await self.claims_repo.get(claim_id)
		if not claim:
			raise ValueError('Claim not found')

		if claim.status == ClaimStatus.CONFIRMED:
			raise ValueError('Claim already confirmed')

		if claim.status == ClaimStatus.RELEASED:
			raise ValueError('Claim expired')

		return claim

	async def _get_event(self, event_id: int):
		event = await self.events_repo.get_event(event_id)
		if not event:
			raise ValueError('Event not found')
		return event

	async def _initialize_paystack_payment(self, reference: str, amount: int, email: str):
		try:
			response = await self.client.post(
				self.PAYSTACK_INITIALIZE_URL,
				json={
					'amount': amount,
					'email': email,
					'reference': reference,
				},
				headers={
					'Authorization': f'Bearer {self.paystack_secret}',
					'Content-Type': 'application/json',
				},
			)
			response.raise_for_status()
		except httpx.HTTPError as e:
			raise RuntimeError('Payment provider error') from e

		data = response.json()

		if not data.get('status'):
			raise RuntimeError('Failed to initialize payment')

		return data['data']

	def _parse_payload(self, payload: bytes) -> dict:
		try:
			return json.loads(payload)
		except json.JSONDecodeError as e:
			raise ValueError('Invalid JSON payload') from e
