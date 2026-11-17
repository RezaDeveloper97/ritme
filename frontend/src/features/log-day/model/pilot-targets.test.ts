import { describe, expect, it } from 'vitest';

import type { LogCategory, LogParam, LogTaxonomy } from '@/entities/health-log';
import { isTargetId } from '@/shared/pilot';

import { logPilotId, logTargetsOf } from './pilot-targets';

const param = (over: Partial<LogParam> & Pick<LogParam, 'code' | 'type'>): LogParam => ({
  label: over.code,
  modes: [],
  detail: false,
  alert: false,
  options: [],
  levels: [],
  score: null,
  range: null,
  scale: null,
  unit: null,
  maxLength: null,
  dynamic: false,
  source: null,
  ...over,
});

const option = (value: string, modes: string[] | null = null, legacyOnly = false) => ({ value, label: `«${value}»`, modes, legacyOnly });

const category = (code: string, params: LogParam[]): LogCategory => ({
  code,
  label: `دسته ${code}`,
  group: { value: 'g', label: 'g' },
  modes: [],
  conditions: {},
  params,
});

const TAXONOMY: LogTaxonomy = {
  mode: 'cycle',
  categories: [
    category('bleeding', [
      param({ code: 'flow', type: 'single', options: [option('light'), option('heavy'), option('old', null, true), option('preg_only', ['pregnancy'])] }),
      param({ code: 'spotting', type: 'bool' }),
      param({ code: 'color', type: 'single', detail: true, options: [option('red')] }),
    ]),
    category('pain', [
      param({ code: 'location', type: 'items', options: [option('head')], levels: [{ value: 'no', label: 'نه' }, { value: 'mild', label: 'کم' }, { value: 'severe', label: 'شدید' }] }),
      param({ code: 'relief', type: 'multi', options: [option('rest')] }),
    ]),
    category('sleep', [
      param({ code: 'hours', type: 'number', range: { min: 0, max: 24 }, unit: { value: 'h', label: 'ساعت' } }),
      param({ code: 'quality', type: 'single', detail: true, options: [option('good')] }),
    ]),
    category('note', [param({ code: 'text', type: 'text', maxLength: 2000 })]),
    category('custom', [param({ code: 'items', type: 'multi' }), param({ code: 'kicks', type: 'link' }), param({ code: 'free', type: 'text_items' })]),
  ],
};

describe('logPilotId', () => {
  it('builds valid target ids', () => {
    for (const id of [logPilotId.save, logPilotId.category('bleeding'), logPilotId.more('pain'), logPilotId.param('bp', 'bp_systolic'), logPilotId.level('pain', 'location'), logPilotId.date('2026-11-15')]) {
      expect(isTargetId(id)).toBe(true);
    }
  });
});

describe('logTargetsOf', () => {
  const targets = logTargetsOf(TAXONOMY);
  const byId = new Map(targets.map((t) => [t.id, t]));

  it('lists each category as a section', () => {
    expect(byId.get('log.cat:bleeding')).toEqual({ id: 'log.cat:bleeding', kind: 'section', label: 'دسته bleeding' });
  });

  it('offers only the options this mode can pick, by code', () => {
    expect(byId.get('log.param:bleeding.flow')).toEqual({
      id: 'log.param:bleeding.flow',
      kind: 'single',
      label: 'flow',
      group: 'دسته bleeding',
      options: [{ value: 'light', label: '«light»' }, { value: 'heavy', label: '«heavy»' }],
    });
  });

  it('describes numbers, switches and notes', () => {
    expect(byId.get('log.param:sleep.hours')).toMatchObject({ kind: 'number', min: 0, max: 24, unit: 'ساعت' });
    expect(byId.get('log.param:bleeding.spotting')).toMatchObject({ kind: 'bool' });
    expect(byId.get('log.param:note.text')).toMatchObject({ kind: 'text', maxLength: 2000 });
  });

  it('adds a severity target for a graded items param, without «no»', () => {
    expect(byId.get('log.param:pain.location')).toMatchObject({ kind: 'multi', options: [{ value: 'head', label: '«head»' }] });
    expect(byId.get('log.level:pain.location')?.options?.map((o) => o.value)).toEqual(['mild', 'severe']);
  });

  it('leaves out what the accordions do not render', () => {
    // Detail params of a panel category and pain relief live in the detail panels.
    expect(byId.has('log.param:bleeding.color')).toBe(false);
    expect(byId.has('log.param:pain.relief')).toBe(false);
    // …but a detail param of an ordinary category sits behind «جزئیات بیشتر» and is reachable.
    expect(byId.has('log.param:sleep.quality')).toBe(true);
    // Nothing to pick, a link tile, free-text items.
    expect(byId.has('log.param:custom.items')).toBe(false);
    expect(byId.has('log.param:custom.kicks')).toBe(false);
    expect(byId.has('log.param:custom.free')).toBe(false);
  });
});
