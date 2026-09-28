import { describe, expect, it } from 'vitest';

import { articleCategoryLabel, type CategoryTranslator } from './category';

function translator(messages: Record<string, string>): CategoryTranslator {
  const t = ((key: string) => {
    if (!(key in messages)) throw new Error(`missing ${key}`);
    return messages[key];
  }) as CategoryTranslator;
  t.has = (key: string) => key in messages;
  return t;
}

describe('articleCategoryLabel', () => {
  const t = translator({
    'categories.nutrition': 'تغذیه',
    'categories.cycle_science': 'علم چرخه',
  });

  it('translates a known slug', () => {
    expect(articleCategoryLabel('nutrition', t)).toBe('تغذیه');
    expect(articleCategoryLabel('cycle_science', t)).toBe('علم چرخه');
  });

  it('keeps an untranslated slug or free text as stored', () => {
    expect(articleCategoryLabel('sleep_hygiene', t)).toBe('sleep_hygiene');
    expect(articleCategoryLabel('سلامت روان', t)).toBe('سلامت روان');
  });

  it('never looks up a dotted or empty value', () => {
    expect(articleCategoryLabel('categories.nutrition', t)).toBe('categories.nutrition');
    expect(articleCategoryLabel('', t)).toBe('');
  });
});
