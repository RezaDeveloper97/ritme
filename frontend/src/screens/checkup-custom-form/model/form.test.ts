import { describe, expect, it } from 'vitest';

import { emptyCustomForm, intervalKey, validateCustomForm } from './form';

describe('custom checkup form', () => {
  it('maps interval chips', () => {
    expect(intervalKey(6)).toBe('m6');
    expect(intervalKey(5)).toBeNull();
  });
  it('requires a title', () => {
    expect(validateCustomForm(emptyCustomForm())).toBe('title');
    expect(validateCustomForm({ ...emptyCustomForm(), title: '  eye ' })).toBeNull();
  });
});
