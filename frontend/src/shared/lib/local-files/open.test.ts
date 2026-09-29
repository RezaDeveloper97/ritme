import { describe, expect, it } from 'vitest';

import { blobForOpen, openModeFor } from './open';

const INLINE = ['image/jpeg', 'image/png', 'application/pdf'];

describe('openModeFor', () => {
  it('opens allow-listed types inline', () => {
    expect(openModeFor('image/jpeg', INLINE)).toBe('inline');
    expect(openModeFor('Application/PDF; x=1', INLINE)).toBe('inline');
  });

  it('forces a download for everything else — SVG and HTML above all', () => {
    expect(openModeFor('image/svg+xml', INLINE)).toBe('download');
    expect(openModeFor('text/html', INLINE)).toBe('download');
    expect(openModeFor('', INLINE)).toBe('download');
    expect(openModeFor('image/png', ['image/*'])).toBe('download');
  });
});

describe('blobForOpen', () => {
  it('re-types a download as octet-stream so the browser never renders it', () => {
    const svg = new Blob(['<svg onload="alert(1)"/>'], { type: 'image/svg+xml' });
    expect(blobForOpen(svg, 'download').type).toBe('application/octet-stream');
  });

  it('keeps the exact allow-listed type for an inline open', () => {
    expect(blobForOpen(new Blob(['x'], { type: 'image/PNG' }), 'inline').type).toBe('image/png');
  });
});
