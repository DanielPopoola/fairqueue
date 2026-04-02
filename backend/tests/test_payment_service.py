import json
import httpx
import pytest
import respx
from unittest.mock import AsyncMock

from models import ClaimStatus, PaymentStatus
from services.payment_service import PaymentService
from tests.conftest import seed_claim, seed_event, seed_user


@pytest.mark.asyncio
@respx.mock
async def test_initialize_payment_standard_success(payment_service, db_session, monkeypatch):
    user = await seed_user(db_session)
    event = await seed_event(db_session)
    claim = await seed_claim(db_session, event.id, user.id, ClaimStatus.CLAIMED)

    monkeypatch.setattr('services.payment_service.uuid.uuid4', lambda: 'fixed-ref')

    route = respx.post(PaymentService.PAYSTACK_INITIALIZE_URL).mock(
        return_value=httpx.Response(
            200,
            json={
                'status': True,
                'data': {
                    'authorization_url': 'https://paystack.com/pay/test123',
					'reference': 'ps-ref',
                }
            }
        )
    )

    authorization_url = await payment_service.initialize_payment(claim.id, user.email)
    assert authorization_url == 'https://paystack.com/pay/test123'
    assert route.called
    
    payment = await payment_service.payments_repo.get_by_claim_id(claim.id)
    assert payment is not None
    assert payment.status == PaymentStatus.PENDING
    assert payment.authorization_url == 'https://paystack.com/pay/test123'
    assert payment.payment_reference == 'fixed-ref'
    
    updated_claim = await payment_service.claims_repo.get(claim.id)
    assert updated_claim is not None
    assert updated_claim.status == ClaimStatus.PAYMENT_PENDING


@pytest.mark.asyncio
@respx.mock
async def test_initialize_payment_idempotent_existing_payment(payment_service, db_session, monkeypatch):
    user = await seed_user(db_session)
    event = await seed_event(db_session)
    claim = await seed_claim(db_session, event.id, user.id, ClaimStatus.CLAIMED)

    await payment_service.payments_repo.initialize_payment(
        claim_id=claim.id,
        payment_reference='already-there',
        price=event.price_per_item,
        status=PaymentStatus.INITIALIZING
    )

    await payment_service.payments_repo.mark_pending(
        reference='already-there',
        authorization_url='https://paystack.com/pay/existing'
    )

    monkeypatch.setattr('services.payment_service.uuid.uuid4', lambda: 'unused-ref')
    paystack_route = respx.post(PaymentService.PAYSTACK_INITIALIZE_URL).mock(
		return_value=httpx.Response(200, json={'status': True, 'data': {}})
	)
    
    url = await payment_service.initialize_payment(claim.id, user.email)
    
    assert url == 'https://paystack.com/pay/existing'
    assert paystack_route.called is False


@pytest.mark.asyncio
async def test_initialize_payment_claim_not_found(payment_service):
	with pytest.raises(ValueError, match='Claim not found'):
		await payment_service.initialize_payment(999999, 'test@example.com')
            

@pytest.mark.asyncio
async def test_initialize_payment_claim_already_confirmed(payment_service, db_session):
	user = await seed_user(db_session)
	event = await seed_event(db_session)
	claim = await seed_claim(db_session, event.id, user.id, ClaimStatus.CONFIRMED)

	with pytest.raises(ValueError, match='Claim already confirmed'):
		await payment_service.initialize_payment(claim.id, user.email)
		

@pytest.mark.asyncio
async def test_initialize_payment_claim_released(payment_service, db_session):
	user = await seed_user(db_session)
	event = await seed_event(db_session)
	claim = await seed_claim(db_session, event.id, user.id, ClaimStatus.RELEASED)

	with pytest.raises(ValueError, match='Claim expired'):
		await payment_service.initialize_payment(claim.id, user.email)
		

@pytest.mark.asyncio
@respx.mock
async def test_initialize_payment_paystack_http_error(payment_service, db_session):
	user = await seed_user(db_session)
	event = await seed_event(db_session)
	claim = await seed_claim(db_session, event.id, user.id, ClaimStatus.CLAIMED)

	respx.post(PaymentService.PAYSTACK_INITIALIZE_URL).mock(side_effect=httpx.ConnectTimeout('timeout'))

	with pytest.raises(RuntimeError, match='Payment provider error'):
		await payment_service.initialize_payment(claim.id, user.email)
		

