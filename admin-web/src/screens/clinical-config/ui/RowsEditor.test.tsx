import { NextIntlClientProvider } from 'next-intl';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';

import en from '../../../../messages/en.json';
import type { ConfigRow, ConfigTable } from '../api/clinical-config';
import { parseDraft, toDraft, type DraftRow } from '../lib/rows';
import { RowsEditor } from './RowsEditor';

const table: ConfigTable = {
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
const current: ConfigRow[] = [
  { ga_until_days: 62, max_diff_days: 5 },
  { ga_until_days: null, max_diff_days: 21 },
];

function render(draft: DraftRow[]) {
  return renderToStaticMarkup(
    <NextIntlClientProvider locale="en" messages={en} timeZone="Asia/Tehran">
      <RowsEditor table={table} draft={draft} parsed={parseDraft(table, draft)} current={current} onChange={() => undefined} />
    </NextIntlClientProvider>,
  );
}

describe('RowsEditor', () => {
  it('renders one labelled input per cell and the diff against the version in force', () => {
    const html = render(toDraft(table, current));
    expect(html.match(/<input[^>]*>/g)).toHaveLength(4);
    expect(html).toContain('aria-label="Row 2, ga_until_days"');
    expect(html).toContain('Same as the version in force.');
    expect(html).toContain('Version in force (2 rows)');
  });

  it('shows per-cell and table messages', () => {
    const html = render([
      { ga_until_days: '62', max_diff_days: '50' },
      { ga_until_days: '10', max_diff_days: '' },
    ]);
    expect(html).toContain('Out of the allowed range');
    expect(html).toContain('Required');
    expect(parseDraft(table, [{ ga_until_days: '90', max_diff_days: '5' }, { ga_until_days: '10', max_diff_days: '7' }]).table).toBe(
      'sort',
    );
    const changed = render([
      { ga_until_days: '62', max_diff_days: '6' },
      { ga_until_days: '', max_diff_days: '21' },
      { ga_until_days: '', max_diff_days: '' },
    ]);
    expect(changed).toContain('Required');
    const ok = render([
      { ga_until_days: '62', max_diff_days: '6' },
      { ga_until_days: '97', max_diff_days: '7' },
      { ga_until_days: '', max_diff_days: '21' },
    ]);
    expect(ok).toContain('Against the version in force: 2 changed · 1 added · 0 removed');
  });
});
