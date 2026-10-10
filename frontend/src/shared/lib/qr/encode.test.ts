import { describe, expect, it } from 'vitest';

import { encodeQr, qrSvgPath } from './encode';
import fixtures from './fixtures.json';

/*
 * Fixtures: module matrices from the Python `qrcode` package (level M, byte mode, each of the 8 masks) for short
 * texts and share-URL-sized ones (versions 1–10), so the encoder is checked bit for bit against a reference.
 */
describe('encodeQr', () => {
  for (const [key, [version, bits]] of Object.entries(fixtures as unknown as Record<string, [number, string]>)) {
    const sep = key.lastIndexOf('|');
    const text = key.slice(0, sep);
    const mask = Number(key.slice(sep + 1));
    it(`matches the reference for ${text.length} bytes, mask ${mask}`, () => {
      const qr = encodeQr(text, mask);
      expect(qr.version).toBe(version);
      expect(qr.modules.map((r) => r.map((c) => (c ? '1' : '0')).join('')).join('')).toBe(bits);
    });
  }

  it('picks a mask on its own and draws one path', () => {
    const qr = encodeQr('https://example.com/fa/d');
    expect(qr.mask).toBeGreaterThanOrEqual(0);
    expect(qr.mask).toBeLessThan(8);
    expect(qrSvgPath(qr)).toMatch(/^M\d+ \d+h1v1h-1z/);
  });

  it('refuses text that does not fit version 10', () => {
    expect(() => encodeQr('z'.repeat(300))).toThrow();
  });
});
