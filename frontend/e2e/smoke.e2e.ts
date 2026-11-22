import { expect, test, type Page } from '@playwright/test';

import { findRawKeys } from './support/raw-keys';
import { expectNoRawKeys, freshMobile, requestOtp, tehranToday, typeOtp, waitOtp } from './support/stack';

/**
 * The main routine, in a real browser against the local stack, in light and dark (fa):
 * login -> onboarding -> log a period through the log sheet -> home ring shows day 1 -> calendar marks the day.
 * Selectors are the autopilot `pilot()` names (docs/autopilot-catalog.md) wherever one exists.
 */
const pilotSel = (id: string) => `[data-pilot="${id}"]`;
// «روز ۱» — `\b` does not see Persian digits as word characters, so use a lookahead.
const DAY_ONE = /روز\s*[۱1](?![\d۰-۹])/;
const next = (page: Page) => page.getByRole('button', { name: 'ادامه' }).click();

for (const theme of ['light', 'dark'] as const) {
  test.describe(`main routine (${theme})`, () => {
    test.use({ colorScheme: theme });

    test(`login, onboarding, log period, ring, calendar [${theme}]`, async ({ page }) => {
      await page.addInitScript((t) => {
        try {
          localStorage.setItem('ritme_theme', t);
        } catch {
          /* private mode */
        }
      }, theme);
      const today = tehranToday();
      const yesterday = tehranToday(-1);

      await test.step('login with an OTP read from the DB', async () => {
        const mobile = freshMobile();
        await requestOtp(page, mobile);
        await typeOtp(page, await waitOtp(mobile));
        await page.waitForURL(/\/onboarding\/name/);
      });

      await test.step('onboarding', async () => {
        await expectNoRawKeys(page, 'onboarding/name');
        await page.getByLabel('نام', { exact: true }).fill('سارا');
        await next(page);
        await page.waitForURL(/\/onboarding\/gender/);
        await expectNoRawKeys(page, 'onboarding/gender');
        await page.getByRole('radio').first().click(); // زن
        await next(page);
        await page.waitForURL(/\/onboarding\/intention/);
        await expectNoRawKeys(page, 'onboarding/intention');
        await page.getByRole('radio').first().click(); // پیگیری چرخه
        await next(page);
        await page.waitForURL(/\/onboarding\/cycle/);
        await expectNoRawKeys(page, 'onboarding/cycle');
        await page.getByRole('checkbox').click(); // «دقیق یادم نیست»: the period is logged through the sheet below
        await next(page);
        await page.waitForURL(/\/onboarding\/conditions/);
        await expectNoRawKeys(page, 'onboarding/conditions');
        await next(page);
        await page.waitForURL(/\/onboarding\/health/);
        await expectNoRawKeys(page, 'onboarding/health');
        await next(page);
        await page.waitForURL(/\/onboarding\/setting-up/);
        await page.getByRole('button', { name: 'ورود به ریتمی' }).click();
        await page.waitForURL(/\/fa\/home/);
      });

      await test.step('home before logging: no period day', async () => {
        await expect(page.locator(pilotSel('nav.log'))).toBeVisible();
        await expect(page.locator('.home-ring')).toBeVisible();
        await expect(page.locator('html')).toHaveAttribute('data-theme', theme);
        await expectNoRawKeys(page, 'home (before)');
        await expect(page.locator('.home-ring')).not.toContainText(DAY_ONE);
      });

      await test.step('log a period through the log sheet', async () => {
        await page.locator(pilotSel('nav.log')).click();
        await expect(page.locator(pilotSel('log.save'))).toBeVisible();
        // Nothing chosen yet: saving is off (boundary).
        await expect(page.locator(pilotSel('log.save'))).toBeDisabled();
        await page.locator(pilotSel('log.tile:bleeding')).click();
        await page.getByRole('button', { name: 'متوسط', exact: true }).click();
        await expectNoRawKeys(page, 'log sheet');
        await expect(page.locator(pilotSel('log.save'))).toBeEnabled();
        await page.locator(pilotSel('log.save')).click();
        await expect(page.locator(pilotSel('log.save'))).toBeHidden();
      });

      await test.step('home ring shows the new period day', async () => {
        await expect(page.locator('.home-ring')).toContainText('پریود');
        await expect(page.locator('.home-ring')).toContainText(DAY_ONE);
        await expectNoRawKeys(page, 'home (after)');
      });

      await test.step('calendar shows the period on today, not before', async () => {
        await page.goto('/fa/calendar');
        const todayCell = page.locator(`[data-pilot="calendar.day:${today}"]`).first();
        await expect(todayCell).toHaveAttribute('aria-label', /پریود/);
        await expect(page.locator(`[data-pilot="calendar.day:${yesterday}"]`).first()).not.toHaveAttribute(
          'aria-label',
          /پریود/,
        );
        await expect(page.locator('html')).toHaveAttribute('data-theme', theme);
        await expectNoRawKeys(page, 'calendar');
      });

      await test.step('the detector fires on a raw key injected into the live page', async () => {
        await page.evaluate(() => {
          const p = document.createElement('p');
          p.textContent = 'home.ring.noPredictionValue';
          document.body.appendChild(p);
        });
        const text = await page.evaluate(() => document.body.innerText);
        expect(findRawKeys(text)).toEqual(['home.ring.noPredictionValue']);
      });
    });
  });
}
