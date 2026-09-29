import { describe, expect, it } from 'vitest';

import type { PregnancyAlertV2 } from '@/entities/pregnancy';

import { alertActionStyle, groupAlertsByDay, resolveAlertAction, visibleActions } from './alerts';

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
  factDate: null,
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

  it('puts the most important level first within a day, then the newest', () => {
    const g = groupAlertsByDay([
      alert({ id: 1, level: 'info', createdAt: '2026-09-29T09:00:00+03:30' }),
      alert({ id: 2, level: 'suggestion', createdAt: '2026-09-29T09:00:00+03:30' }),
      alert({ id: 3, level: 'follow_up', createdAt: '2026-09-29T08:00:00+03:30' }),
      alert({ id: 4, level: 'follow_up', createdAt: '2026-09-29T10:00:00+03:30' }),
      alert({ id: 5, level: 'urgent', createdAt: '2026-09-29T07:00:00+03:30' }),
    ]);
    expect(g[0].alerts.map((a) => a.id)).toEqual([5, 4, 3, 2, 1]);
  });

  it('groups by the fact date when the server sends one', () => {
    const g = groupAlertsByDay([
      alert({ id: 1, createdAt: '2026-09-29T09:00:00+03:30', factDate: '2026-09-26' }),
      alert({ id: 2, createdAt: '2026-09-29T09:00:00+03:30' }),
    ]);
    expect(g.map((x) => x.day)).toEqual(['2026-09-29', '2026-09-26']);
  });
});

describe('actions per level', () => {
  const ack = { r: { kind: 'server', key: 'ack', action: 'ack' } as const };
  const weight = { r: { kind: 'link', key: 'log_weight', href: '/x' } as const };
  it('styles and filters by level', () => {
    expect(alertActionStyle('urgent')).toBe('button');
    expect(alertActionStyle('follow_up')).toBe('button');
    expect(alertActionStyle('suggestion')).toBe('link');
    expect(alertActionStyle('info')).toBe('none');
    expect(visibleActions('follow_up', [weight, ack])).toEqual([weight, ack]);
    expect(visibleActions('suggestion', [weight, ack])).toEqual([weight]);
    expect(visibleActions('info', [weight, ack])).toEqual([]);
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
