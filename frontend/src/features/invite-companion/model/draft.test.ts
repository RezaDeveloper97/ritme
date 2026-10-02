import { describe, expect, it } from 'vitest';

import { draftToInput, initialDraft, nextStep, normalizeMobile, numberedSteps, phoneProblem, previousStep } from './draft';

describe('invite wizard steps', () => {
  it('skips children for a partner', () => {
    expect(numberedSteps('partner')).toEqual(['type', 'access']);
    expect(numberedSteps('spouse')).toEqual(['type', 'access', 'children']);
    expect(nextStep('access', 'partner')).toBe('invite');
    expect(nextStep('access', 'spouse')).toBe('children');
    expect(previousStep('invite', 'partner')).toBe('access');
    expect(previousStep('invite', 'spouse')).toBe('children');
    expect(previousStep('type', null)).toBeNull();
  });

  it('starts with nothing shared', () => {
    expect(Object.values(initialDraft().grants).every((l) => l === 'none')).toBe(true);
  });
});

describe('normalizeMobile', () => {
  it.each([
    ['09121234567', '09121234567'],
    ['۰۹۱۲ ۱۲۳ ۴۵۶۷', '09121234567'],
    ['+98 912 123 4567', '09121234567'],
    ['00989121234567', '09121234567'],
    ['9121234567', '09121234567'],
  ])('%s → %s', (input, out) => {
    expect(normalizeMobile(input)).toBe(out);
  });

  it('rejects non-mobiles', () => {
    expect(normalizeMobile('0212345678')).toBeNull();
    expect(normalizeMobile('0912abc4567')).toBeNull();
    expect(phoneProblem('')).toBe(false);
    expect(phoneProblem('123')).toBe(true);
  });
});

describe('draftToInput', () => {
  const draft = { ...initialDraft(), type: 'spouse' as const, name: '  Ali ', phone: '۰۹۱۲۱۲۳۴۵۶۷' };

  it('sends the normalized phone and trimmed name', () => {
    expect(draftToInput(draft, true)).toMatchObject({ type: 'spouse', displayName: 'Ali', phone: '09121234567' });
  });

  it('drops the phone for a code-only invite', () => {
    expect(draftToInput(draft, false)).not.toHaveProperty('phone');
  });

  it('needs a type', () => {
    expect(draftToInput(initialDraft(), true)).toBeNull();
  });
});
