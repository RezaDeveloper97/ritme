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

export type Gender = 'female' | 'male' | 'none';

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
  gender: Gender | null;
  birth: JalaliBirth;
  weightUnit: WeightUnit;
  weight: number;
  heightUnit: HeightUnit;
  height: number;
  periodLen: number;
  lastPeriodDay: number;
}
