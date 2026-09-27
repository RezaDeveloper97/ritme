import { describe, expect, it } from 'vitest';

import { formatPdfFooter } from './render';

describe('formatPdfFooter', () => {
  it('fills page numbers with Persian digits for fa', () => {
    expect(formatPdfFooter('ریتمی · صفحه {page} از {pages}', 1, 12, 'fa')).toBe('ریتمی · صفحه ۱ از ۱۲');
  });

  it('keeps Latin digits for en', () => {
    expect(formatPdfFooter('Ritme · page {page} of {pages}', 2, 3, 'en')).toBe('Ritme · page 2 of 3');
  });
});
