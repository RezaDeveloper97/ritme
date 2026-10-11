import { describe, expect, it } from 'vitest';

import { ageParts, sourceHref } from './links';

describe('sourceHref', () => {
  it('deep-links every source to its own review screen', () => {
    expect(sourceHref({ source: 'message', id: 11, ref: 'info' })).toBe('/messages/rules/11');
    expect(sourceHref({ source: 'threshold', id: 2, ref: 'bp.crisis' })).toBe('/clinical-config');
    expect(sourceHref({ source: 'lesson', id: 3, ref: '' })).toBe('/learning/reviews');
    expect(sourceHref({ source: 'assistant', id: 4, ref: 'general' })).toBe('/assistant/departments');
    expect(sourceHref({ source: 'catalog', id: 5, ref: 'lab markers' })).toBe('/catalog/lab%20markers');
  });
});

describe('ageParts', () => {
  it('uses hours below two days and days after', () => {
    expect(ageParts(5)).toEqual({ unit: 'h', value: 5 });
    expect(ageParts(47)).toEqual({ unit: 'h', value: 47 });
    expect(ageParts(100)).toEqual({ unit: 'd', value: 4 });
  });
});
