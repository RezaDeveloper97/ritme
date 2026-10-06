import { describe, expect, it } from 'vitest';

import {
  isBusy,
  labHref,
  nextPollDelay,
  POLL_FAST_COUNT,
  POLL_FAST_MS,
  POLL_MAX_COUNT,
  POLL_SLOW_MS,
  ringValue,
  stepStates,
  viewOf,
} from './status';
import type { LabStage, LabStatus } from './types';

describe('lab status state machine', () => {
  it('knows which statuses are still being worked on', () => {
    const busy: LabStatus[] = ['queued', 'extracting', 'interpreting'];
    const settled: LabStatus[] = ['needs_review', 'ready', 'failed'];
    busy.forEach((s) => expect(isBusy(s)).toBe(true));
    settled.forEach((s) => expect(isBusy(s)).toBe(false));
  });

  it('routes every status to its screen', () => {
    expect(viewOf('queued')).toBe('processing');
    expect(viewOf('extracting')).toBe('processing');
    expect(viewOf('interpreting')).toBe('processing');
    expect(viewOf('needs_review')).toBe('verify');
    expect(viewOf('ready')).toBe('result');
    expect(viewOf('failed')).toBe('failed');
    expect(labHref(7, 'extracting')).toBe('/labs/7/processing');
    expect(labHref(7, 'needs_review')).toBe('/labs/7/verify');
    expect(labHref(7, 'ready')).toBe('/labs/7');
    expect(labHref(7, 'failed')).toBe('/labs/7/processing');
  });

  it('polls fast, then slower, and stops when settled or after the cap', () => {
    expect(nextPollDelay(undefined, 0)).toBe(POLL_FAST_MS);
    expect(nextPollDelay('queued', 1)).toBe(POLL_FAST_MS);
    expect(nextPollDelay('extracting', POLL_FAST_COUNT - 1)).toBe(POLL_FAST_MS);
    expect(nextPollDelay('interpreting', POLL_FAST_COUNT)).toBe(POLL_SLOW_MS);
    expect(nextPollDelay('extracting', POLL_MAX_COUNT)).toBe(false);
    expect(nextPollDelay('needs_review', 3)).toBe(false);
    expect(nextPollDelay('ready', 3)).toBe(false);
    expect(nextPollDelay('failed', 3)).toBe(false);
  });

  it('walks the four processing steps with the stage', () => {
    expect(stepStates('queued')).toEqual({ read: 'current', extract: 'todo', compare: 'todo', explain: 'todo' });
    expect(stepStates('reading')).toEqual({ read: 'current', extract: 'todo', compare: 'todo', explain: 'todo' });
    expect(stepStates('extracting')).toEqual({ read: 'done', extract: 'current', compare: 'todo', explain: 'todo' });
    expect(stepStates('review')).toEqual({ read: 'done', extract: 'done', compare: 'current', explain: 'todo' });
    expect(stepStates('explaining')).toEqual({ read: 'done', extract: 'done', compare: 'done', explain: 'current' });
    expect(stepStates('done')).toEqual({ read: 'done', extract: 'done', compare: 'done', explain: 'done' });
    expect(stepStates('failed')).toEqual({ read: 'todo', extract: 'todo', compare: 'todo', explain: 'todo' });
  });

  it('never moves the ring backwards along the stages', () => {
    const order: [LabStage, number][] = [
      ['queued', 0],
      ['reading', 10],
      ['extracting', 60],
      ['extracting', 100],
      ['review', 100],
      ['explaining', 100],
      ['done', 100],
    ];
    let last = -1;
    for (const [stage, progress] of order) {
      const v = ringValue(stage, progress);
      expect(v).toBeGreaterThanOrEqual(last);
      expect(v).toBeLessThanOrEqual(1);
      last = v;
    }
    expect(ringValue('extracting', Number.NaN)).toBeGreaterThan(0);
  });
});
