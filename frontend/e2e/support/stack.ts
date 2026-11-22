import { execSync } from 'node:child_process';

import { expect, type Page } from '@playwright/test';

import { E2E } from './env';
import { findRawKeys } from './raw-keys';

/** The newest OTP for `mobile`, read from the private DB (no OTP bypass exists; we own the database). */
export function readOtp(mobile: string): string {
  if (!/^\d{11}$/.test(mobile)) throw new Error('mobile must be 11 digits');
  const sql = `select code from otp_verifications where mobile='${mobile}' order by id desc limit 1`;
  const out = execSync(`${E2E.dbCli} ${E2E.dbName} -N -e "${sql}"`, {
    encoding: 'utf8',
    // stdin closed: `docker compose exec -T` otherwise waits for EOF on an open pipe and hangs the worker.
    stdio: ['ignore', 'pipe', 'pipe'],
    timeout: 20_000,
  }).trim();
  if (!out) throw new Error('no OTP row yet');
  return out;
}

/** A never-seen mobile per journey (a new number registers on verify). */
export function freshMobile(): string {
  return `0912${String(Math.floor(Math.random() * 9_000_000) + 1_000_000)}`;
}

/** Today in Tehran as `YYYY-MM-DD` (the browser context runs in Asia/Tehran). */
export function tehranToday(offsetDays = 0): string {
  const d = new Date(Date.now() + offsetDays * 86_400_000);
  return new Intl.DateTimeFormat('en-CA', { timeZone: 'Asia/Tehran' }).format(d);
}

/** Fails the test when the visible text of the current page contains a raw `namespace.key`. */
export async function expectNoRawKeys(page: Page, where: string): Promise<void> {
  const text = await page.evaluate(() => document.body.innerText);
  expect(findRawKeys(text), `raw i18n keys on ${where}`).toEqual([]);
}

/** signup screen: phone + both consents, then «دریافت کد». Lands on /otp. */
export async function requestOtp(page: Page, mobile: string): Promise<void> {
  await page.goto('/fa/signup');
  // The first hit on a cold dev server can act before hydration and lose the input: retry until the form took it.
  await expect(async () => {
    await page.locator('input[type=tel]').fill(mobile);
    for (const box of await page.getByRole('checkbox').all()) await box.check({ timeout: 3_000 });
    await expect(page.getByRole('button', { name: 'دریافت کد' })).toBeEnabled({ timeout: 2_000 });
  }).toPass({ timeout: 40_000 });
  await page.getByRole('button', { name: 'دریافت کد' }).click();
  await page.waitForURL(/\/otp/);
  await expectNoRawKeys(page, '/otp');
}

export async function typeOtp(page: Page, code: string): Promise<void> {
  const boxes = page.locator('.onb2-otp input');
  await expect(boxes).toHaveCount(code.length);
  for (let i = 0; i < code.length; i += 1) await boxes.nth(i).fill(code[i]!);
}

/** The OTP row is written by the API a moment after the click. */
export async function waitOtp(mobile: string): Promise<string> {
  let last: unknown;
  for (let i = 0; i < 20; i += 1) {
    try {
      return readOtp(mobile);
    } catch (e) {
      last = e;
      await new Promise((r) => setTimeout(r, 500));
    }
  }
  throw last;
}
