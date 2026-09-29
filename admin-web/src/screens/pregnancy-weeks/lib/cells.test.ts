import { describe, expect, it } from 'vitest';

import { cellFilled, cellHref, detailsFallback, weekCells } from './cells';

const texts = Array.from({ length: 40 }, (_, i) => ({ week: i + 1, id: i === 4 ? 77 : null }));
const details = Array.from({ length: 42 }, (_, i) => ({ week_number: i + 1, exists: i !== 2 }));

describe('weekCells (QA 2026-09-29-c L7)', () => {
  it('counts weeks with structured details only as filled, and adds weeks 41–42', () => {
    const cells = weekCells(texts, details);
    expect(cells).toHaveLength(42);
    expect(cells.filter(cellFilled)).toHaveLength(41);
    expect(cells[2]).toEqual({ week: 3, textId: null, hasDetails: false });
    expect(cells.at(-1)).toEqual({ week: 42, textId: null, hasDetails: true });
  });

  it('links a text row, else the details tab, else the create page', () => {
    const cells = weekCells(texts, details);
    expect(cellHref(cells[4]!)).toBe('/pregnancy-weeks/77');
    expect(cellHref(cells[9]!)).toBe('/pregnancy-weeks/new?week=10&tab=details');
    expect(cellHref(cells[2]!)).toBe('/pregnancy-weeks/new?week=3');
  });

  it('works without the details list', () => {
    expect(weekCells(texts, undefined).filter(cellFilled)).toHaveLength(1);
  });
});

describe('detailsFallback', () => {
  it('opens the details of week n when /pregnancy-weeks/n has no text row', () => {
    expect(detailsFallback(10, true, details)).toBe('/pregnancy-weeks/new?week=10&tab=details');
    expect(detailsFallback(3, true, details)).toBeNull();
    expect(detailsFallback(10, false, details)).toBeNull();
    expect(detailsFallback(99, true, details)).toBeNull();
    expect(detailsFallback(10, true, undefined)).toBeNull();
  });
});
