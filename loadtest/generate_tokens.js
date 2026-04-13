import fs from 'node:fs';

const BASE_URL = process.env.BASE_URL;
const TOTAL_USERS = Number(process.env.TOTAL_USERS || '20000');

if (!BASE_URL) {
  throw new Error('BASE_URL is required');
}

async function generate() {
  const tokens = [];

  for (let i = 0; i < TOTAL_USERS; i += 1) {
    const email = `loadtest-${i}@test.com`;

    await fetch(`${BASE_URL}/auth/customer/otp/request`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email }),
    });

    const otpRes = await fetch(`${BASE_URL}/loadtest/otp?email=${encodeURIComponent(email)}`);
    const otpData = await otpRes.json();

    const authRes = await fetch(`${BASE_URL}/auth/customer/otp/verify`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, otp: otpData.otp }),
    });

    const authData = await authRes.json();
    if (!authData.token) {
      throw new Error(`No token returned for ${email}`);
    }

    tokens.push(authData.token);

    if (i > 0 && i % 1000 === 0) {
      console.log(`Generated ${i} tokens`);
    }
  }

  fs.writeFileSync('tokens.json', JSON.stringify(tokens, null, 2));
  console.log(`Wrote ${tokens.length} tokens to tokens.json`);
}

generate().catch((err) => {
  console.error(err);
  process.exit(1);
});
