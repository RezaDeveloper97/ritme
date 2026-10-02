import { describe, expect, it } from 'vitest';

import type { CheckupItem } from '@/entities/checkup';

import {
  groupForMenopause,
  menoCadence,
  menoCheckupGroupsSchema,
  menoCheckupsIntroSchema,
  menoChip,
  menoWhen,
} from './meno';

const item = (id: number, key: string | null, over: Partial<CheckupItem> = {}): CheckupItem => ({
  id,
  key,
  title: `t${id}`,
  subtitle: null,
  category: 'annual',
  section: 'annual',
  status: 'due',
  icon: null,
  tone: 'neutral',
  intervalLabel: null,
  timingLabel: null,
  lastDoneOn: null,
  nextDueOn: null,
  nextDueLabel: null,
  isCustom: false,
  ...over,
});

describe('menoCheckupGroupsSchema', () => {
  it('reads code, title and the checkup keys; drops malformed items', () => {
    const groups = menoCheckupGroupsSchema.parse({
      items: [
        { code: 'bone', title: 'استخوان', meta: { checkups: ['meno_bone_density'] } },
        { code: 'empty', title: '', meta: null },
        { title: 'no code' },
      ],
    });
    expect(groups).toEqual([
      { code: 'bone', title: 'استخوان', keys: ['meno_bone_density'] },
      { code: 'empty', title: null, keys: [] },
    ]);
  });

  it('survives a missing payload', () => {
    expect(menoCheckupGroupsSchema.parse(null)).toEqual([]);
  });
});

describe('menoCheckupsIntroSchema', () => {
  it('picks the checkups_intro body', () => {
    expect(
      menoCheckupsIntroSchema.parse({
        items: [
          { code: 'stage_meno', body: 'x' },
          { code: 'checkups_intro', body: ' intro ' },
        ],
      }),
    ).toBe('intro');
  });

  it('is null without it', () => {
    expect(menoCheckupsIntroSchema.parse({ items: [{ code: 'checkups_intro', body: '' }] })).toBeNull();
  });
});

describe('groupForMenopause', () => {
  it('orders rows by group keys, drops empty groups, and collects the rest under `more`', () => {
    const items = [item(1, 'breast_self_exam'), item(2, 'mammography'), item(3, 'meno_lipids'), item(4, null)];
    const sections = groupForMenopause(items, [
      { code: 'heart', title: 'H', keys: ['meno_lipids', 'meno_blood_pressure'] },
      { code: 'cancer', title: 'C', keys: ['mammography'] },
      { code: 'bone', title: 'B', keys: ['meno_bone_density'] },
      { code: 'dup', title: 'D', keys: ['mammography'] },
    ]);
    expect(sections.map((s) => [s.code, s.items.map((i) => i.id)])).toEqual([
      ['heart', [3]],
      ['cancer', [2]],
      ['more', [1, 4]],
    ]);
  });

  it('puts everything in `more` when the groups failed', () => {
    expect(groupForMenopause([item(1, 'a')], []).map((s) => s.code)).toEqual(['more']);
  });
});

describe('menoChip', () => {
  it('maps statuses to the board chips', () => {
    expect(menoChip({ status: 'overdue', category: 'annual' })).toBe('overdue');
    expect(menoChip({ status: 'soon', category: 'annual' })).toBe('soon');
    expect(menoChip({ status: 'up_to_date', category: 'monthly' })).toBe('up_to_date');
    expect(menoChip({ status: 'due', category: 'monthly' })).toBe('log');
    expect(menoChip({ status: 'due', category: 'age_based' })).toBe('ask');
    expect(menoChip({ status: 'not_yet', category: 'age_based' })).toBe('ask');
    expect(menoChip({ status: 'disabled', category: 'annual' })).toBe('disabled');
  });
});

describe('menoCadence', () => {
  it('uses the subtitle of menopause types and the interval of shared ones', () => {
    expect(menoCadence({ key: 'meno_lipids', intervalLabel: 'هر ۱ تا ۵ سال', subtitle: 'معمولاً هر ۱ تا ۵ سال' })).toBe(
      'معمولاً هر ۱ تا ۵ سال',
    );
    expect(menoCadence({ key: 'mammography', intervalLabel: 'هر ۱ تا ۲ سال', subtitle: 'غربالگری' })).toBe(
      'هر ۱ تا ۲ سال',
    );
    expect(menoCadence({ key: null, intervalLabel: null, subtitle: 's' })).toBe('s');
  });
});

describe('menoWhen', () => {
  it('shows the due date when soon, else the last time, else never', () => {
    expect(menoWhen({ status: 'soon', lastDoneOn: '2025-01-01', nextDueOn: '2026-11-01' })).toEqual({
      kind: 'next',
      date: '2026-11-01',
    });
    expect(menoWhen({ status: 'overdue', lastDoneOn: '2024-03-01', nextDueOn: null })).toEqual({
      kind: 'last',
      date: '2024-03-01',
    });
    expect(menoWhen({ status: 'due', lastDoneOn: null, nextDueOn: null })).toEqual({ kind: 'never' });
  });
});
