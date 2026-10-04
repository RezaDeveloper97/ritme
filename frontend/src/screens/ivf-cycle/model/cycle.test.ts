import { describe, expect, it } from 'vitest';

import { type IvfCycle, type IvfMed, toIvfCyclePatchBody, toIvfCycleStartBody } from '@/entities/ivf';

import {
  draftFromCycle,
  draftProblems,
  draftToPatch,
  emptySetup,
  joinWallClock,
  medsToStop,
  setDraftStage,
  setSetupStage,
  setupProblems,
  setupToInput,
  splitWallClock,
} from './cycle';

const TODAY = '2026-10-04';

const CYCLE: IvfCycle = {
  id: 3,
  number: 1,
  protocol: 'antagonist',
  stage: 'stim',
  status: 'open',
  startedOn: '2026-09-20',
  stimStartedOn: '2026-09-24',
  retrievalAt: null,
  transferAt: null,
  nextScanAt: '2026-10-05 09:30:00',
  betaOn: null,
  notifyCompanion: false,
  stageDay: 11,
  daysToBeta: null,
  timeline: [],
};

function med(over: Partial<IvfMed>): IvfMed {
  return {
    id: 1,
    reminderId: 10,
    name: 'FSH',
    role: 'stimulation',
    route: 'subcutaneous',
    isTrigger: false,
    triggerAt: null,
    dose: '150',
    unit: 'iu',
    times: ['20:00'],
    startsOn: '2026-09-24',
    endsOn: null,
    isActive: true,
    notes: null,
    inventory: null,
    ...over,
  };
}

describe('cycle setup (CB-IVF-06b)', () => {
  it('starts at preparation today and fills the stimulation start when she is already stimulating', () => {
    const draft = emptySetup(TODAY);
    expect(setupToInput(draft)).toEqual({ protocol: null, stage: 'prep', startedOn: TODAY, stimStartedOn: null });
    const stim = setSetupStage({ ...draft, startedOn: '2026-09-28' }, 'stim');
    expect(stim.stimStartedOn).toBe('2026-09-28');
    expect(setSetupStage(stim, 'prep').stimStartedOn).toBeNull();
  });

  it('rejects a future start and a stimulation start before it', () => {
    expect(setupProblems({ ...emptySetup(TODAY), startedOn: '2026-10-05' }, TODAY)).toEqual({ startedOn: 'future' });
    const stim = { protocol: null, stage: 'stim' as const, startedOn: '2026-09-28', stimStartedOn: '2026-09-27' };
    expect(setupProblems(stim, TODAY)).toEqual({ stimStartedOn: 'beforeStart' });
  });

  it('leaves null keys out of the POST body', () => {
    expect(toIvfCycleStartBody({ protocol: null, stage: 'prep', startedOn: TODAY, stimStartedOn: null })).toEqual({
      stage: 'prep',
      started_on: TODAY,
    });
  });
});

describe('stage + dates editor (CB-IVF-06b)', () => {
  it('splits and joins Tehran wall clocks', () => {
    expect(splitWallClock('2026-10-05 09:30:00')).toEqual({ date: '2026-10-05', time: '09:30' });
    expect(splitWallClock('2026-10-05')).toEqual({ date: '2026-10-05', time: '09:00' });
    expect(splitWallClock(null)).toEqual({ date: null, time: '09:00' });
    expect(joinWallClock({ date: '2026-10-05', time: '07:15' })).toBe('2026-10-05 07:15');
    expect(joinWallClock({ date: null, time: '07:15' })).toBeNull();
  });

  it('sends only what changed (a cleared date as null)', () => {
    const draft = draftFromCycle(CYCLE);
    expect(draftToPatch(draft, CYCLE)).toEqual({});
    const moved = {
      ...setDraftStage(draft, 'retrieval', TODAY),
      nextScan: { date: null, time: '09:30' },
      retrieval: { date: '2026-10-06', time: '08:00' },
    };
    const patch = draftToPatch(moved, CYCLE);
    expect(patch).toEqual({ stage: 'retrieval', nextScanAt: null, retrievalAt: '2026-10-06 08:00' });
    expect(toIvfCyclePatchBody(patch)).toEqual({
      stage: 'retrieval',
      next_scan_at: null,
      retrieval_at: '2026-10-06 08:00',
    });
  });

  it('fills a missing stimulation start when the stage moves to stimulation', () => {
    const prep = draftFromCycle({ ...CYCLE, stage: 'prep', stimStartedOn: null });
    expect(setDraftStage(prep, 'stim', TODAY).stimStartedOn).toBe(TODAY);
    expect(setDraftStage(draftFromCycle(CYCLE), 'stim', TODAY).stimStartedOn).toBe('2026-09-24');
  });

  it('checks the same order as the API', () => {
    const draft = draftFromCycle(CYCLE);
    expect(draftProblems(draft, TODAY)).toEqual({});
    expect(draftProblems({ ...draft, betaOn: '2026-09-01' }, TODAY)).toEqual({ betaOn: 'beforeStart' });
    const order = {
      ...draft,
      retrieval: { date: '2026-10-06', time: '09:00' },
      transfer: { date: '2026-10-06', time: '08:00' },
      betaOn: '2026-10-05',
    };
    expect(draftProblems(order, TODAY)).toEqual({ transfer: 'transfer', betaOn: 'beta' });
    expect(draftProblems({ ...draft, startedOn: '2026-10-09' }, TODAY).startedOn).toBe('future');
  });

  it('offers to stop active stimulation medicines only when moving past stimulation', () => {
    const meds = [
      med({ id: 1 }),
      med({ id: 2, role: 'suppression' }),
      med({ id: 3, role: 'trigger', isTrigger: true }),
      med({ id: 4, role: 'luteal_support' }),
      med({ id: 5, isActive: false }),
      med({ id: 6, reminderId: null }),
    ];
    expect(medsToStop(meds, 'stim', 'retrieval').map((m) => m.id)).toEqual([1, 2]);
    expect(medsToStop(meds, 'prep', 'tww').map((m) => m.id)).toEqual([1, 2]);
    expect(medsToStop(meds, 'prep', 'stim')).toEqual([]);
    expect(medsToStop(meds, 'retrieval', 'transfer')).toEqual([]);
  });
});
