import { describe, expect, it } from 'vitest';

import type { PregnancyAlertV2 } from '@/entities/pregnancy';

import { openKickAlerts } from './kick-alerts';

const alert = (over: Partial<PregnancyAlertV2>): PregnancyAlertV2 => ({
  id: 1,
  ruleKey: 'fetal_movement',
  level: 'urgent',
  title: 'Fewer than 10 movements in 120 minutes',
  whatWeSaw: null,
  howSure: null,
  advice: null,
  actions: [],
  contact: null,
  createdAt: null,
  factDate: null,
  dateLabel: null,
  isRead: false,
  isAcked: false,
  ...over,
});

describe('openKickAlerts', () => {
  it('keeps only urgent, unacknowledged fetal-movement alerts', () => {
    const list = [
      alert({ id: 1 }),
      alert({ id: 2, isAcked: true }),
      alert({ id: 3, level: 'follow_up' }),
      alert({ id: 4, ruleKey: 'bp_high' }),
    ];
    expect(openKickAlerts(list).map((a) => a.id)).toEqual([1]);
  });

  it('is empty without data', () => {
    expect(openKickAlerts(null)).toEqual([]);
  });
});
