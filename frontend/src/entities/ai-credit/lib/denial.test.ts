import { describe, expect, it } from 'vitest';

import { ApiError } from '@/shared/api';

import { aiDenialOf, isGateDenial } from './denial';

const failed = (status: number, data: unknown) => new ApiError('x', { response: { status, data, headers: {} }, sentToken: null });

describe('aiDenialOf', () => {
  it('maps ai_subscription_required with the kept extra credits', () => {
    expect(aiDenialOf(failed(402, { error_code: 'ai_subscription_required', feature: 'assistant', extra_credits: 50 }))).toMatchObject({
      kind: 'subscription',
      extraCredits: 50,
      feature: 'assistant',
    });
    expect(aiDenialOf(failed(402, { error_code: 'ai_subscription_required' }))).toMatchObject({ kind: 'subscription', extraCredits: 0 });
  });

  it('maps ai_credits_exhausted with every amount', () => {
    expect(
      aiDenialOf(
        failed(402, {
          error_code: 'ai_credits_exhausted',
          feature: 'lab_analysis',
          required: 5,
          balance: 2,
          plan_remaining: 2,
          extra_credits: 0,
          period_ends_at: '2026-12-12T10:00:00+03:30',
        }),
      ),
    ).toEqual({
      kind: 'credits',
      code: 'ai_credits_exhausted',
      required: 5,
      balance: 2,
      planRemaining: 2,
      extraCredits: 0,
      periodEndsAt: '2026-12-12T10:00:00+03:30',
      feature: 'lab_analysis',
      message: null,
    });
  });

  it('maps a disabled feature', () => {
    expect(aiDenialOf(failed(403, { error_code: 'ai_feature_disabled', feature: 'voice_log' }))).toMatchObject({ kind: 'disabled', feature: 'voice_log' });
  });

  it('maps consent with its code and version', () => {
    expect(aiDenialOf(failed(403, { error_code: 'consent_required', consent: 'ai_assistant', version: 2 }))).toMatchObject({
      kind: 'consent',
      consent: 'ai_assistant',
      version: 2,
    });
  });

  it.each(['ai_busy', 'ai_user_budget_exhausted', 'too_many_requests'])('maps %s to busy with retry_after', (code) => {
    expect(aiDenialOf(failed(429, { error_code: code, retry_after: 30, message: 'later' }))).toMatchObject({ kind: 'busy', retryAfter: 30, message: 'later' });
    expect(aiDenialOf(failed(429, { error_code: code }))).toMatchObject({ kind: 'busy', retryAfter: null });
  });

  it.each(['ai_unavailable', 'ai_failed', 'ai_budget_exhausted'])('maps %s to unavailable', (code) => {
    expect(aiDenialOf(failed(503, { error_code: code }))).toMatchObject({ kind: 'unavailable' });
  });

  it('accepts the error_code string of an SSE error event and its whole data object', () => {
    expect(aiDenialOf('ai_credits_exhausted')).toMatchObject({ kind: 'credits', required: 0 });
    expect(aiDenialOf('ai_failed')).toMatchObject({ kind: 'unavailable' });
    expect(aiDenialOf({ error_code: 'ai_subscription_required', extra_credits: 10 })).toMatchObject({ kind: 'subscription', extraCredits: 10 });
  });

  it('returns null for anything else', () => {
    expect(aiDenialOf(null)).toBeNull();
    expect(aiDenialOf(undefined)).toBeNull();
    expect(aiDenialOf(new Error('x'))).toBeNull();
    expect(aiDenialOf('something_else')).toBeNull();
    expect(aiDenialOf('')).toBeNull();
    expect(aiDenialOf({ message: 'no code' })).toBeNull();
    expect(aiDenialOf(failed(402, { error_code: 'plus_required' }))).toBeNull(); // the non-AI Plus gate is not ours
    expect(aiDenialOf(failed(500, 'html'))).toBeNull();
    expect(aiDenialOf(new ApiError('t', { code: 'timeout', sentToken: null }))).toBeNull();
  });

  it('flags only subscription / credits / disabled as gate denials', () => {
    expect(isGateDenial(aiDenialOf('ai_subscription_required'))).toBe(true);
    expect(isGateDenial(aiDenialOf('ai_credits_exhausted'))).toBe(true);
    expect(isGateDenial(aiDenialOf('ai_feature_disabled'))).toBe(true);
    expect(isGateDenial(aiDenialOf('consent_required'))).toBe(false);
    expect(isGateDenial(aiDenialOf('ai_busy'))).toBe(false);
    expect(isGateDenial(aiDenialOf('ai_failed'))).toBe(false);
    expect(isGateDenial(null)).toBe(false);
  });
});
