import { describe, expect, it } from 'vitest';

import { codeEntryUrl, qrTarget, toggleDocument, toggleSection } from './share-code';

const PATH = '/fa/shared/report/AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA';

describe('qrTarget', () => {
  it('uses the server URL when it is this origin and ends in the same path', () => {
    expect(qrTarget({ qrUrl: `https://web.ritme.app${PATH}`, qrPath: PATH }, 'https://web.ritme.app')).toBe(
      `https://web.ritme.app${PATH}`,
    );
  });

  it('falls back to this origin for a missing, foreign or odd URL', () => {
    expect(qrTarget({ qrUrl: null, qrPath: PATH }, 'http://localhost:3000/')).toBe(`http://localhost:3000${PATH}`);
    expect(qrTarget({ qrUrl: `https://evil.example${PATH}`, qrPath: PATH }, 'https://a.b')).toBe(`https://a.b${PATH}`);
    expect(qrTarget({ qrUrl: `http://a.b${PATH}`, qrPath: PATH }, 'https://a.b')).toBe(`https://a.b${PATH}`);
    expect(qrTarget({ qrUrl: `https://u:p@a.b${PATH}`, qrPath: PATH }, 'https://a.b')).toBe(`https://a.b${PATH}`);
    expect(qrTarget({ qrUrl: `javascript:alert(1)//${PATH}`, qrPath: PATH }, 'https://a.b')).toBe(`https://a.b${PATH}`);
    expect(qrTarget({ qrUrl: `https://a.b${PATH}?next=1`, qrPath: PATH }, 'https://a.b')).toBe(`https://a.b${PATH}`);
  });

  it('gives up without a usable origin or path', () => {
    expect(qrTarget({ qrUrl: null, qrPath: PATH }, '')).toBeNull();
    expect(qrTarget({ qrUrl: null, qrPath: PATH }, 'file:///x')).toBeNull();
    expect(qrTarget({ qrUrl: null, qrPath: '/fa/home' }, 'https://a.b')).toBeNull();
  });
});

describe('codeEntryUrl', () => {
  it('points at the public code page', () => {
    expect(codeEntryUrl('https://web.ritme.app/', 'fa')).toBe('https://web.ritme.app/fa/d');
  });
});

describe('toggles', () => {
  it('toggles sections', () => {
    expect(toggleSection(['basics', 'labs'], 'labs')).toEqual(['basics']);
    expect(toggleSection(['basics'], 'labs')).toEqual(['basics', 'labs']);
  });

  it('caps documents at 10', () => {
    const ten = Array.from({ length: 10 }, (_, i) => i + 1);
    expect(toggleDocument(ten, 11)).toEqual(ten);
    expect(toggleDocument(ten, 3)).toHaveLength(9);
    expect(toggleDocument([], 7)).toEqual([7]);
  });
});
