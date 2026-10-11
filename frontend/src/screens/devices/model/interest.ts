import { onSessionEnd } from '@/shared/session';

/**
 * Watches & gadgets (bloom B-N10-07) — what a real integration needs, and why this screen is a «soon» state:
 *  - The web cannot pair watches/bands: Web Bluetooth is Chromium-only with no background sync, and HealthKit /
 *    Health Connect are native-only (out of scope here: no Android/native work).
 *  - The real route is a vendor cloud (Fitbit, Garmin, Huawei Health, Withings) over OAuth behind a `DeviceProvider`
 *    adapter: token vault, per-metric consent, import into vitals (B-N6), a webhook/poll worker, revoke + delete.
 *    Availability of those clouds in Iran is uncertain.
 *  - Interest: the analytics allow-list (backend `analyticsevents.Spec`) has no event that can carry a device kind,
 *    and it 422s unknown names. Until a `device_interest {kind}` event (or tiny endpoint, Q12) is added, the choice
 *    is kept on this device only and never sent. It holds no health data.
 */
export const DEVICE_KINDS = ['watch', 'band', 'scale', 'thermometer', 'pressure', 'glucose'] as const;
export type DeviceKind = (typeof DEVICE_KINDS)[number];

const KEY = 'ritme_device_interest';

export function readInterest(): DeviceKind[] {
  try {
    const raw = JSON.parse(window.localStorage.getItem(KEY) ?? '[]') as unknown;
    return Array.isArray(raw) ? DEVICE_KINDS.filter((k) => raw.includes(k)) : [];
  } catch {
    return [];
  }
}

export function writeInterest(kinds: readonly DeviceKind[]): void {
  try {
    window.localStorage.setItem(KEY, JSON.stringify(kinds));
  } catch {
    // storage blocked: the toggle just doesn't persist
  }
}

if (typeof window !== 'undefined') {
  onSessionEnd(() => {
    try {
      window.localStorage.removeItem(KEY);
    } catch {
      // ignore
    }
  });
}
