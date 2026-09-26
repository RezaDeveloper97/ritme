import { describe, expect, it } from 'vitest';

import type { PregnancyAlertV2 } from '@/entities/pregnancy';

import { groupAlertsByDay, resolveAlertAction } from './alerts';

const alert = (p: Partial<PregnancyAlertV2>): PregnancyAlertV2 => ({
  id: 1,
  ruleKey: null,
  level: 'info',
  title: 't',
  whatWeSaw: null,
  howSure: null,
  advice: null,
  actions: [],
  contact: null,
  createdAt: null,
  dateLabel: null,
  isRead: false,
  isAcked: false,
  ...p,
});

describe('groupAlertsByDay', () => {
  it('groups by day, newest first', () => {
    const g = groupAlertsByDay([
      alert({ id: 1, createdAt: '2026-09-20T10:00:00+03:30' }),
      alert({ id: 2, createdAt: '2026-09-22T10:00:00+03:30' }),
      alert({ id: 3, createdAt: '2026-09-20T08:00:00+03:30' }),
    ]);
    expect(g.map((x) => x.day)).toEqual(['2026-09-22', '2026-09-20']);
    expect(g[1].alerts.map((a) => a.id)).toEqual([1, 3]);
  });
});

describe('resolveAlertAction', () => {
  it('maps keys to behaviours', () => {
    const a = alert({ level: 'urgent', contact: { text: 'x', phone: '0912 000 1234' } });
    expect(resolveAlertAction({ key: 'ack', label: null }, a)).toEqual({ kind: 'server', key: 'ack', action: 'ack' });
    expect(resolveAlertAction({ key: 'log_weight', label: null }, a)).toEqual({
      kind: 'link',
      key: 'log_weight',
      href: '/pregnancy/log?focus=weight',
    });
    expect(resolveAlertAction({ key: 'call', label: null }, a)).toEqual({ kind: 'tel', key: 'call', href: 'tel:09120001234' });
    expect(resolveAlertAction({ key: 'nope', label: null }, a)).toBeNull();
    expect(resolveAlertAction({ key: 'ack', label: null }, { ...a, isAcked: true })).toBeNull();
  });
});
