import { describe, expect, it } from 'vitest';

import { toApplyInput, validateApply } from './validate';

describe('validateApply', () => {
  it('requires a 2–80 letter display name', () => {
    expect(validateApply({ display_name: ' ', title: '', bio: '' }).display_name).toBe('nameRequired');
    expect(validateApply({ display_name: 'ل', title: '', bio: '' }).display_name).toBe('nameShort');
    expect(validateApply({ display_name: 'ل'.repeat(81), title: '', bio: '' }).display_name).toBe('nameLong');
    expect(validateApply({ display_name: 'لیلا', title: '', bio: '' })).toEqual({});
  });
  it('caps title and bio', () => {
    const e = validateApply({ display_name: 'لیلا', title: 'x'.repeat(61), bio: 'x'.repeat(501) });
    expect(e).toEqual({ title: 'titleLong', bio: 'bioLong' });
  });
  it('trims and nulls empty optionals', () => {
    expect(toApplyInput({ display_name: ' لیلا ', title: ' ', bio: '' })).toEqual({
      display_name: 'لیلا',
      title: null,
      bio: null,
    });
  });
});
