import { ApiError } from '@/shared/api';

/**
 * ONE normaliser for every AI gate refusal (docs/ai-credits.md, «Gate errors»). It accepts an {@link ApiError}
 * from `apiClient` / the SSE stream's non-2xx, the `error_code` string of an SSE `error` event, or that event's
 * whole data object. Pure; reads only the envelope's top-level gate fields — never a health payload.
 */
export type AiDenial =
  | { kind: 'subscription'; code: string; extraCredits: number; feature: string | null; message: string | null }
  | {
      kind: 'credits';
      code: string;
      required: number;
      balance: number;
      planRemaining: number;
      extraCredits: number;
      periodEndsAt: string | null;
      feature: string | null;
      message: string | null;
    }
  | { kind: 'disabled'; code: string; feature: string | null; message: string | null }
  | { kind: 'consent'; code: string; consent: string | null; version: number | null; message: string | null }
  | { kind: 'busy'; code: string; retryAfter: number | null; message: string | null }
  | { kind: 'unavailable'; code: string; message: string | null };

export type AiDenialKind = AiDenial['kind'];

/** The kinds that open the gate sheet (the user can fix them with a subscription / credits). */
export type AiGateDenial = Extract<AiDenial, { kind: 'subscription' | 'credits' | 'disabled' }>;

const BUSY = ['ai_busy', 'ai_user_budget_exhausted', 'too_many_requests'];
const UNAVAILABLE = ['ai_unavailable', 'ai_failed', 'ai_budget_exhausted'];

function str(v: unknown): string | null {
  return typeof v === 'string' && v.trim() ? v : null;
}
function num(v: unknown): number | null {
  return typeof v === 'number' && Number.isFinite(v) ? v : null;
}

function bodyOf(input: unknown): Record<string, unknown> | null {
  if (typeof input === 'string') return { error_code: input };
  if (input instanceof ApiError) {
    const data = input.response?.data;
    return typeof data === 'object' && data !== null ? (data as Record<string, unknown>) : null;
  }
  if (typeof input === 'object' && input !== null && 'error_code' in input) return input as Record<string, unknown>;
  return null;
}

export function aiDenialOf(input: unknown): AiDenial | null {
  const b = bodyOf(input);
  if (!b) return null;
  const code = str(b.error_code);
  if (!code) return null;
  const message = str(b.message);
  const feature = str(b.feature);
  if (code === 'ai_subscription_required')
    return { kind: 'subscription', code, extraCredits: num(b.extra_credits) ?? 0, feature, message };
  if (code === 'ai_credits_exhausted')
    return {
      kind: 'credits',
      code,
      required: num(b.required) ?? 0,
      balance: num(b.balance) ?? 0,
      planRemaining: num(b.plan_remaining) ?? 0,
      extraCredits: num(b.extra_credits) ?? 0,
      periodEndsAt: str(b.period_ends_at),
      feature,
      message,
    };
  if (code === 'ai_feature_disabled') return { kind: 'disabled', code, feature, message };
  if (code === 'consent_required') return { kind: 'consent', code, consent: str(b.consent), version: num(b.version), message };
  if (BUSY.includes(code)) return { kind: 'busy', code, retryAfter: num(b.retry_after), message };
  if (UNAVAILABLE.includes(code)) return { kind: 'unavailable', code, message };
  return null;
}

/** Whether the denial is one the gate sheet handles. */
export function isGateDenial(d: AiDenial | null): d is AiGateDenial {
  return d !== null && (d.kind === 'subscription' || d.kind === 'credits' || d.kind === 'disabled');
}
