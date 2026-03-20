from pydantic import BaseModel
from models import QueueStatus


class JoinQueueRequest(BaseModel):
    event_id: int
    user_id: int


class QueuePositionResponse(BaseModel):
    position: int
    status: QueueStatus
    event_id: int
    user_id: int


class JoinQueueResponse(BaseModel):
    position: int
    event_id: int
    user_id: int