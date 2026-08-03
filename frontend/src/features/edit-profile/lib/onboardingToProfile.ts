import type { OnboardingData } from '@/entities/user';
import { toApiDate, today } from '@/shared/lib/date';

import type { UpdateProfileInput } from '../api/mutations';
import { datePartsToApiDate } from './datePartsToApiDate';

const clamp = (value: number, lo: number, hi: number): number =>
  Math.min(hi, Math.max(lo, value));

/**
 * Map the answers collected across the onboarding steps onto the POST /profile
 * payload, so registration actually persists to the user's profile.
 *
 * Values are converted to the API's canonical units (kg, cm) and clamped to its
 * validation ranges (weight 20–300, height 50–250, period 1–15, cycle 15–60) so
 * a stray wheel value can't trigger a 422. Date parts are interpreted in
 * `o.locale`'s calendar (Jalali for fa, Gregorian for en) and cross the boundary
 * as Gregorian (CLAUDE.md §7); any date the API would reject — a birthday not
 * before today,
 * or a future last-period date — is dropped rather than sent. This is sensitive
 * health data (§11): it is submitted, never logged.
 *
 * The flow branches on {@link OnboardingData.intention}: pregnant users skip the
 * cycle questions entirely, so their cycle fields are omitted (period tracking
 * is disabled in pregnancy mode).
 */
export function onboardingToProfileInput(o: OnboardingData): UpdateProfileInput {
  const todayStr = toApiDate(today());
  const payload: UpdateProfileInput = {};

  const name = o.name.trim();
  if (name) payload.name = name;

  const birthday = datePartsToApiDate(
    { year: o.birth.y, month: o.birth.m, day: o.birth.d },
    o.locale,
  );
  if (birthday < todayStr) payload.birthday = birthday;

  const weightKg = o.weightUnit === 'lb' ? o.weight / 2.205 : o.weight;
  payload.weight = clamp(Math.round(weightKg), 20, 300);

  const heightCm = o.heightUnit === 'ft' ? o.height * 30.48 : o.height;
  payload.height = clamp(Math.round(heightCm), 50, 250);

  if (o.intention) payload.pregnancy_intention = o.intention;
  payload.chronic_conditions = o.chronicConditions;

  // Cycle questions are only asked (and only meaningful) off the pregnant branch.
  if (o.intention !== 'pregnant') {
    payload.period_duration = clamp(Math.round(o.periodLen), 1, 15);
    payload.cycle_duration = clamp(Math.round(o.cycleDuration), 15, 60);

    if (o.lastPeriod) {
      const lastPeriodStart = datePartsToApiDate(o.lastPeriod, o.locale);
      if (lastPeriodStart <= todayStr) payload.last_period_start = lastPeriodStart;
    }
  }

  return payload;
}
