import { describe, expect, it } from 'vitest';

import { blankToNull, isoToLocalInput, localInputToApi, toFormData, toIntOrNull } from './form-data';

const entries = (fd: FormData) => [...fd.entries()].map(([k, v]) => [k, typeof v === 'string' ? v : 'FILE']);

describe('toFormData', () => {
  it('encodes translations, lists, booleans, nulls and files in bracket syntax', () => {
    const file = new Blob(['x'], { type: 'image/png' });
    const fd = toFormData({
      title: { fa: 'سلام', en: '' },
      cycle_phases: ['menstrual', 'follicular_early'],
      is_active: true,
      is_published: false,
      category: null,
      sort_order: 10,
      remove_image: undefined,
      image: file,
    });
    expect(entries(fd)).toEqual([
      ['title[fa]', 'سلام'],
      ['title[en]', ''],
      ['cycle_phases[]', 'menstrual'],
      ['cycle_phases[]', 'follicular_early'],
      ['is_active', '1'],
      ['is_published', '0'],
      ['category', ''],
      ['sort_order', '10'],
      ['image', 'FILE'],
    ]);
  });
});

describe('form value helpers', () => {
  it('blankToNull / toIntOrNull', () => {
    expect(blankToNull('  ')).toBeNull();
    expect(blankToNull(' a ')).toBe('a');
    expect(toIntOrNull('')).toBeNull();
    expect(toIntOrNull('12')).toBe(12);
    expect(toIntOrNull('abc')).toBeNull();
  });

  it('datetime-local round trip keeps Tehran wall-clock time', () => {
    expect(isoToLocalInput('2026-09-23T13:05:00+03:30')).toBe('2026-09-23T13:05');
    expect(isoToLocalInput(null)).toBe('');
    expect(localInputToApi('2026-09-23T13:05')).toBe('2026-09-23 13:05');
    expect(localInputToApi('')).toBeNull();
  });
});
