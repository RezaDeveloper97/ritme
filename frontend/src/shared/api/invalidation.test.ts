import type { QueryClient } from '@tanstack/react-query';
import { describe, expect, it, vi } from 'vitest';

import { analysisKeys } from '@/entities/analysis';
import { careKeys } from '@/entities/care-reminder';
import { cycleKeys } from '@/entities/cycle';
import { fertilityKeys } from '@/entities/fertility';
import { messageKeys } from '@/entities/message';

import { CYCLE_DEPENDENT_ROOTS, invalidateCycleDependents } from './invalidation';

describe('CYCLE_DEPENDENT_ROOTS', () => {
  it('equals the entity key factories` roots (shared cannot import them, so it is pinned here)', () => {
    const roots = CYCLE_DEPENDENT_ROOTS.map((r) => JSON.stringify(r));
    for (const all of [
      cycleKeys.all,
      fertilityKeys.all,
      analysisKeys.all,
      messageKeys.all,
      careKeys.all,
      ['period-status'],
      ['period-history'],
    ]) {
      expect(roots).toContain(JSON.stringify(all));
    }
    expect(roots).toHaveLength(7);
  });
});

describe('invalidateCycleDependents', () => {
  it('invalidates every root', () => {
    const invalidateQueries = vi.fn().mockResolvedValue(undefined);
    invalidateCycleDependents({ invalidateQueries } as unknown as QueryClient);
    expect(invalidateQueries.mock.calls.map(([arg]) => arg.queryKey)).toEqual(
      CYCLE_DEPENDENT_ROOTS.map((r) => [...r]),
    );
  });
});
