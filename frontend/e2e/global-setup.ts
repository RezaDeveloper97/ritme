import { E2E } from './support/env';
import { readOtp } from './support/stack';

/**
 * The harness boots the web app (playwright.config webServer) but EXPECTS the API and its private database to be
 * running — fail fast with the recipe instead of a timeout in the middle of a journey.
 */
export default async function globalSetup(): Promise<void> {
  const up = await fetch(`${E2E.apiUrl}/up`).catch(() => null);
  if (!up?.ok) {
    throw new Error(
      `backend-go is not answering on ${E2E.apiUrl}/up. Start it on the private database with SMS_PROVIDER=log, ` +
        `CYCLE_ENGINE=v6 and CORS_ALLOWED_ORIGINS=http://localhost:${E2E.webPort} (frontend/CLAUDE.md, Testing).`,
    );
  }
  try {
    readOtp('09000000000');
  } catch (e) {
    if (!/no OTP row/.test(String(e))) {
      throw new Error(`cannot query database "${E2E.dbName}" via E2E_DB_CLI: ${String(e).split('\n')[0]}`);
    }
  }
}
