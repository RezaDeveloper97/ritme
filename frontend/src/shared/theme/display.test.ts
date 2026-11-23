import { describe, expect, it } from 'vitest';

import {
  clampTextScaleIndex,
  DEFAULT_TEXT_SCALE_INDEX,
  isMotionPreference,
  reducesMotion,
  TEXT_SCALES,
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
