import { describe, expect, it } from 'vitest';

import { createGate, sanitizeAppPath, urlBase64ToBytes } from './urls';

describe('sanitizeAppPath', () => {
  it('keeps in-app paths with query and hash', () => {
    expect(sanitizeAppPath('/fa/reminders')).toBe('/fa/reminders');
    expect(sanitizeAppPath('/fa/reminders?x=1#a')).toBe('/fa/reminders?x=1#a');
  });
  it.each(['//evil.com', '/\\evil.com', 'https://evil.com', 'javascript:alert(1)', 'fa/reminders', '', '/a\nb', '/a\u0000b'])(
    'rejects %j',
    (value) => {
      expect(sanitizeAppPath(value)).toBe('/');
    },
  );
  it('rejects non-strings and uses the fallback', () => {
    expect(sanitizeAppPath(undefined, '/home')).toBe('/home');
    expect(sanitizeAppPath(42)).toBe('/');
  });
});

describe('urlBase64ToBytes', () => {
  it('decodes base64url without padding', () => {
    expect(Array.from(urlBase64ToBytes('AQID'))).toEqual([1, 2, 3]);
    expect(Array.from(urlBase64ToBytes('-_8'))).toEqual([251, 255]);
  });
});

describe('createGate', () => {
  it('throttles to one run per gap and lets force through', () => {
    let t = 1000;
    const gate = createGate(60_000, () => t);
    expect(gate.tryEnter()).toBe(true);
    t += 59_000;
    expect(gate.tryEnter()).toBe(false);
    expect(gate.tryEnter(true)).toBe(true);
    t += 59_000;
    expect(gate.tryEnter()).toBe(false);
    t += 2_000;
    expect(gate.tryEnter()).toBe(true);
  });
});
