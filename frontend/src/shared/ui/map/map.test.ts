import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { resolveNeshanKey } from '@/shared/config';

import { boundsToBbox, mapTypeFor, toLngLat } from './geo';
import { loadNeshanSdk } from './sdk';

async function freshMapModule(key: string | undefined) {
  vi.resetModules();
  vi.stubEnv('NEXT_PUBLIC_NESHAN_KEY', key);
  return import('./index');
}

afterEach(() => {
  vi.unstubAllEnvs();
  vi.resetModules();
});

describe('Neshan key resolution', () => {
  it('treats a missing, empty or blank key as no key', () => {
    expect(resolveNeshanKey(undefined)).toBeNull();
    expect(resolveNeshanKey(null)).toBeNull();
    expect(resolveNeshanKey('')).toBeNull();
    expect(resolveNeshanKey('   ')).toBeNull();
  });

  it('keeps a real key, trimmed', () => {
    expect(resolveNeshanKey(' web.abc ')).toBe('web.abc');
  });
});

describe('no-key fallback', () => {
  it('flags the map as unavailable so callers show the list', async () => {
    const { isMapEnabled } = await freshMapModule('');
    expect(isMapEnabled()).toBe(false);
  });

  it('renders nothing without a key', async () => {
    const { NeshanMap } = await freshMapModule(undefined);
    const html = renderToStaticMarkup(
      createElement(NeshanMap, { ariaLabel: 'map', pins: [{ id: 'a', label: 'A', lat: 35.7, lng: 51.4 }] }),
    );
    expect(html).toBe('');
  });

  it('flags the map as available once a key is set', async () => {
    const { isMapEnabled } = await freshMapModule('test-key');
    expect(isMapEnabled()).toBe(true);
  });

  it('never loads the SDK outside a browser', async () => {
    await expect(loadNeshanSdk()).rejects.toThrow('no DOM');
  });
});

describe('geo helpers', () => {
  it('swaps to mapbox [lng, lat] order', () => {
    expect(toLngLat({ lat: 35.7, lng: 51.4 })).toEqual([51.4, 35.7]);
  });

  it('normalises bounds to a rounded bbox', () => {
    const bbox = boundsToBbox({
      getSouth: () => 35.123456789,
      getWest: () => 51.1,
      getNorth: () => 35.8,
      getEast: () => 51.987654321,
    });
    expect(bbox).toEqual({ south: 35.123457, west: 51.1, north: 35.8, east: 51.987654 });
  });

  it('picks the Neshan night style for dark mode', () => {
    expect(mapTypeFor('light')).toBe('neshanVector');
    expect(mapTypeFor('dark')).toBe('neshanVectorNight');
  });
});
