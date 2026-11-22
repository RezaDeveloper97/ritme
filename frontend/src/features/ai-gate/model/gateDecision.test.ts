import { describe, expect, it } from 'vitest';

import { aiDenialOf, gateStateOf, isGateDenial, type AiCredits } from '@/entities/ai-credit';
import { ApiError } from '@/shared/api';

import { denialFromBalance, guardDecision } from './gateDecision';

const KEY = 'chat';

/** A balance read; the cost always comes from `features[].creditCost`, never a literal in the code under test. */
function credits(over: Partial<AiCredits> & { cost?: number; available?: boolean; affordable?: boolean } = {}): AiCredits {
  const { cost = 7, available = true, affordable = true, ...rest } = over;
  return {
    canUse: true,
    period: 'month' as AiCredits['period'],
    periodEndsAt: '2026-12-01T00:00:00Z',
    plan: { total: 100, used: 20, remaining: 80 },
    extra: { remaining: 5, locked: false },
    spendable: 85,
    features: [{ key: KEY, title: 'Chat', creditCost: cost, available, affordable } as AiCredits['features'][number]],
    ...rest,
  };
}

function apiError(status: number, body: Record<string, unknown>): ApiError {
  return new ApiError('x', { response: { data: body, status, headers: {} }, sentToken: 't' });
}

function decide(c: AiCredits | undefined, loading = false) {
  return guardDecision(gateStateOf(c, KEY, loading), c, KEY);
}

describe('gate guard (balance → sheet)', () => {
  it('lets a user with enough credits through, no sheet', () => {
    expect(decide(credits())).toEqual({ allowed: true, denial: null });
  });

  it('no subscription → subscription sheet carrying the extra credits kept', () => {
    const r = decide(credits({ canUse: false, spendable: 0, extra: { remaining: 9, locked: true } }));
    expect(r.allowed).toBe(false);
    expect(r.denial).toMatchObject({ kind: 'subscription', code: 'ai_subscription_required', extraCredits: 9, feature: KEY });
  });

  it('out of credits → credits sheet with required (the API cost) and balance', () => {
    const r = decide(credits({ cost: 12, spendable: 11, affordable: false }));
    expect(r.allowed).toBe(false);
    expect(r.denial).toMatchObject({ kind: 'credits', required: 12, balance: 11, planRemaining: 80, extraCredits: 5 });
  });

  it('feature off → disabled sheet', () => {
    const r = decide(credits({ available: false }));
    expect(r.allowed).toBe(false);
    expect(r.denial).toMatchObject({ kind: 'disabled', code: 'ai_feature_disabled', feature: KEY });
  });

  it('boundary: balance exactly equal to cost is allowed, one below is not', () => {
    expect(decide(credits({ cost: 10, spendable: 10 })).allowed).toBe(true);
    expect(decide(credits({ cost: 10, spendable: 9, affordable: true })).allowed).toBe(false);
  });

  it('fails open while the balance is unknown (loading, failed read)', () => {
    expect(decide(undefined, true)).toEqual({ allowed: true, denial: null });
    expect(decide(undefined, false)).toEqual({ allowed: true, denial: null });
  });

  it('a feature missing from the catalogue does not block', () => {
    const c = credits();
    c.features = [];
    expect(decide(c).allowed).toBe(true);
  });

  it('cost changes between render and press: the decision uses the state at press time', () => {
    const rendered = credits({ cost: 5, spendable: 8 });
    expect(decide(rendered).allowed).toBe(true);
    // admin raised the cost; the refetched balance no longer covers it
    const pressed = credits({ cost: 9, spendable: 8, affordable: false });
    const r = decide(pressed);
    expect(r.allowed).toBe(false);
    expect(r.denial).toMatchObject({ kind: 'credits', required: 9, balance: 8 });
  });

  it('the cost is never a literal: any API cost round-trips into the sheet', () => {
    for (const cost of [1, 3, 37, 1000]) {
      const r = decide(credits({ cost, spendable: cost - 1, affordable: false }));
      expect(r.denial).toMatchObject({ kind: 'credits', required: cost });
    }
  });

  it('denialFromBalance is null for ready/loading, so a ready state never opens a sheet', () => {
    const c = credits();
    expect(denialFromBalance(c, KEY, 'ready')).toBeNull();
    expect(denialFromBalance(c, KEY, 'loading')).toBeNull();
  });
});

describe('gate error handling (server answer → sheet or own state)', () => {
  const cases: Array<[number, Record<string, unknown>, string, boolean]> = [
    [402, { error_code: 'ai_subscription_required', feature: KEY, extra_credits: 3 }, 'subscription', true],
    [402, { error_code: 'ai_credits_exhausted', feature: KEY, required: 4, balance: 1, plan_remaining: 0, extra_credits: 1, period_ends_at: '2026-12-01' }, 'credits', true],
    [403, { error_code: 'ai_feature_disabled', feature: KEY }, 'disabled', true],
    [403, { error_code: 'consent_required', consent: 'ai', version: 2, reason: 'x' }, 'consent', false],
    [429, { error_code: 'ai_busy', retry_after: 5 }, 'busy', false],
    [503, { error_code: 'ai_unavailable' }, 'unavailable', false],
  ];

  it.each(cases)('%i %j → %s (sheet: %s)', (status, body, kind, sheet) => {
    const d = aiDenialOf(apiError(status, body));
    expect(d?.kind).toBe(kind);
    expect(isGateDenial(d)).toBe(sheet);
  });

  it('credits error carries the server required/balance, not a local number', () => {
    const d = aiDenialOf(apiError(402, { error_code: 'ai_credits_exhausted', required: 14, balance: 2 }));
    expect(d).toMatchObject({ kind: 'credits', required: 14, balance: 2 });
  });

  it('an unrelated error (validation, session 401, plain Error) is not a gate denial', () => {
    expect(aiDenialOf(apiError(422, { error_code: 'ai_invalid_request' }))).toBeNull();
    expect(aiDenialOf(apiError(401, { error_code: 'token_expired' }))).toBeNull();
    expect(aiDenialOf(new Error('boom'))).toBeNull();
    expect(isGateDenial(null)).toBe(false);
  });
});
