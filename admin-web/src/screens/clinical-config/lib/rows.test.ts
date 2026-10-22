import { describe, expect, it } from 'vitest';

import en from '../../../../messages/en.json';
import fa from '../../../../messages/fa.json';
import type { ConfigRow, ConfigTable } from '../api/clinical-config';
import { cellId, diffRows, emptyRow, formatRow, moveRow, parseDraft, removeRow, toDraft } from './rows';

// pregnancy.redating and screening.uspstf as GET /clinical-config returns their schema.
const redating: ConfigTable = {
  columns: [
    { name: 'ga_until_days', type: 'number', nullable: true, min: 0, max: 300, integer: true },
    { name: 'max_diff_days', type: 'number', nullable: false, min: 1, max: 42, integer: true },
  ],
  min_rows: 1,
  max_rows: 12,
  sort_by: { column: 'ga_until_days', strict: true },
  range: null,
  unique: [],
  row_order: [],
};
const v1: ConfigRow[] = [
  { ga_until_days: 62, max_diff_days: 5 },
  { ga_until_days: 97, max_diff_days: 7 },
  { ga_until_days: null, max_diff_days: 21 },
];

const screening: ConfigTable = {
  ...redating,
  columns: [
    { name: 'type_key', type: 'string', nullable: false, max_length: 64 },
    { name: 'last_test_kind', type: 'enum', nullable: true, enum: ['cytology', 'hpv', 'cotest'] },
  ],
  sort_by: null,
};

describe('rows editor model', () => {
  it('round-trips the version in force through the draft', () => {
    const draft = toDraft(redating, v1);
    expect(draft).toEqual([
      { ga_until_days: '62', max_diff_days: '5' },
      { ga_until_days: '97', max_diff_days: '7' },
      { ga_until_days: '', max_diff_days: '21' },
    ]);
    const parsed = parseDraft(redating, draft);
    expect(parsed).toEqual({ rows: v1, cells: {}, table: null });
    expect(diffRows(redating, v1, parsed.rows!)).toEqual({
      rows: [
        { status: 'same', changed: [] },
        { status: 'same', changed: [] },
        { status: 'same', changed: [] },
      ],
      removed: 0,
    });
  });

  it('adds, removes and reorders rows; the diff follows', () => {
    let draft = toDraft(redating, v1);
    draft = [...draft, emptyRow(redating)];
    expect(draft[3]).toEqual({ ga_until_days: '', max_diff_days: '' });
    draft = removeRow(draft, 3);
    draft = moveRow(draft, 0, 1);
    expect(draft.map((r) => r.max_diff_days)).toEqual(['7', '5', '21']);
    expect(parseDraft(redating, draft).table).toBe('sort');
    draft = moveRow(draft, 1, 0);
    draft[0] = { ...draft[0]!, max_diff_days: '6' };
    draft = removeRow(draft, 1);
    const parsed = parseDraft(redating, draft);
    expect(diffRows(redating, v1, parsed.rows!)).toEqual({
      rows: [
        { status: 'changed', changed: ['max_diff_days'] },
        { status: 'changed', changed: ['ga_until_days', 'max_diff_days'] },
      ],
      removed: 1,
    });
    expect(moveRow(draft, 0, -1)).toBe(draft);
  });

  it('validates each column', () => {
    const parsed = parseDraft(redating, [
      { ga_until_days: '62.5', max_diff_days: '' },
      { ga_until_days: 'x', max_diff_days: '43' },
    ]);
    expect(parsed.rows).toBeNull();
    expect(parsed.cells).toEqual({
      [cellId(0, 'ga_until_days')]: 'integer',
      [cellId(0, 'max_diff_days')]: 'required',
      [cellId(1, 'ga_until_days')]: 'number',
      [cellId(1, 'max_diff_days')]: 'range',
    });
    expect(parseDraft(redating, []).table).toBe('count');
    expect(parseDraft(redating, [{ ga_until_days: '', max_diff_days: '5' }, { ga_until_days: '90', max_diff_days: '7' }]).table).toBe(
      'sort',
    );

    const s = parseDraft(screening, [
      { type_key: 'pap_smear', last_test_kind: 'pap' },
      { type_key: 'x'.repeat(65), last_test_kind: '' },
      { type_key: 'dentist', last_test_kind: 'hpv' },
    ]);
    expect(s.cells).toEqual({ [cellId(0, 'last_test_kind')]: 'enum', [cellId(1, 'type_key')]: 'length' });
    expect(parseDraft(screening, [{ type_key: 'dentist', last_test_kind: '' }]).rows).toEqual([
      { type_key: 'dentist', last_test_kind: null },
    ]);
  });

  it('has a message for every validation code in every message file', () => {
    const codes = ['required', 'number', 'integer', 'range', 'enum', 'length', 'count', 'sort'];
    for (const messages of [en, fa]) {
      const rows = messages.clinicalConfig.rows as { errors: Record<string, string> };
      for (const c of codes) expect(rows.errors[c], c).toBeTruthy();
    }
    expect(Object.keys(fa.clinicalConfig.rows).sort()).toEqual(Object.keys(en.clinicalConfig.rows).sort());
  });

  it('prints a row with open cells', () => {
    expect(formatRow(redating, v1[2]!, String, 'open')).toBe('open · 21');
  });
});
