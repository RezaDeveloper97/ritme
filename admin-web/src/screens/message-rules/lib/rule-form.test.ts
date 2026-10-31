import { describe, expect, it } from 'vitest';

import type { FactDef } from '@/features/rule-conditions';
import { ApiError } from '@/shared/api';

import { ruleSchema } from '../api/rules';
import { authorityFor } from './authority';
import { contentDiff, diffWords } from './diff';
import { bannerKey, fieldError } from './errors';
import { setOverlayField, FERTILE_TOPIC, emptyForm, emptyRow, fromRule, kindOfPriority, needsApproval, scheduleIn, scheduleOut, summarize, toPayload, weightSum } from './rule-form';

const FACTS: FactDef[] = [
  { name: 'mode', type: 'enum', enum: ['cycle', 'ttc'], unit: null, group: 'account', description: {}, member_safe: false, pending: null },
  { name: 'fertile_today', type: 'bool', enum: null, unit: null, group: 'cycle', description: {}, member_safe: false, pending: null },
];

const base = {
  id: 3, title: 'T', level: 'suggestion', kind: 'product', priority: 30, placement: 'push',
  content: { a: { title: 'a', body: 'b' }, b: { title: 'b', body: 'c' } },
  variants_enabled: true, ab_split: 40,
  conditions: [{ op: 'or', items: [{ fact: 'x', op: 'gte', value: 3 }] }], exceptions: [],
  cap_count: 2, cap_period: 'week', cooldown_days: 3, hours_from: 8, hours_to: 22,
  starts_on: '2026-10-01', ends_on: null, audience_percent: 50, goal: 'g',
  status: 'draft', is_active: false, version: 2, needs_medical_approval: false,
  approved_by: null, approved_at: null, updated_at: null,
};

