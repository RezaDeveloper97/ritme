import { describe, expect, it } from 'vitest';

import { exportPdfSpec } from './export-pdf';

const labels = { title: 'T', subtitle: 'S', footer: '{page}/{pages}', dir: 'rtl' as const, locale: 'fa' };

describe('exportPdfSpec', () => {
  it('prints every top-level key with its records', () => {
    const spec = exportPdfSpec(
      { user: { id: 1, name: 'A' }, reminders: [{ id: 2, title: 'x' }], empty: [] },
      labels,
    );
    const texts = spec.blocks.map((b) => b.text).filter(Boolean);
    expect(texts.slice(0, 2)).toEqual(['T', 'S']);
    expect(texts).toContain('user');
    expect(texts).toContain('id: 1\nname: A');
    expect(texts).toContain('id: 2\ntitle: x');
    expect(texts).toContain('empty');
    expect(spec.dir).toBe('rtl');
  });

  it('clips very long values', () => {
    const spec = exportPdfSpec({ blob: { data: 'x'.repeat(5000) } }, labels);
    const text = spec.blocks.find((b) => b.text?.startsWith('data:'))?.text ?? '';
    expect(text.length).toBeLessThan(700);
  });
});
