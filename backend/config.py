from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
	DATABASE_URL: str
	REDIS_URL: str

	PAYSTACK_SECRET: str

	CLAIM_TTL_SECONDS: int = 300
	CLAIM_EXPIRY_WORKER_INTERVAL: int = 60
	CLAIM_EXPIRY_WORKER_BATCH: int = 100
	EVENT_ACTIVATION_WORKER_INTERVAL: int = 3600

	APP_NAME: str = 'FairQueue API'

	LOG_LEVEL: str = 'INFO'

	model_config = SettingsConfigDict(env_file='.env', case_sensitive=False, extra='ignore')


settings = Settings()
