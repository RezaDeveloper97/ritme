import { describe, expect, it } from 'vitest';

import { joinList } from './list';

describe('joinList', () => {
  it('joins Persian lists without a comma before «و»', () => {
    expect(joinList(['علائم و حال روزانه', 'داروها', 'بارداری'], 'fa')).toBe('علائم و حال روزانه، داروها و بارداری');
    expect(joinList(['داروها', 'بارداری'], 'fa')).toBe('داروها و بارداری');
    expect(joinList(['بارداری'], 'fa')).toBe('بارداری');
    expect(joinList([], 'fa')).toBe('');
    expect(joinList(['a', 'b', 'c'], 'fa')).not.toContain('،‏');
  });

  it('uses Intl.ListFormat elsewhere', () => {
    expect(joinList(['symptoms', 'medications', 'pregnancy'], 'en')).toBe('symptoms, medications, and pregnancy');
    expect(joinList(['meds'], 'en')).toBe('meds');
  });
});
