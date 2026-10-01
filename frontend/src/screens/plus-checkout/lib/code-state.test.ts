import { describe, expect, it } from 'vitest';

import { codeState } from './code-state';

describe('codeState', () => {
  it('is none without a quote priced with the code', () => {
    expect(codeState(undefined)).toBe('none');
  });

  it('is accepted when the code priced the quote', () => {
    expect(codeState({ discountSource: 'code' })).toBe('accepted');
  });

  it('is outranked when the trial offer won over the code', () => {
    expect(codeState({ discountSource: 'trial_offer' })).toBe('outranked');
  });

  it('is outranked when the code gave no discount at all', () => {
    expect(codeState({ discountSource: null })).toBe('outranked');
  });
});
