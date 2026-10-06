import { describe, expect, it } from 'vitest';

import { formatPdfFooter, rowColumns } from './render';

describe('formatPdfFooter', () => {
  it('fills page numbers with Persian digits for fa', () => {
    expect(formatPdfFooter('ریتمی · صفحه {page} از {pages}', 1, 12, 'fa')).toBe('ریتمی · صفحه ۱ از ۱۲');
  });

  it('keeps Latin digits for en', () => {
    expect(formatPdfFooter('Ritme · page {page} of {pages}', 2, 3, 'en')).toBe('Ritme · page 2 of 3');
  });
});

describe('rowColumns', () => {
  it('splits the width by weight, first column first', () => {
    expect(rowColumns(100, 2)).toEqual([
      { start: 0, width: 50 },
      { start: 50, width: 50 },
    ]);
    const cols = rowColumns(120, 3, [2, 3, 1]);
    expect(cols.map((c) => c.width)).toEqual([40, 60, 20]);
    expect(cols[2]?.start).toBe(100);
  });

  it('treats missing or non-positive weights as 1', () => {
    expect(rowColumns(90, 3, [0]).map((c) => c.width)).toEqual([30, 30, 30]);
  });
});
