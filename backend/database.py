from contextlib import asynccontextmanager

from sqlalchemy.ext.asyncio import AsyncSession, async_sessionmaker, create_async_engine
from sqlalchemy.orm import DeclarativeBase


class Base(DeclarativeBase):
	pass


class Database:
	def __init__(self, db_url: str):
		self.engine = create_async_engine(db_url)
		self.session_factory = async_sessionmaker(
			self.engine, class_=AsyncSession, autoflush=False, expire_on_commit=False
		)

	@asynccontextmanager
	async def managed_session(self):
		async with self.session_factory() as session:
			try:
				yield session
				await session.commit()
			except Exception:
				await session.rollback()
				raise
			finally:
				await session.close()
