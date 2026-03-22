"""add partial unique constraint on active claims

Revision ID: 83c9f513abeb
Revises: 9a954a89f28b
Create Date: 2026-03-22 20:01:42.720439

"""
from typing import Sequence, Union

from alembic import op
import sqlalchemy as sa
from sqlalchemy.dialects import postgresql

# revision identifiers, used by Alembic.
revision: str = '83c9f513abeb'
down_revision: Union[str, Sequence[str], None] = '9a954a89f28b'
branch_labels: Union[str, Sequence[str], None] = None
depends_on: Union[str, Sequence[str], None] = None


def upgrade() -> None:
    """Upgrade schema."""
    op.create_index(
        'uq_claims_active_user_event',
        'claims',
        ['user_id', 'event_id'],
        unique=True,
        postgresql_where=sa.text("status IN ('CLAIMED', 'PAYMENT_PENDING')")
    )


def downgrade() -> None:
    """Downgrade schema."""
    op.drop_index(
        'uq_claims_active_user_event',
        table_name='claims',
        postgresql_where=sa.text("status IN ('CLAIMED', 'PAYMENT_PENDING')")
    )