describe('rule form', () => {
  it('knows the priority bands and the medical-approval rule', () => {
    expect(kindOfPriority(1000)).toBe('safety');
    expect(kindOfPriority(850)).toBe('engine');
    expect(kindOfPriority(60)).toBe('product');
    expect(kindOfPriority(0)).toBeNull();
    expect(needsApproval('info', 'product')).toBe(false);
    expect(needsApproval('worth_followup', 'product')).toBe(true);
    expect(needsApproval('info', 'clinical')).toBe(true);
  });

  it('round-trips a legacy a/b rule unchanged (no v6 columns appear)', () => {
    const rule = ruleSchema.parse(base);
    const back = toPayload(fromRule(rule), FACTS);
    expect(back.priority).toBe(30);
    expect(back.variants_enabled).toBe(true);
    expect(back.ab_split).toBe(40);
    expect(back.content).toEqual({ a: { title: 'a', body: 'b', cta_label: '', cta_target: '' }, b: { title: 'b', body: 'c', cta_label: '', cta_target: '' } });
    expect(back.variants).toBeUndefined();
    expect(back.caps).toBeUndefined();
    expect(back.hour_windows).toBeUndefined();
    expect(back.hours_from).toBe(8);
    expect(back.hours_to).toBe(22);
    expect(back.cap_count).toBe(2);
    expect(back.cap_period).toBe('week');
    expect(back.starts_on).toBe('2026-10-01');
    expect(back.starts_at).toBe('');
    expect(back.conditions).toEqual([{ op: 'or', items: [{ fact: 'x', op: 'gte', value: 3 }] }]);
    expect(summarize(rule.conditions)).toBe('x ≥ 3');
  });

  it('round-trips a full v6 rule', () => {
    const rule = ruleSchema.parse({
      ...base, kind: 'content', level: 'info', placement: 'daily_advice',
      rule_key: 'pms.cramps', variants: [{ key: 'a', weight: 2 }, { key: 'b', weight: 1 }, { key: 'c', weight: 1 }],
      content: { a: { title: 'a', body: 'b' }, b: { title: 'b', body: 'c' }, variants: { c: { title: 'c', body: 'd' } } },
      caps: [{ count: 1, period: 'day' }, { count: 3, period: 'week' }],
      hour_windows: [[8, 12], [18, 22]], starts_at: '2026-10-01 08:30:00', ends_at: '2026-11-01 00:00:00',
      topics: ['fertile_window'], goal_event: 'log_saved', locked: false, approved_version: 1, served_version: 1,
      cap_count: null, cap_period: null, hours_from: null, hours_to: null, starts_on: null, ab_split: 50,
    });
    const f = fromRule(rule);
    expect(f.variants.map((v) => v.key)).toEqual(['a', 'b', 'c']);
    expect(weightSum(f.variants)).toBe(4);
    const p = toPayload(f, FACTS);
    expect(p.variants).toEqual([{ key: 'a', weight: 2 }, { key: 'b', weight: 1 }, { key: 'c', weight: 1 }]);
    expect((p.content as { variants: unknown }).variants).toEqual({ c: { title: 'c', body: 'd', cta_label: '', cta_target: '' } });
    expect(p.caps).toEqual([{ count: 1, period: 'day' }, { count: 3, period: 'week' }]);
    expect(p.hour_windows).toEqual([[8, 12], [18, 22]]);
    expect(p.hours_from).toBeNull();
    expect(p.starts_at).toBe('2026-10-01 08:30:00');
    expect(p.ends_at).toBe('2026-11-01 00:00:00');
    expect(p.rule_key).toBe('pms.cramps');
    expect(p.topics).toEqual(['fertile_window']);
    expect(p.goal_event).toBe('log_saved');
  });

  it('moves a legacy rule to the v6 columns only when it outgrows the a/b shape', () => {
    const f = fromRule(ruleSchema.parse(base));
    f.variants = [...f.variants, emptyRow('c', '10')];
    f.caps = [...f.caps, { count: '1', period: 'day' }];
    f.hours = [...f.hours, { from: '1', to: '2' }];
    const p = toPayload(f, FACTS);
    expect(p.variants).toHaveLength(3);
    expect(p.caps).toHaveLength(2);
    expect(p.hour_windows).toEqual([[8, 22], [1, 2]]);
  });

  it('adds the fertile_window topic when a fertility fact is read, and drops caps for safety', () => {
    const f = emptyForm();
    f.title = 'x';
    f.conditions.allOf = [{ fact: 'fertile_today', op: 'eq', value: 'true' }];
    expect(toPayload(f, FACTS, ['fertile_today']).topics).toEqual([FERTILE_TOPIC]);
    expect(toPayload(f, FACTS, []).topics).toBeUndefined();
    f.kind = 'safety';
    const p = toPayload(f, FACTS);
    expect(p.cap_count).toBeNull();
    expect(p.caps).toBeUndefined();
  });

  it('converts the datetime-local format', () => {
    expect(scheduleIn('2026-10-01 08:30:00')).toBe('2026-10-01T08:30');
    expect(scheduleOut('2026-10-01T08:30')).toBe('2026-10-01 08:30:00');
    expect(scheduleOut('')).toBe('');
  });
});

describe('authority', () => {
  const kinds = ['safety', 'clinical', 'content'];
  it('makes clinical kinds read-only without the clinical capability', () => {
    expect(authorityFor({ kind: 'clinical', locked: false }, false, kinds).readOnly).toBe(true);
    expect(authorityFor({ kind: 'clinical', locked: false }, true, kinds).readOnly).toBe(false);
    expect(authorityFor({ kind: 'product', locked: false }, false, kinds).readOnly).toBe(false);
  });
  it('hides delete and switch-off of locked rules, and lets only clinical edit their copy', () => {
    const a = authorityFor({ kind: 'engine', locked: true }, true, kinds);
    expect(a.canDelete).toBe(false);
    expect(a.canSwitchOff).toBe(false);
    expect(a.copyOnly).toBe(true);
    expect(a.readOnly).toBe(false);
    expect(authorityFor({ kind: 'engine', locked: true }, false, kinds).readOnly).toBe(true);
  });
  it('hides toggle and delete of safety rules without the clinical capability', () => {
    const a = authorityFor({ kind: 'safety', locked: false }, false, kinds);
    expect(a.canDelete).toBe(false);
    expect(a.canSwitchOff).toBe(false);
    expect(authorityFor({ kind: 'safety', locked: false }, true, kinds).canDelete).toBe(true);
  });
  it('lets only the clinical capability author safety / clinical / content', () => {
    expect(authorityFor(null, false, kinds).canAuthorKind('content')).toBe(false);
    expect(authorityFor(null, false, kinds).canAuthorKind('product')).toBe(true);
    expect(authorityFor(null, true, kinds).canAuthorKind('safety')).toBe(true);
  });
});

