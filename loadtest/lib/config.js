export const BASE_URL = __ENV.BASE_URL || 'http://loadtest-app:8082';

export const DEFAULT_HEADERS = (token) => ({
  'Content-Type': 'application/json',
  Authorization: `Bearer ${token}`,
});

export function requireEnv(name) {
  const value = __ENV[name];
  if (!value) {
    throw new Error(`Missing required environment variable: ${name}`);
  }
  return value;
}