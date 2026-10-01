import { describe, expect, it } from 'vitest';

import { searchTarget } from './target';

describe('searchTarget', () => {
  it('opens screens that exist, keeping the query string', () => {
    expect(searchTarget('/analysis/symptoms?category=pain')).toEqual({
      kind: 'route',
      href: '/analysis/symptoms?category=pain',
    });
    expect(searchTarget('/reminders/medication/101802')).toEqual({ kind: 'route', href: '/reminders/medication/101802' });
    expect(searchTarget('/checkups/12')).toEqual({ kind: 'route', href: '/checkups/12' });
    expect(searchTarget('/checkups/self-exam')).toEqual({ kind: 'route', href: '/checkups/self-exam' });
    expect(searchTarget('/contraception')).toEqual({ kind: 'route', href: '/contraception' });
  });

  it('opens an article in the article sheet', () => {
    expect(searchTarget('/articles/period-pain-relief')).toEqual({ kind: 'sheet', id: 'article', arg: 'period-pain-relief' });
  });

  it('hides screens that are not built and anything off-app', () => {
    expect(searchTarget('/programs/pelvic')).toBeNull();
    expect(searchTarget('/reminders/other/1')).toBeNull();
    expect(searchTarget('//evil.example/x')).toBeNull();
    expect(searchTarget('https://evil.example')).toBeNull();
    expect(searchTarget('')).toBeNull();
  });
});
