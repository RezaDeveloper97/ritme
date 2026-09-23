import { describe, expect, it } from 'vitest';

import { excerpt, pickTranslation } from './translations';

describe('pickTranslation', () => {
  it('prefers the given codes in order, then any non-empty value', () => {
    expect(pickTranslation({ fa: 'سلام', en: 'Hi' }, ['en', 'fa'])).toBe('Hi');
    expect(pickTranslation({ fa: 'سلام', en: ' ' }, ['en', 'fa'])).toBe('سلام');
    expect(pickTranslation({ ar: 'مرحبا' }, ['en', 'fa'])).toBe('مرحبا');
    expect(pickTranslation(null, ['fa'])).toBe('');
    expect(pickTranslation({}, [undefined])).toBe('');
  });
});

describe('excerpt', () => {
  it('strips tags and cuts long text', () => {
    expect(excerpt('<p>a&nbsp;<b>b</b></p>')).toBe('a b');
    expect(excerpt('x'.repeat(100), 10)).toBe(`${'x'.repeat(10)}…`);
  });
});
