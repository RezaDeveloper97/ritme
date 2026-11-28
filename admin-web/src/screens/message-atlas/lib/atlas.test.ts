import { describe, expect, it } from 'vitest';

import fa from '../../../../messages/fa.json';
import type { AtlasRule } from '../api/atlas';
import {
  bySurface,
  conditionText,
  entryOfRule,
  filterEntries,
  GROUP_ENTRIES,
  groupEntries,
  modesOfRule,
  SURFACES,
  surfaceOfPlacement,
  whenOfRule,
  type ConditionWords,
} from './atlas';

const words: ConditionWords = {
  fact: (n) => ({ days_late: 'روزهای تأخیر', 'preg.bp_high_level': 'سطح فشار', has_log: 'ثبت امروز' })[n] ?? n,
  op: (op) => ({ gte: 'دست‌کم', eq: 'برابر', in: 'یکی از' })[op] ?? op,
  yes: 'بله',
  no: 'خیر',
  listJoin: '، ',
  and: ' و ',
  or: ' یا ',
  between: (a, b) => `بین ${a} و ${b}`,
  num: String,
};

const rule = (over: Partial<AtlasRule> = {}): AtlasRule => ({
  id: 1,
  title: 'تأخیر',
  rule_key: 'banner.overdue',
  level: 'info',
  kind: 'engine',
  priority: 500,
  placement: 'status_banner',
  content: { a: { title: 'پریودت عقب افتاده', body: '…', cta_label: 'ثبت', cta_target: '' } },
  conditions: [{ op: 'and', items: [{ fact: 'days_late', op: 'gte', value: 5 }] }],
  exceptions: [],
  status: 'approved',
  is_active: true,
  locked: false,
  ...over,
});
const groupOf = (f: string) => ({ days_late: 'engine', 'preg.bp_high_level': 'pregnancy_alerts', 'meno.stage': 'menopause' })[f];

describe('message atlas', () => {
  it('covers every message group, each with its «when» and a known surface', () => {
    expect(Object.keys(GROUP_ENTRIES).sort()).toEqual(Object.keys(fa.smartMessages.groups).sort());
    for (const [g, e] of Object.entries(GROUP_ENTRIES)) {
      expect(fa.messageAtlas.when, g).toHaveProperty(g);
      expect(SURFACES, g).toContain(e.surface);
    }
    for (const s of SURFACES) expect(fa.messageAtlas.surfaces, s).toHaveProperty(s);
  });

  it('says the «when» of a rule in words', () => {
    expect(whenOfRule(rule(), words)).toEqual({ all: ['روزهای تأخیر دست‌کم 5'], except: [] });
    expect(conditionText({ fact: 'has_log', op: 'eq', value: false }, words)).toBe('خیر: ثبت امروز');
    expect(conditionText({ fact: 'days_late', op: 'between', value: [3, 7] }, words)).toBe('روزهای تأخیر بین 3 و 7');
    const r = rule({
      conditions: [
        { op: 'and', items: [{ fact: 'days_late', op: 'gte', value: 5 }] },
        { op: 'or', items: [{ fact: 'has_log', op: 'eq', value: true }, { fact: 'preg.bp_high_level', op: 'in', value: ['high', 'severe'] }] },
      ],
      exceptions: [{ fact: 'has_log', op: 'eq', value: true }],
    });
    expect(whenOfRule(r, words)).toEqual({
      all: ['روزهای تأخیر دست‌کم 5', '(ثبت امروز یا سطح فشار یکی از high، severe)'],
      except: ['ثبت امروز'],
    });
  });

  it('finds who a rule reaches: explicit mode, then the mode its facts belong to, else everyone', () => {
    expect(modesOfRule(rule(), groupOf)).toEqual([]);
    expect(modesOfRule(rule({ conditions: [{ op: 'and', items: [{ fact: 'mode', op: 'in', value: ['ttc', 'cycle'] }] }] }), groupOf)).toEqual(['cycle', 'ttc']);
    expect(modesOfRule(rule({ conditions: [{ op: 'and', items: [{ fact: 'preg.bp_high_level', op: 'eq', value: 'high' }] }] }), groupOf)).toEqual(['pregnancy']);
    expect(modesOfRule(rule({ conditions: [{ op: 'and', items: [{ fact: 'meno.stage', op: 'eq', value: 'peri' }] }] }), groupOf)).toEqual(['menopause']);
  });

  it('maps placements to surfaces and filters / orders the entries', () => {
    expect(surfaceOfPlacement('alerts_inbox')).toBe('inbox');
    expect(surfaceOfPlacement('new_slot')).toBe('other');
    const entries = [
      entryOfRule(rule({ id: 1, priority: 100, placement: 'smart_tip' }), groupOf),
      entryOfRule(rule({ id: 2, priority: 900, placement: 'smart_tip' }), groupOf),
      ...groupEntries(),
    ];
    const tip = bySurface(entries).find((b) => b.surface === 'smartTip');
    expect(tip?.entries.slice(0, 2).map((e) => e.id)).toEqual(['rule:2', 'rule:1']);
    expect(tip?.entries.at(-1)?.source).toBe('group');
    const meno = filterEntries(entries, { mode: 'menopause', source: 'all', q: '' }, (e) => e.name);
    // everyone-rules + the menopause group + bmi (every mode)
    expect(meno.map((e) => e.id)).toEqual(['rule:1', 'rule:2', 'group:menopause_message', 'group:bmi_message']);
    expect(filterEntries(entries, { mode: 'all', source: 'group', q: 'companion' }, (e) => e.name).map((e) => e.id)).toEqual(['group:companion_tip']);
  });
});
