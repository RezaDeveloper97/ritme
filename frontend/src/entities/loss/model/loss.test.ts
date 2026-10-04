import { describe, expect, it } from 'vitest';

import {
  crisisHotlines,
  emergencyNumber,
  lifeModeOfNextStep,
  shouldRecordLoss,
  splitWallClock,
  visitAt,
  warningSignParts,
} from './loss';
import type { LossCatalogItem, LossEvent } from './types';

const item = (code: string, title: string | null, body: string | null = null, meta: Record<string, unknown> | null = null): LossCatalogItem => ({
  code,
  title,
  body,
  meta,
});

const loss = (createdAt: string): LossEvent => ({
  id: 1,
  type: 'unspecified',
  occurredOn: null,
  notifyCompanion: false,
  companionNotified: false,
  contentStopped: true,
  nextStep: null,
  createdAt,
});

describe('lifeModeOfNextStep', () => {
  it('follows the catalog meta, else ttc → ttc and everything else → cycle', () => {
    expect(lifeModeOfNextStep('ttc')).toBe('ttc');
    expect(lifeModeOfNextStep('nothing')).toBe('cycle');
    expect(lifeModeOfNextStep('cycle', item('cycle', null, null, { life_mode: 'ttc' }))).toBe('ttc');
    expect(lifeModeOfNextStep('ttc', item('ttc', null, null, { life_mode: 'bogus' }))).toBe('ttc');
  });
});

describe('emergencyNumber', () => {
  it('takes a warning sign hotline, then the emergency line, then 115', () => {
    expect(emergencyNumber([item('fever', 'x', null, { hotline: '112' })], [])).toBe('112');
    expect(emergencyNumber([], [item('emergency', 'x', null, { number: '115', kind: 'medical' })])).toBe('115');
    expect(emergencyNumber([item('fever', 'x', null, { hotline: 'call me' })], [])).toBe('115');
  });
});

describe('warningSignParts', () => {
  it('joins a title with its detail and skips untitled items', () => {
    expect(warningSignParts([item('a', 'خونریزی', 'دو نوار'), item('b', 'تب'), item('c', null, 'x')])).toEqual([
      'خونریزی (دو نوار)',
      'تب',
    ]);
  });

  it('drops the stand-alone capitals inside the joined English sentence', () => {
    expect(
      warningSignParts([item('a', 'Very heavy bleeding', 'Soaking 2 pads'), item('b', 'A fever'), item('c', 'hCG still high'), item('d', 'IVF')]),
    ).toEqual(['Very heavy bleeding (soaking 2 pads)', 'a fever', 'hCG still high', 'IVF']);
  });
});

describe('crisisHotlines', () => {
  it('reads the crisis meta, else the non-medical hotlines', () => {
    const crisis = item('crisis', 't', 'b', { hotlines: [{ number: '1480', label: 'صدای مشاور' }, { number: 'x' }] });
    expect(crisisHotlines(crisis, [])).toEqual([{ number: '1480', label: 'صدای مشاور' }]);
    const hotlines = [
      item('emergency', 'اورژانس ۱۱۵', null, { number: '115', kind: 'medical' }),
      item('counselling', 'صدای مشاور ۱۴۸۰', null, { number: '1480', kind: 'mental_health' }),
      item('social_emergency', 'اورژانس اجتماعی ۱۲۳', null, { number: '123', kind: 'crisis' }),
    ];
    expect(crisisHotlines(null, hotlines).map((h) => h.number)).toEqual(['1480', '123']);
  });
});

describe('shouldRecordLoss', () => {
  it('records a first loss, a same-day correction, or a loss in a new pregnancy — never re-posts an old one', () => {
    expect(shouldRecordLoss(null, false, '2026-09-23')).toBe(true);
    expect(shouldRecordLoss(loss('2026-09-23T10:00:00+03:30'), false, '2026-09-23')).toBe(true);
    expect(shouldRecordLoss(loss('2026-09-20T10:00:00+03:30'), false, '2026-09-23')).toBe(false);
    expect(shouldRecordLoss(loss('2026-09-20T10:00:00+03:30'), true, '2026-09-23')).toBe(true);
  });
});

describe('wall clock', () => {
  it('builds and splits the visit time', () => {
    expect(visitAt('2026-10-07', '11:30')).toBe('2026-10-07 11:30');
    expect(splitWallClock('2026-10-07 11:30:00')).toEqual({ day: '2026-10-07', time: '11:30' });
  });
});
