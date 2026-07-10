import type { JalaliParts } from '@/shared/lib/date';

/**
 * The authenticated account as returned by the Auth endpoints
 * (`/auth/verify-otp`, `/auth/user`). Mirrors the OpenAPI `user` object — this
 * is identity/account data, distinct from the health-profile onboarding data
 * below. Never log it (CLAUDE.md §11).
 */
export interface AuthUser {
  id: number;
  name: string | null;
  mobile: string;
  mobileVerifiedAt: string | null;
}

/**
 * The user's health/onboarding profile as returned by `GET /profile`
 * (OpenAPI `profile` object). This is sensitive cycle data (CLAUDE.md §11) —
 * never log it. Every field is nullable: a freshly-registered account has no
 * profile yet, and the backend may omit individual values. Dates arrive in the
 * API's format (Gregorian ISO) and are converted to Jalali only for display
 * (§7).
 */
export interface HealthProfile {
  /** Date of birth, ISO `YYYY-MM-DD`. */
  birthday: string | null;
  /** Weight in kilograms. */
  weight: number | null;
  /** Height in centimetres. */
  height: number | null;
  /** Typical period length, in days. */
  periodDuration: number | null;
  /** Typical cycle length, in days. */
  cycleDuration: number | null;
  /** Start date of the last period, ISO `YYYY-MM-DD`. */
  lastPeriodStart: string | null;
  /** Pregnancy intention stated at onboarding, or `null` if not set. */
  pregnancyIntention: PregnancyIntention | null;
  /** Self-reported chronic conditions (empty when none). */
  chronicConditions: ChronicCondition[];
}

/**
 * The full `GET /profile` payload: account identity flattened together with the
 * optional {@link HealthProfile}. Distinct from {@link AuthUser} (which carries
 * verification/timestamps from the auth endpoints).
 */
export interface UserProfile {
  id: number;
  name: string | null;
  mobile: string;
  health: HealthProfile | null;
}

/**
 * How the user relates to pregnancy, asked once during onboarding. Drives the
 * branch: `pregnant` → pregnancy mode (period tracking off); everything else →
 * the cycle questions. Mirrors the backend `PregnancyIntention` enum.
 */
export type PregnancyIntention = 'avoiding' | 'pregnant' | 'trying' | 'unsure';

/** Optional self-reported chronic conditions. Mirrors the backend enum. */
export type ChronicCondition =
  | 'pcos'
  | 'hypothyroidism'
  | 'hyperthyroidism'
  | 'hypertension'
  | 'heart_disease'
  | 'diabetes';

/**
 * Basis for dating the pregnancy, collected only when {@link PregnancyIntention}
 * is `pregnant`. Kept locally in this slice (not imported from `entities/pregnancy`)
 * because sibling entities must not cross-import (CLAUDE.md §3.3); the onboarding
 * `setting-up` screen composes it into the pregnancy `OnboardingInput`.
 */
export type OnboardingAgeSource = 'lmp' | 'ultrasound' | 'manual';

export interface PregnancyBasis {
  source: OnboardingAgeSource | null;
  /** First day of the last menstrual period. */
  lmp: JalaliParts | null;
  ultrasoundDate: JalaliParts | null;
  ultrasoundWeeks: number | null;
  ultrasoundDays: number | null;
  manualWeeks: number | null;
  manualDays: number | null;
}

export interface JalaliBirth {
  d: number;
  m: number;
  y: number;
}

export type WeightUnit = 'kg' | 'lb';
export type HeightUnit = 'cm' | 'ft';

export interface OnboardingData {
  phone: string;
  name: string;
  birth: JalaliBirth;
  weightUnit: WeightUnit;
  weight: number;
  heightUnit: HeightUnit;
  height: number;
  /** Pregnancy intention, or `null` until the user answers. */
  intention: PregnancyIntention | null;
  /** Dating basis, populated only on the `pregnant` branch. */
  pregnancyBasis: PregnancyBasis;
  /** Optional chronic conditions (empty when none selected). */
  chronicConditions: ChronicCondition[];
  periodLen: number;
  /** Typical cycle length in days (API `cycle_duration`, validated 15–60). */
  cycleDuration: number;
  /** Start date of the last period, or `null` until the user picks it. */
  lastPeriod: JalaliParts | null;
}
