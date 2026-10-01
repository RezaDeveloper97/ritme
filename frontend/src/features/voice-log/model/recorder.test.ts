import { describe, expect, it } from 'vitest';

import { formatElapsed, pickMimeType, recorderFailure } from './recorder';

describe('recorder helpers', () => {
  it('picks the first container the browser records', () => {
    expect(pickMimeType((t) => t === 'audio/mp4')).toBe('audio/mp4');
    expect(pickMimeType((t) => t.startsWith('audio/webm'))).toBe('audio/webm;codecs=opus');
    expect(pickMimeType(() => false)).toBeUndefined();
    expect(pickMimeType(undefined)).toBeUndefined();
    expect(
      pickMimeType(() => {
        throw new Error('old Safari');
      }),
    ).toBeUndefined();
  });

  it('formats the timer as m:ss', () => {
    expect(formatElapsed(7_400)).toBe('0:07');
    expect(formatElapsed(60_000)).toBe('1:00');
    expect(formatElapsed(-5)).toBe('0:00');
  });

  it('maps permission and device failures', () => {
    expect(recorderFailure({ name: 'NotAllowedError' })).toBe('denied');
    expect(recorderFailure({ name: 'SecurityError' })).toBe('denied');
    expect(recorderFailure({ name: 'NotFoundError' })).toBe('nomic');
    expect(recorderFailure({ name: 'NotSupportedError' })).toBe('unsupported');
    expect(recorderFailure(new Error('boom'))).toBe('error');
    expect(recorderFailure(null)).toBe('error');
  });
});
