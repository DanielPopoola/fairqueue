from typing import Annotated

from fastapi import APIRouter, Depends, HTTPException

from api.schemas import ClaimResponse, CreateClaimRequest
from dependencies import get_claims_service
from services import ClaimService

router = APIRouter(prefix='/claims', tags=['claims'])


@router.post('/', response_model=ClaimResponse, status_code=201)
async def create_claim(
	body: CreateClaimRequest,
	service: Annotated[ClaimService, Depends(get_claims_service)],
):
	try:
		claim = await service.create_claim(event_id=body.event_id, user_id=body.user_id)
		return ClaimResponse.model_validate(claim)
	except ValueError as e:
		raise HTTPException(status_code=409, detail=str(e)) from e