@pytest.mark.asyncio
@respx.mock
async def test_initialize_payment_paystack_logical_error(payment_service, db_session):
	user = await seed_user(db_session)
	event = await seed_event(db_session)
	claim = await seed_claim(db_session, event.id, user.id, ClaimStatus.CLAIMED)

	respx.post(PaymentService.PAYSTACK_INITIALIZE_URL).mock(
		return_value=httpx.Response(200, json={'status': False, 'message': 'fail'})
	)

	with pytest.raises(RuntimeError, match='Failed to initialize payment'):
		await payment_service.initialize_payment(claim.id, user.email)
		

@pytest.mark.asyncio
async def test_handle_webhook_valid_payment_confirmation(payment_service, db_session):
	user = await seed_user(db_session)
	event = await seed_event(db_session)
	claim = await seed_claim(db_session, event.id, user.id, ClaimStatus.PAYMENT_PENDING)
	
	await payment_service.payments_repo.initialize_payment(
        claim_id=claim.id,
        payment_reference='webhook-ref',
        price=event.price_per_item,
        status=PaymentStatus.INITIALIZING
    )
	await payment_service.payments_repo.mark_pending(
        reference='webhook-ref',
        authorization_url='https://paystack.com/pay/x'
    )

	payload = json.dumps({'event': 'charge.success', 'data': {'reference': 'webhook-ref'}}).encode()
	await payment_service.handle_webhook(payload)

	payment = await payment_service.payments_repo.get_payment_by_reference('webhook-ref')
	assert payment is not None
	assert payment.status == PaymentStatus.CONFIRMED
	updated_claim = await payment_service.claims_repo.get(claim.id)
	assert updated_claim is not None
	assert updated_claim.status == ClaimStatus.CONFIRMED


@pytest.mark.asyncio
async def test_handle_webhook_duplicate_webhook_idempotent(payment_service, db_session):
	user = await seed_user(db_session)
	event = await seed_event(db_session)
	claim = await seed_claim(db_session, event.id, user.id, ClaimStatus.PAYMENT_PENDING)
	await payment_service.payments_repo.initialize_payment(
        claim_id=claim.id,
        payment_reference='dupe-ref',
        price=event.price_per_item,
        status=PaymentStatus.INITIALIZING
    )
	await payment_service.payments_repo.update_status(
		reference='dupe-ref',
		status=PaymentStatus.CONFIRMED
    )

	update_spy = AsyncMock(wraps=payment_service.payments_repo.update_status)
	payment_service.payments_repo.update_status = update_spy

	payload = json.dumps({'event': 'charge.success', 'data': {'reference': 'dupe-ref'}}).encode()
	await payment_service.handle_webhook(payload)

	update_spy.assert_not_called()

@pytest.mark.asyncio
async def test_handle_webhook_irrelevant_event_ignored(payment_service, db_session):
	user = await seed_user(db_session)
	event = await seed_event(db_session)
	claim = await seed_claim(db_session, event.id, user.id, ClaimStatus.PAYMENT_PENDING)
	await payment_service.payments_repo.initialize_payment(
		claim_id=claim.id,
		payment_reference='ignore-ref',
		price=event.price_per_item,
		status=PaymentStatus.PENDING,
	)

	payload = json.dumps({'event': 'transfer.success', 'data': {'reference': 'ignore-ref'}}).encode()
	await payment_service.handle_webhook(payload)

	payment = await payment_service.payments_repo.get_payment_by_reference('ignore-ref')
	assert payment is not None
	assert payment.status == PaymentStatus.PENDING


@pytest.mark.asyncio
async def test_handle_webhook_unknown_reference_graceful(payment_service):
	payload = json.dumps({'event': 'charge.success', 'data': {'reference': 'unknown-ref'}}).encode()
	await payment_service.handle_webhook(payload)


@pytest.mark.asyncio
async def test_handle_webhook_malformed_json(payment_service):
	with pytest.raises(ValueError, match='Invalid JSON payload'):
		await payment_service.handle_webhook(b'not-json')


@pytest.mark.asyncio
async def test_http_client_timeout_configuration(payment_service):
	assert payment_service.client.timeout.connect == 10.0
	assert payment_service.client.timeout.read == 10.0
	assert payment_service.client.timeout.write == 10.0
	assert payment_service.client.timeout.pool == 10.0