import { describe, expect, it } from 'vitest';

import { cycleViewSchema } from '../api/schema';

import { isFertilitySuppressed } from './suppression';

const view = (extra: Record<string, unknown>) =>
  cycleViewSchema.parse({ date: '2026-07-18', ...extra });

describe('isFertilitySuppressed', () => {
  it('is false without a view', () => {
    expect(isFertilitySuppressed(null)).toBe(false);
    expect(isFertilitySuppressed(undefined)).toBe(false);
  });

  it('is false for a plain Laravel view', () => {
    expect(isFertilitySuppressed(view({}))).toBe(false);
  });

  it('follows limits.fertility_suppressed', () => {
    expect(isFertilitySuppressed(view({ limits: { fertility_suppressed: true } }))).toBe(true);
    expect(isFertilitySuppressed(view({ limits: { fertility_suppressed: false } }))).toBe(false);
  });

  it('falls back to the suppressing warnings when limits is absent', () => {
    expect(isFertilitySuppressed(view({ warnings: ['hormonal_contraception_active'] }))).toBe(true);
    expect(isFertilitySuppressed(view({ warnings: ['postpartum_cycles_not_returned'] }))).toBe(
      true,
    );
    expect(isFertilitySuppressed(view({ warnings: ['low_history'] }))).toBe(false);
  });

  it('limits wins as true even with no warnings; a clear limits does not hide a warning', () => {
    expect(
      isFertilitySuppressed(view({ limits: { fertility_suppressed: true }, warnings: [] })),
    ).toBe(true);
    expect(
      isFertilitySuppressed(
        view({
          limits: { fertility_suppressed: false },
          warnings: ['hormonal_contraception_active'],
        }),
      ),
    ).toBe(true);
  });
});
