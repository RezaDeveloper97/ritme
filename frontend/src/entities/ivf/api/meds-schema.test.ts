import { describe, expect, it } from 'vitest';

import {
  ivfInjectionSitesSchema,
  ivfMedPresetsSchema,
  ivfMedsViewSchema,
  toIvfMedBody,
} from './meds-schema';

const CODES = [
  'abdomen_upper_right',
  'abdomen_upper_left',
  'thigh_right',
  'thigh_left',
  'abdomen_lower_right',
  'abdomen_lower_left',
  'arm_right',
  'arm_left',
];

const dose = (over: Record<string, unknown> = {}) => ({
  med_id: 1,
  reminder_id: 101805,
  name: 'FSH',
  dose: '150',
  unit: 'iu',
  route: 'subcutaneous',
  role: 'stimulation',
  is_trigger: false,
  date: '2026-09-23',
  slot: '20:00',
  taken: false,
  site: null,
  ...over,
});

// Shape of backend-go/contract/golden/ivf/meds_flow (GET /ivf/meds after a log).
const VIEW = {
  cycle: null,
  trigger: { med_id: 3, name: 'hCG', dose: '10000', unit: 'iu', trigger_at: '2026-09-25 22:30:00', taken: false },
  today: { date: '2026-09-23', doses: [dose({ med_id: 2, name: 'Antagonist', dose: null, unit: null, slot: '08:00', taken: true, site: 'abdomen_upper_right' }), dose()] },
  tomorrow: { date: '2026-09-24', doses: [dose({ date: '2026-09-24' })] },
  sites: { codes: CODES, last: { site: 'abdomen_upper_right', date: '2026-09-23', slot: '08:00' }, suggested: 'abdomen_upper_left' },
  meds: [
    {
      id: 1,
      reminder_id: 101805,
      name: 'FSH',
      role: 'stimulation',
      route: 'subcutaneous',
      is_trigger: false,
      trigger_at: null,
      dose: '150',
      unit: 'iu',
      times: ['20:00'],
      starts_on: '2026-09-17',
      ends_on: null,
      is_active: true,
      notes: null,
      inventory: {
        stock_units: 3,
        stock_unit: 'pen',
        doses_per_unit: 1,
        counted_at: '2026-09-23 10:00:00',
        doses_left: 3,
        units_left: 3,
        days_left: 3,
        runs_out_on: '2026-09-26',
        low: true,
      },
    },
    { id: 3, name: 'hCG', role: 'trigger', route: 'subcutaneous', is_trigger: true, trigger_at: '2026-09-25 22:30:00', dose: 10000, unit: 'iu', times: [], starts_on: null, ends_on: null, is_active: true, notes: null, inventory: null },
  ],
};

describe('ivfMedsViewSchema', () => {
  it('maps trigger, days, sites and meds with inventory', () => {
    const view = ivfMedsViewSchema.parse(VIEW);
    expect(view.trigger).toEqual({ medId: 3, name: 'hCG', dose: '10000', unit: 'iu', triggerAt: '2026-09-25 22:30:00', taken: false });
    expect(view.today.doses.map((d) => d.site)).toEqual(['abdomen_upper_right', null]);
    expect(view.tomorrow.date).toBe('2026-09-24');
    expect(view.sites).toEqual({ codes: CODES, last: VIEW.sites.last, suggested: 'abdomen_upper_left' });
    expect(view.meds[0]?.inventory).toMatchObject({ unitsLeft: 3, stockUnit: 'pen', low: true, runsOutOn: '2026-09-26' });
    expect(view.meds[1]).toMatchObject({ isTrigger: true, dose: '10000', inventory: null, times: [] });
  });

  it('drops a malformed medicine, keeps the rest; bad sites fall back to empty', () => {
    const broken = { ...VIEW, meds: [...VIEW.meds, { id: 'x' }], sites: 'nope' };
    const view = ivfMedsViewSchema.parse(broken);
    expect(view.meds).toHaveLength(2);
    expect(view.sites).toEqual({ codes: [], last: null, suggested: null });
  });

  it('reads the empty schedule (no cycle)', () => {
    const view = ivfMedsViewSchema.parse({
      cycle: null,
      trigger: null,
      today: { date: '2026-09-23', doses: [] },
      tomorrow: { date: '2026-09-24', doses: [] },
      sites: { codes: CODES, last: null, suggested: 'abdomen_upper_right' },
      meds: [],
    });
    expect(view).toMatchObject({ cycle: null, trigger: null, meds: [] });
    expect(view.sites.suggested).toBe('abdomen_upper_right');
  });
});

describe('toIvfMedBody', () => {
  it('sends snake_case and drops stock fields without a count', () => {
    expect(
      toIvfMedBody({
        name: 'FSH',
        role: 'stimulation',
        route: 'subcutaneous',
        dose: '150',
        unit: 'iu',
        times: ['20:00'],
        triggerAt: null,
        startsOn: '2026-09-17',
        endsOn: null,
        notes: null,
        stockUnits: null,
        stockUnit: 'pen',
        dosesPerUnit: 2,
      }),
    ).toEqual({
      name: 'FSH',
      role: 'stimulation',
      route: 'subcutaneous',
      dose: '150',
      unit: 'iu',
      times: ['20:00'],
      trigger_at: null,
      starts_on: '2026-09-17',
      ends_on: null,
      notes: null,
      stock_units: null,
      stock_unit: null,
      doses_per_unit: null,
    });
  });
});

describe('catalog groups', () => {
  it('reads site meta and preset meta', () => {
    expect(
      ivfInjectionSitesSchema.parse({
        items: [
          { code: 'thigh_left', title: 'ران · چپ', body: 'جلو و بیرون ران', meta: { region: 'thigh', side: 'left' } },
          { title: 'no code' },
        ],
      }),
    ).toEqual([{ code: 'thigh_left', title: 'ران · چپ', body: 'جلو و بیرون ران', region: 'thigh', side: 'left' }]);
    expect(
      ivfMedPresetsSchema.parse({
        items: [
          { code: 'fsh', title: 'FSH', meta: { role: 'stimulation', route: 'subcutaneous', unit: 'iu', times: ['20:00'], stock_unit: 'pen' } },
          { code: 'hcg_trigger', title: 'hCG', meta: { role: 'trigger', route: 'subcutaneous', unit: 'iu', stock_unit: 'prefilled_syringe' } },
          { code: 'bad', title: 'x', meta: { role: 'nope' } },
        ],
      }),
    ).toEqual([
      { code: 'fsh', title: 'FSH', role: 'stimulation', route: 'subcutaneous', unit: 'iu', times: ['20:00'], stockUnit: 'pen' },
      { code: 'hcg_trigger', title: 'hCG', role: 'trigger', route: 'subcutaneous', unit: 'iu', times: [], stockUnit: 'prefilled_syringe' },
    ]);
    expect(ivfMedPresetsSchema.parse(null)).toEqual([]);
  });
});
