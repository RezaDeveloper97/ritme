import { describe, expect, it } from 'vitest';

import { flagHintKind } from './flag-hint';

const flags = (over: Partial<{ frequency: string | null; duration: string | null; absent: boolean }> = {}) => ({
  frequency: 'normal',
  duration: 'normal',
  absent: false,
  ...over,
});

describe('flagHintKind', () => {
  it('maps each flag value', () => {
    expect(flagHintKind(flags({ frequency: 'frequent' }))).toBe('frequent');
    expect(flagHintKind(flags({ frequency: 'infrequent' }))).toBe('infrequent');
    expect(flagHintKind(flags({ duration: 'prolonged' }))).toBe('prolonged');
    expect(flagHintKind(flags({ absent: true }))).toBe('absent');
  });
  it('shows nothing for normal, unknown or missing flags (Laravel payload)', () => {
    expect(flagHintKind(flags())).toBeNull();
    expect(flagHintKind(flags({ frequency: 'new_label', duration: null }))).toBeNull();
    expect(flagHintKind(null)).toBeNull();
    expect(flagHintKind(undefined)).toBeNull();
  });
  it('puts absent first, then frequency, then duration', () => {
    expect(flagHintKind(flags({ absent: true, frequency: 'frequent' }))).toBe('absent');
    expect(flagHintKind(flags({ frequency: 'infrequent', duration: 'prolonged' }))).toBe('infrequent');
  });
});
