/**
 * Environment of the smoke harness. Every value has a local default; see frontend/CLAUDE.md for the recipe.
 *
 *   E2E_API_URL          backend-go base (no /api/v1)               default http://127.0.0.1:8020
 *   E2E_WEB_PORT         port `next dev` is started on              default 3100
 *   E2E_BASE_URL         use an already running web app instead of starting `next dev`
 *   E2E_DB_NAME          the PRIVATE database the API runs on       default ritme_e2e  (never ritme_test)
 *   E2E_DB_CLI           command prefix that runs the `mariadb` client, default: the docker test stack
 *   E2E_BROWSER_CHANNEL  playwright channel; default `chrome` locally, Playwright's chromium on CI
 */
const webPort = Number(process.env.E2E_WEB_PORT ?? 3100);
const baseUrl = process.env.E2E_BASE_URL;

export const E2E = {
  apiUrl: (process.env.E2E_API_URL ?? 'http://127.0.0.1:8020').replace(/\/$/, ''),
  webPort,
  webUrl: (baseUrl ?? `http://localhost:${webPort}`).replace(/\/$/, ''),
  startWeb: !baseUrl,
  dbName: process.env.E2E_DB_NAME ?? 'ritme_e2e',
  dbCli:
    process.env.E2E_DB_CLI ??
    'docker compose -f ../backend-go/docker-compose.test.yml exec -T mariadb mariadb -uritme -pritme',
  browserChannel: process.env.E2E_BROWSER_CHANNEL ?? (process.env.CI ? undefined : 'chrome'),
};
