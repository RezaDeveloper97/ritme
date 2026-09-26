import { describe, expect, it } from 'vitest';

import { buildImagePdf } from './writer';

const decode = (b: Uint8Array) => new TextDecoder('latin1').decode(b);

describe('buildImagePdf', () => {
  it('writes a valid xref pointing at every object', () => {
    const jpeg = new Uint8Array([0xff, 0xd8, 0xff, 0xd9]);
    const pdf = buildImagePdf([
      { jpeg, pixelWidth: 10, pixelHeight: 20 },
      { jpeg, pixelWidth: 10, pixelHeight: 20 },
    ]);
    const text = decode(pdf);
    expect(text.startsWith('%PDF-1.4')).toBe(true);
    expect(text).toContain('/Count 2');
    const startxref = Number(/startxref\n(\d+)/.exec(text)?.[1]);
    expect(text.slice(startxref, startxref + 4)).toBe('xref');
    const entries = [...text.slice(startxref).matchAll(/(\d{10}) 00000 n/g)].map((m) => Number(m[1]));
    expect(entries).toHaveLength(8);
    entries.forEach((offset, i) => expect(text.slice(offset).startsWith(`${i + 1} 0 obj`)).toBe(true));
  });

  it('refuses an empty document', () => {
    expect(() => buildImagePdf([])).toThrow();
  });
});
