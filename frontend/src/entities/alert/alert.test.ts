import { describe, expect, it } from 'vitest';

import { parseAlertPage } from './api/schema';
import type { AlertItem } from './model/types';
import { callHref, canAck, canAddToVisitNote, canDismiss, evidenceRows, groupByLevel } from './model/view';

const row = (p: Record<string, unknown> = {}) => ({
  id: 1, level: 'info', state: 'active', title: 't', body: 'b', cta_text: null, cta_deeplink: null,
  evidence: null, hit_count: 1, dismissible: true, last_at: '2026-10-30T09:00:00+03:30', acked_at: null, ...p,
});
const item = (p: Record<string, unknown> = {}): AlertItem => parseAlertPage({ items: [row(p)] }).items[0];

describe('parseAlertPage', () => {
  it('maps the API row and the cursor', () => {
    const page = parseAlertPage({
      items: [row({ id: 7, level: 'follow_up', evidence: { seen: 'دو بار', confidence: 'از ثبت‌های خودت' } })],
      next_cursor: 'abc',
    });
    expect(page.nextCursor).toBe('abc');
    expect(page.items[0]).toMatchObject({ id: 7, level: 'follow_up', evidence: { seen: 'دو بار', confidence: 'از ثبت‌های خودت', contact: null } });
  });
  it('drops rows without an id and survives garbage', () => {
    expect(parseAlertPage({ items: [{ title: 'x' }, row()], next_cursor: null }).items).toHaveLength(1);
    expect(parseAlertPage(null)).toEqual({ items: [], nextCursor: null });
  });
  it('falls back for unknown level or state', () => {
    expect(item({ level: 'weird', state: 'weird' })).toMatchObject({ level: 'info', state: 'active' });
  });
  it('never trusts dismissible for an unacknowledged urgent item', () => {
    expect(item({ level: 'urgent', dismissible: true }).dismissible).toBe(false);
  });
});

describe('groupByLevel', () => {
  it('orders urgent, follow_up, suggestion, info and drops empty levels; newest first inside', () => {
    const items = ['info', 'urgent', 'suggestion', 'follow_up', 'info'].map((level, i) =>
      item({ id: i + 1, level, last_at: `2026-10-0${i + 1}T00:00:00+03:30` }),
    );
    const sections = groupByLevel(items);
    expect(sections.map((s) => s.level)).toEqual(['urgent', 'follow_up', 'suggestion', 'info']);
    expect(sections[3].items.map((x) => x.id)).toEqual([5, 1]);
    expect(groupByLevel([item({ level: 'info' })]).map((s) => s.level)).toEqual(['info']);
  });
});

describe('urgent close rule', () => {
  it('has no close action until acknowledged', () => {
    expect(canDismiss(item({ level: 'urgent', state: 'active' }))).toBe(false);
    expect(canDismiss(item({ level: 'urgent', state: 'acknowledged' }))).toBe(true);
    expect(canDismiss(item({ level: 'follow_up', state: 'active' }))).toBe(true);
  });
  it('offers nothing on history items', () => {
    const h = item({ state: 'archived' });
    expect([canAck(h), canDismiss(h), canAddToVisitNote(h)]).toEqual([false, false, false]);
  });
  it('offers the visit note once and the ack only while active', () => {
    expect(canAddToVisitNote(item({ state: 'added_to_visit_note' }))).toBe(false);
    expect(canAck(item({ state: 'acknowledged' }))).toBe(false);
    expect(canAck(item())).toBe(true);
  });
});

describe('evidence rendering', () => {
  const ev = { seen: 'a', confidence: 'b', contact: '115' };
  it('shows both lines on follow_up and urgent only', () => {
    expect(evidenceRows(item({ level: 'follow_up', evidence: ev })).map((r) => r.key)).toEqual(['seen', 'confidence']);
    expect(evidenceRows(item({ level: 'info', evidence: ev }))).toEqual([]);
    expect(evidenceRows(item({ level: 'urgent' }))).toEqual([]);
  });
  it('builds a call link for a phone contact on urgent items only', () => {
    expect(callHref(item({ level: 'urgent', evidence: ev }))).toBe('tel:115');
    expect(callHref(item({ level: 'urgent', evidence: { ...ev, contact: 'اورژانس بیمارستان' } }))).toBeNull();
    expect(callHref(item({ level: 'follow_up', evidence: ev }))).toBeNull();
  });
});
