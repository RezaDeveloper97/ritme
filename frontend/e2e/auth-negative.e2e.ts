import { expect, test } from '@playwright/test';

import { expectNoRawKeys, freshMobile, requestOtp, typeOtp, waitOtp } from './support/stack';

test('a wrong OTP shows the error and stays on the OTP step', async ({ page }) => {
  const mobile = freshMobile();
  await requestOtp(page, mobile);
  const real = await waitOtp(mobile);
  // Same length, guaranteed different from the real code.
  const wrong = real.replace(/\d$/, (d) => String((Number(d) + 1) % 10));
  expect(wrong).not.toBe(real);

  await typeOtp(page, wrong);
  const alert = page.getByRole('alert');
  await expect(alert).toBeVisible();
  await expectNoRawKeys(page, '/otp (wrong code)');
  await expect(page).toHaveURL(/\/otp/);
  // The boxes are cleared for a retry, and nothing signed us in.
  await expect(page.locator('.onb2-otp input').first()).toHaveValue('');
  expect(await page.evaluate(() => localStorage.getItem('ritme_token'))).toBeNull();

  // The right code still works afterwards (the step is not dead).
  await typeOtp(page, real);
  await page.waitForURL(/\/onboarding\//);
});

test('an incomplete phone cannot ask for a code', async ({ page }) => {
  await page.goto('/fa/signup');
  await page.locator('input[type=tel]').fill('0912');
  await expect(page.getByRole('button', { name: 'دریافت کد' })).toBeDisabled();
  await expectNoRawKeys(page, '/signup');
});
