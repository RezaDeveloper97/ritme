import { describe, expect, it } from 'vitest';

import { formatDuration, lessonActions, noteError, reviewHref, reviewTone } from './review';

describe('review helpers', () => {
  it('maps states to tones', () => {
    expect(reviewTone('flagged')).toBe('red');
    expect(reviewTone('changed')).toBe('data');
    expect(reviewTone('weird')).toBe('neutral');
  });

  it('links the default tab without a parameter', () => {
    expect(reviewHref('open')).toBe('/learning/reviews');
    expect(reviewHref('flagged')).toBe('/learning/reviews?state=flagged');
  });

  it('offers only meaningful actions', () => {
    expect(lessonActions({ status: 'published', review: { state: 'pending' } })).toEqual({
      approve: true,
      flag: true,
      unpublish: true,
    });
    expect(lessonActions({ status: 'draft', review: { state: 'flagged' } })).toEqual({
      approve: true,
      flag: false,
      unpublish: false,
    });
    expect(lessonActions({ status: 'published', review: { state: 'approved' } }).approve).toBe(false);
  });

  it('validates notes like the API', () => {
    expect(noteError('  ', true)).toBe('required');
    expect(noteError('', false)).toBeNull();
    expect(noteError('ab', false)).toBe('min');
    expect(noteError('x'.repeat(301), true)).toBe('max');
    expect(noteError('ادعای نادرست', true)).toBeNull();
  });

  it('formats durations', () => {
    expect(formatDuration(null)).toBeNull();
    expect(formatDuration(65)).toBe('1:05');
    expect(formatDuration(3725)).toBe('1:02:05');
  });
});