describe('error mapping', () => {
  const e = (status: number, code: string, fieldErrors: Record<string, string[]> = {}) => new ApiError({ status, code, fieldErrors });
  it('maps 422 codes and 409 / 403 answers to banners', () => {
    expect(bannerKey(e(422, 'unknown_fact'))).toBe('errUnknownFact');
    expect(bannerKey(e(422, 'bad_value'))).toBe('errBadValue');
    expect(bannerKey(e(409, 'locked'))).toBe('errLocked');
    expect(bannerKey(e(403, 'forbidden'))).toBe('errForbiddenClinical');
    expect(bannerKey(e(422, 'validation_failed'))).toBeNull();
    expect(bannerKey(new Error('x'))).toBeNull();
  });
  it('maps server fields onto editor fields', () => {
    const err = e(422, 'bad_value', { conditions: ['«mode»: unknown value'], hour_windows: ['bad'], rule_key: ['taken'] });
    expect(fieldError(err, 'conditions')).toBe('«mode»: unknown value');
    expect(fieldError(err, 'hours')).toBe('bad');
    expect(fieldError(err, 'rule_key')).toBe('taken');
    expect(fieldError(err, 'title')).toBeUndefined();
  });
});

describe('approval diff', () => {
  it('marks added and removed words', () => {
    const d = diffWords('drink water often', 'drink more water');
    expect(d.filter((p) => p.type === 'add').map((p) => p.text)).toEqual(['more']);
    expect(d.filter((p) => p.type === 'del').map((p) => p.text)).toEqual(['often']);
  });
  it('lists only the fields that changed', () => {
    const out = contentDiff({ a: { title: 'T', body: 'old' } }, { a: { title: 'T', body: 'new' } });
    expect(out.map((o) => o.label)).toEqual(['a.body']);
  });
});

describe('content.translations overlay', () => {
  const seeded = {
    ...base,
    variants_enabled: false,
    content: {
      a: { title: 'ب', body: 'ج', cta_label: 'د', cta_target: 'ritme://period/end' },
      translations: { en: { a: { title: 'Period end', body: 'Log it', cta_label: 'Log', cta_target: 'ritme://period/end' } } },
    },
  };

  it('keeps the stored overlay byte-identical on an untouched save', () => {
    const rule = ruleSchema.parse(seeded);
    const content = toPayload(fromRule(rule), FACTS).content as Record<string, unknown>;
    expect(JSON.stringify(content.translations)).toBe(JSON.stringify(seeded.content.translations));
  });

  it('editing en changes only that variant overlay', () => {
    const rule = ruleSchema.parse(seeded);
    const f = fromRule(rule);
    f.translations = setOverlayField(f.translations, 'en', 'a', 'title', 'New');
    const content = toPayload(f, FACTS).content as { a: unknown; translations: Record<string, Record<string, unknown>> };
    expect(content.translations.en?.a).toEqual({ ...seeded.content.translations.en.a, title: 'New' });
    expect(content.a).toEqual(seeded.content.a);
  });

  it('clearing every field removes the entry, and a rule without overlay sends none', () => {
    expect(setOverlayField({ en: { a: { title: 'x' } } }, 'en', 'a', 'title', '')).toBeNull();
    const plain = toPayload(fromRule(ruleSchema.parse(base)), FACTS).content as Record<string, unknown>;
    expect('translations' in plain).toBe(false);
  });
});
