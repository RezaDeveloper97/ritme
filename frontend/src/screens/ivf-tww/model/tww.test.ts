import { describe, expect, it } from 'vitest';

import { countdown, dangerCopy, followUps, guidanceBody, waitProgress } from './tww';

describe('countdown', () => {
  it('counts the days to the beta test', () => {
    expect(countdown(9, 2)).toEqual({ kind: 'days', days: 9 });
    expect(countdown(0, 11)).toEqual({ kind: 'today' });
    expect(countdown(-2, 13)).toEqual({ kind: 'passed' });
  });

  it('falls back to the day since transfer, then to unknown', () => {
    expect(countdown(null, 3)).toEqual({ kind: 'since', day: 3 });
    expect(countdown(null, null)).toEqual({ kind: 'unknown' });
  });
});

describe('waitProgress', () => {
  it('is the part of the wait already behind her', () => {
    expect(waitProgress(2, 9)).toBeCloseTo(2 / 11);
    expect(waitProgress(0, 11)).toBe(0);
  });

  it('is full on and after the beta day, empty while an end is unknown', () => {
    expect(waitProgress(11, 0)).toBe(1);
    expect(waitProgress(null, -1)).toBe(1);
    expect(waitProgress(null, 9)).toBe(0);
    expect(waitProgress(3, null)).toBe(0);
  });
});

describe('guidanceBody', () => {
  it('reads a catalog row by code', () => {
    const items = [{ code: 'early_test', title: 'T', body: 'Wait for the beta' }];
    expect(guidanceBody(items, 'early_test')).toBe('Wait for the beta');
    expect(guidanceBody(items, 'tww_feelings')).toBeNull();
    expect(guidanceBody(undefined, 'early_test')).toBeNull();
  });
});

describe('dangerCopy', () => {
  it('leads with OHSS and merges the hotlines', () => {
    const copy = dangerCopy([
      { code: 'fever_after_procedure', title: 'Fever', body: 'Report a fever', hotlines: ['115'] },
      { code: 'ohss', title: 'Tell your doctor', body: 'Severe bloating', hotlines: ['115'] },
    ]);
    expect(copy).toEqual({ title: 'Tell your doctor', bodies: ['Severe bloating', 'Report a fever'], hotlines: ['115'] });
  });

  it('keeps 115 when the catalog is empty', () => {
    expect(dangerCopy(undefined)).toEqual({ title: null, bodies: [], hotlines: ['115'] });
  });
});

describe('followUps', () => {
  it('uses the API steps', () => {
    expect(followUps('negative', ['loss', 'new_cycle'])).toEqual(['loss', 'new_cycle']);
    expect(followUps('positive', ['pregnancy_setup'])).toEqual(['pregnancy_setup']);
  });

  it('falls back per result', () => {
    expect(followUps('negative', [])).toEqual(['loss', 'new_cycle']);
    expect(followUps('cancelled', [])).toEqual(['new_cycle']);
  });

  it('never offers pregnancy setup or the loss path for the wrong result', () => {
    expect(followUps('cancelled', ['pregnancy_setup', 'loss', 'new_cycle'])).toEqual(['new_cycle']);
    expect(followUps('positive', ['loss', 'pregnancy_setup'])).toEqual(['pregnancy_setup']);
  });
});
