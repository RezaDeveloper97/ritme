import { afterEach, describe, expect, it, vi } from 'vitest';

import {
  clampTextScaleIndex,
  DEFAULT_TEXT_SCALE_INDEX,
  isMotionPreference,
  reducesMotion,
  TEXT_SCALES,
  themeInitScript,
} from './index';

describe('text scale', () => {
  it('defaults to the middle stop (artboard thumb at 50%) = 100%', () => {
    expect(DEFAULT_TEXT_SCALE_INDEX).toBe(Math.floor(TEXT_SCALES.length / 2));
    expect(TEXT_SCALES[DEFAULT_TEXT_SCALE_INDEX]).toBe(1);
  });

  it('clamps stored junk to the default', () => {
    expect(clampTextScaleIndex('3')).toBe(3);
    expect(clampTextScaleIndex(0)).toBe(0);
    for (const junk of [null, '', 'big', -1, 99, 1.5]) {
      expect(clampTextScaleIndex(junk)).toBe(DEFAULT_TEXT_SCALE_INDEX);
    }
  });
});

describe('reduced motion', () => {
  it('accepts only system and reduce', () => {
    expect(isMotionPreference('system')).toBe(true);
    expect(isMotionPreference('reduce')).toBe(true);
    expect(isMotionPreference('full')).toBe(false);
    expect(isMotionPreference(null)).toBe(false);
  });

  it('an explicit reduce wins over the OS; system follows it', () => {
    expect(reducesMotion('reduce', false)).toBe(true);
    expect(reducesMotion('system', true)).toBe(true);
    expect(reducesMotion('system', false)).toBe(false);
  });
});

// The pre-paint bootstrap duplicates TEXT_SCALES and the motion key; prove they agree.
describe('themeInitScript display preferences', () => {
  afterEach(() => vi.unstubAllGlobals());

  function run(store: Record<string, string>) {
    const props: Record<string, string> = {};
    const root = {
      dataset: {} as Record<string, string>,
      style: { setProperty: (name: string, value: string) => (props[name] = value) },
    };
    vi.stubGlobal('localStorage', { getItem: (key: string) => store[key] ?? null });
    vi.stubGlobal('window', { matchMedia: () => ({ matches: false }) });
    vi.stubGlobal('document', { documentElement: root, querySelectorAll: () => [] });
    vi.stubGlobal('getComputedStyle', () => ({ getPropertyValue: () => '' }));
    new Function(themeInitScript)();
    return { scale: props['--text-scale'], motion: root.dataset.motion };
  }

  it('applies every stored text-scale stop exactly as the store does', () => {
    TEXT_SCALES.forEach((scale, index) => {
      expect(run({ ritme_text_scale: String(index) }).scale).toBe(String(scale));
    });
  });

  it('leaves the scale alone when nothing valid is stored', () => {
    expect(run({}).scale).toBeUndefined();
    expect(run({ ritme_text_scale: '42' }).scale).toBeUndefined();
  });

  it('forces reduced motion only for a stored reduce', () => {
    expect(run({ ritme_motion: 'reduce' }).motion).toBe('reduce');
    expect(run({ ritme_motion: 'system' }).motion).toBeUndefined();
  });
});
