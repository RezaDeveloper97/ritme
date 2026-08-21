import { describe, expect, it } from 'vitest';

import { hrefWithSheet, readSheetTarget } from './url';

const BASE = 'https://ritmeapp.ir/fa/profile';

describe('readSheetTarget', () => {
  it('returns null when no sheet is named', () => {
    expect(readSheetTarget('')).toBeNull();
    expect(readSheetTarget('?date=2026-08-06')).toBeNull();
  });

  it('reads the id and its argument', () => {
    expect(readSheetTarget('?sheet=info&sheetArg=terms')).toEqual({
      id: 'info',
      arg: 'terms',
    });
  });

  it('omits the argument entirely when absent, so targets compare equal', () => {
    expect(readSheetTarget('?sheet=notifications')).toEqual({ id: 'notifications' });
  });
});

describe('hrefWithSheet', () => {
  it('adds the sheet parameters', () => {
    expect(hrefWithSheet(BASE, { id: 'info', arg: 'privacy' })).toBe(
      '/fa/profile?sheet=info&sheetArg=privacy',
    );
  });

  it('keeps the screen’s own query state', () => {
    expect(hrefWithSheet(`${BASE}?date=2026-08-06`, { id: 'phase' })).toBe(
      '/fa/profile?date=2026-08-06&sheet=phase',
    );
  });

  it('drops a stale argument when the next sheet takes none', () => {
    const opened = hrefWithSheet(BASE, { id: 'article', arg: 'cramps' });
    expect(hrefWithSheet(`https://ritmeapp.ir${opened}`, { id: 'articles' })).toBe(
      '/fa/profile?sheet=articles',
    );
  });

  it('removes both parameters when closing, leaving the rest alone', () => {
    const opened = hrefWithSheet(`${BASE}?date=2026-08-06`, { id: 'info', arg: 'help' });
    expect(hrefWithSheet(`https://ritmeapp.ir${opened}`, null)).toBe(
      '/fa/profile?date=2026-08-06',
    );
  });
});
