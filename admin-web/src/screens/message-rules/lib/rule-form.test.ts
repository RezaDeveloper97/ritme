import { describe, expect, it } from 'vitest';

import { ruleSchema } from '../api/rules';
import { emptyForm, formatValue, fromRule, kindOfPriority, needsApproval, parseValue, summarize, toPayload } from './rule-form';

describe('rule form', () => {
  it('parses condition values by operator', () => {
    expect(parseValue('in', 'cycle, ttc')).toEqual(['cycle', 'ttc']);
    expect(parseValue('between', '1, 4')).toEqual([1, 4]);
    expect(parseValue('gte', '7')).toBe(7);
    expect(parseValue('eq', 'true')).toBe(true);
    expect(parseValue('eq', 'late_luteal')).toBe('late_luteal');
    expect(formatValue([1, 4])).toBe('1, 4');
  });

  it('knows the priority bands and the medical-approval rule', () => {
    expect(kindOfPriority(1000)).toBe('safety');
    expect(kindOfPriority(850)).toBe('engine');
    expect(kindOfPriority(60)).toBe('product');
    expect(kindOfPriority(0)).toBeNull();
    expect(needsApproval('info', 'product')).toBe(false);
    expect(needsApproval('worth_followup', 'product')).toBe(true);
    expect(needsApproval('info', 'clinical')).toBe(true);
  });

  it('builds the payload: blank conditions dropped, B only with variants, medical fields only for their levels', () => {
    const f = emptyForm();
    f.title = ' PMS ';
    f.a = { title: 'A', body: 'body', cta_label: '', cta_target: '' };
    f.groups = [{ op: 'and', items: [{ fact: 'mode', op: 'in', value: 'cycle, ttc' }, { fact: '', op: 'eq', value: '' }] }];
    f.seen = 'x';
    const p = toPayload(f);
    expect(p.title).toBe('PMS');
    expect(p.conditions).toEqual([{ op: 'and', items: [{ fact: 'mode', op: 'in', value: ['cycle', 'ttc'] }] }]);
    expect((p.content as Record<string, unknown>).b).toBeUndefined();
    expect((p.content as Record<string, unknown>).seen).toBeUndefined();
    expect(p.ab_split).toBeNull();
    f.level = 'follow_soon';
    f.variants_enabled = true;
    f.contact = 'tel:115';
    const q = toPayload(f);
    expect((q.content as Record<string, unknown>).contact).toBe('tel:115');
    expect((q.content as Record<string, unknown>).b).toBeDefined();
    f.cap_count = '';
    expect(toPayload(f).cap_period).toBe('');
  });

  it('round-trips an API rule through the form', () => {
    const rule = ruleSchema.parse({
      id: 3, title: 'T', level: 'suggestion', kind: 'product', priority: 30, placement: 'push',
      content: { a: { title: 'a', body: 'b' }, b: { title: 'b', body: 'c' } },
      variants_enabled: true, ab_split: 40,
      conditions: [{ op: 'or', items: [{ fact: 'x', op: 'gte', value: 3 }] }], exceptions: [],
      cap_count: 2, cap_period: 'week', cooldown_days: 3, hours_from: 8, hours_to: 22,
      starts_on: '2026-10-01', ends_on: null, audience_percent: 50, goal: 'g',
      status: 'draft', is_active: false, version: 2, needs_medical_approval: false,
      approved_by: null, approved_at: null, updated_at: null,
    });
    const f = fromRule(rule);
    expect(f.groups[0]?.items[0]?.value).toBe('3');
    const back = toPayload(f);
    expect(back.priority).toBe(30);
    expect(back.ab_split).toBe(40);
    expect(back.hours_from).toBe(8);
    expect(back.cap_count).toBe(2);
    expect(summarize(rule.conditions)).toBe('x ≥ 3');
  });
});
