from datetime import datetime
from pydantic import BaseModel
from models import ClaimStatus


class CreateClaimRequest(BaseModel):
    event_id: int
    user_id: int


class ClaimResponse(BaseModel):
    id: int
    event_id: int
    user_id: int
    status: ClaimStatus
    expires_at: datetime | None

    model_config = {"from_attributes": True}