import { describe, expect, it } from 'vitest';

import { isChunkLoadError, shouldReloadForChunkError } from './chunkReload';

describe('isChunkLoadError', () => {
  it('matches webpack ChunkLoadError by name', () => {
    const err = new Error('whatever');
    err.name = 'ChunkLoadError';
    expect(isChunkLoadError(err)).toBe(true);
  });

  it('matches the known browser/bundler messages', () => {
    expect(isChunkLoadError(new Error('Loading chunk 8123 failed.'))).toBe(true);
    expect(isChunkLoadError(new Error('Loading CSS chunk app/layout failed'))).toBe(true);
    expect(
      isChunkLoadError(new TypeError('Failed to fetch dynamically imported module: /x.js')),
    ).toBe(true);
    expect(isChunkLoadError(new TypeError('Importing a module script failed.'))).toBe(true);
  });

  it('ignores unrelated errors and non-errors', () => {
    expect(isChunkLoadError(new Error('Network Error'))).toBe(false);
    expect(isChunkLoadError('Loading chunk 1 failed')).toBe(false);
    expect(isChunkLoadError(null)).toBe(false);
    expect(isChunkLoadError(undefined)).toBe(false);
  });
});

describe('shouldReloadForChunkError', () => {
  const now = 1_000_000;

  it('reloads the first time', () => {
    expect(shouldReloadForChunkError(null, now)).toBe(true);
  });

  it('does not reload again right after a reload (loop guard)', () => {
    expect(shouldReloadForChunkError(now - 5_000, now)).toBe(false);
  });

  it('reloads again for a later, separate deploy', () => {
    expect(shouldReloadForChunkError(now - 60_000, now)).toBe(true);
  });

  it('does not trust a timestamp from the future', () => {
    expect(shouldReloadForChunkError(now + 60_000, now)).toBe(true);
  });
});
