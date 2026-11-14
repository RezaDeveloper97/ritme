import { describe, expect, it } from 'vitest';

import { dailyIndex, learnTopic, matchArticle } from './learn';

const article = (slug: string) => ({ id: 1, slug, title: null, readTimeMinutes: null, imageUrl: null });

describe('matchArticle', () => {
  it('finds an article by a slug keyword', () => {
    const a = article('luteal-phase-and-you');
    expect(matchArticle(learnTopic('cycle'), [article('sleep-tips'), a])).toBe(a);
  });

  it('is null when nothing matches', () => {
    expect(matchArticle(learnTopic('empathy'), [article('iron-rich-food')])).toBeNull();
  });
});

describe('dailyIndex', () => {
  it('turns over day by day and wraps', () => {
    const a = dailyIndex('2026-11-08', 5);
    expect(dailyIndex('2026-11-09', 5)).toBe((a + 1) % 5);
    expect(dailyIndex('2026-11-13', 5)).toBe(a);
  });

  it('is 0 for an empty list or a bad day', () => {
    expect(dailyIndex('2026-11-08', 0)).toBe(0);
    expect(dailyIndex('nope', 4)).toBe(0);
  });
});
