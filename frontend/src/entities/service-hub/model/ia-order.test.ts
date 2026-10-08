import { describe, expect, it } from 'vitest';

import { iaOrderedSections } from './ia-order';
import type { ServiceSection } from './types';

const sec = (code: string) => ({ code }) as unknown as ServiceSection;
const codes = (list: string[]) => iaOrderedSections(list.map(sec)).map((s) => s.code);

describe('iaOrderedSections (CB-NAV-03)', () => {
  it('keeps the board order as is', () => {
    const board = ['search', 'booking', 'care', 'checkups', 'programs', 'mother_child', 'learning', 'shop', 'emergency'];
    expect(codes(board)).toEqual(board);
  });

  it('moves a shop the admin put first to the end, before the emergency card', () => {
    expect(codes(['shop', 'emergency', 'care', 'learning'])).toEqual(['care', 'learning', 'shop', 'emergency']);
  });

  it('works without a shop (teen) and without an emergency section', () => {
    expect(codes(['care', 'emergency', 'programs'])).toEqual(['care', 'programs', 'emergency']);
    expect(codes(['shop', 'care'])).toEqual(['care', 'shop']);
  });
});
