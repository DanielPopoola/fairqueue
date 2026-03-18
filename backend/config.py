from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
	DATABASE_URL: str
	REDIS_URL: str

	APP_NAME: str = 'FairQueue API'

	LOG_LEVEL: str = 'INFO'

	model_config = SettingsConfigDict(env_file='.env', case_sensitive=False, extra='ignore')


settings = Settings()
