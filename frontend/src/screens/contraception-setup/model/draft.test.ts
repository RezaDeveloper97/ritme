import { describe, expect, it } from 'vitest';

import type { ContraceptionMethod } from '@/entities/contraception';

import { chooseMethod, draftProblems, draftToPayload, initialDraft, parseClock, toClock } from './draft';

const TODAY = '2026-10-01';

const savedPill: ContraceptionMethod = {
  method: 'combined_pill',
  packType: '24_4',
  packStartedOn: '2026-09-20',
  packsLeft: 2,
  reminder: { enabled: true, time: '08:30' },
  insertedOn: null,
  iudLifetimeYears: null,
  followupOn: null,
  followupDone: false,
  iudReplaceOn: null,
  injectedOn: null,
  nextInjectionOn: null,
  replaceOn: null,
};

describe('initialDraft', () => {
  it('starts empty with the board defaults', () => {
    const d = initialDraft(null, TODAY);
    expect(d).toMatchObject({ method: null, packType: '21_7', packStartedOn: TODAY, reminderTime: '21:00', iudYears: 10 });
  });

  it('edits the saved method', () => {
    expect(initialDraft(savedPill, TODAY)).toMatchObject({
      method: 'combined_pill',
      packType: '24_4',
      packStartedOn: '2026-09-20',
      reminderTime: '08:30',
      packsLeft: 2,
    });
  });
});

describe('chooseMethod', () => {
  it('moves the IUD lifetime to the device default', () => {
    const d = chooseMethod(initialDraft(null, TODAY), 'hormonal_iud', null);
    expect(d.iudYears).toBe(5);
    expect(chooseMethod(d, 'copper_iud', null).iudYears).toBe(10);
  });
});

describe('draftProblems', () => {
  it('needs a method', () => {
    expect(draftProblems(initialDraft(null, TODAY), TODAY)).toEqual({ method: 'missing' });
  });

  it('rejects a pack that starts in the future', () => {
    const d = { ...initialDraft(null, TODAY), method: 'combined_pill' as const, packStartedOn: '2026-10-02' };
    expect(draftProblems(d, TODAY)).toEqual({ packStartedOn: 'future' });
    expect(draftProblems({ ...d, packStartedOn: TODAY }, TODAY)).toEqual({});
  });

  it('requires the IUD and injection dates, not the implant one', () => {
    const base = initialDraft(null, TODAY);
    expect(draftProblems({ ...base, method: 'copper_iud' }, TODAY)).toEqual({ insertedOn: 'missing' });
    expect(draftProblems({ ...base, method: 'injection' }, TODAY)).toEqual({ injectedOn: 'missing' });
    expect(draftProblems({ ...base, method: 'implant' }, TODAY)).toEqual({});
    expect(draftProblems({ ...base, method: 'condom' }, TODAY)).toEqual({});
  });
});

describe('draftToPayload', () => {
  it('sends a combined pill with pack, start and reminder', () => {
    const d = { ...initialDraft(savedPill, TODAY), method: 'combined_pill' as const };
    expect(draftToPayload(d)).toEqual({
      method: 'combined_pill',
      pack_type: '24_4',
      pack_started_on: '2026-09-20',
      packs_left: 2,
      reminder_time: '08:30',
      reminder_enabled: true,
    });
  });

  it('drops the pack type of the progestogen-only pill', () => {
    const d = { ...initialDraft(savedPill, TODAY), method: 'progestin_pill' as const };
    expect(draftToPayload(d).pack_type).toBeNull();
  });

  it('sends only the IUD fields', () => {
    const d = { ...initialDraft(null, TODAY), method: 'copper_iud' as const, insertedOn: '2026-09-01' };
    expect(draftToPayload(d)).toEqual({ method: 'copper_iud', inserted_on: '2026-09-01', iud_lifetime_years: 10, followup_done: false });
  });
});

describe('clock', () => {
  it('round-trips HH:MM', () => {
    expect(parseClock('21:05')).toEqual({ hour: 21, minute: 5 });
    expect(toClock(9, 0)).toBe('09:00');
    expect(parseClock('nope')).toEqual({ hour: 21, minute: 0 });
  });
});
