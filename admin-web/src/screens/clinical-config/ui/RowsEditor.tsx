'use client';

import { useLocale, useTranslations } from 'next-intl';

import { cn, formatNumber } from '@/shared/lib';
import { Button, Icon, Select, TextInput } from '@/shared/ui';

import type { ConfigRow, ConfigTable } from '../api/clinical-config';
import {
  cellId,
  diffRows,
  emptyRow,
  formatRow,
  moveRow,
  removeRow,
  type DraftRow,
  type ParsedTable,
} from '../lib/rows';

/**
 * Table editor of a rows key (T-M10-16): add / remove / reorder rows, per-column validation under each cell, and the
 * row-by-row diff against the version in force. Empty = null (an open bound or "any") where the column allows it.
 */
export function RowsEditor({
  table,
  draft,
  parsed,
  current,
  onChange,
}: {
  table: ConfigTable;
  draft: DraftRow[];
  parsed: ParsedTable;
  current: ConfigRow[];
  onChange: (next: DraftRow[]) => void;
}) {
  const t = useTranslations('clinicalConfig');
  const locale = useLocale();
  const num = (n: number) => formatNumber(n, locale);
  const diff = parsed.rows ? diffRows(table, current, parsed.rows) : null;
  const set = (i: number, col: string, v: string) =>
    onChange(draft.map((r, j) => (j === i ? { ...r, [col]: v } : r)));

  const hint = (c: ConfigTable['columns'][number]) => {
    const parts: string[] = [];
    if (c.type === 'number' && c.min !== undefined && c.max !== undefined) {
      parts.push(t('rows.bounds', { min: num(c.min), max: num(c.max) }));
    }
    if (c.integer) parts.push(t('rows.integer'));
    if (c.nullable) parts.push(t('rows.nullable'));
    return parts.join(' · ');
  };

  return (
    <div className="flex flex-col gap-3 sm:col-span-2">
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <caption className="sr-only">{t('rows.caption')}</caption>
          <thead>
            <tr>
              <th scope="col" className="text-ink-3 p-1 text-start text-xs font-normal">
                #
              </th>
              {table.columns.map((c) => (
                <th key={c.name} scope="col" className="p-1 text-start align-bottom">
                  <span className="block font-mono text-xs" dir="ltr">
                    {c.name}
                  </span>
                  <span className="text-ink-3 block text-xs font-normal">{hint(c)}</span>
                </th>
              ))}
              <th scope="col" className="p-1">
                <span className="sr-only">{t('actions')}</span>
              </th>
            </tr>
          </thead>
          <tbody>
            {draft.map((row, i) => {
              const status = diff?.rows[i]?.status;
              return (
                <tr
                  key={i}
                  className={cn(status === 'added' && 'bg-green-soft', status === 'changed' && 'bg-amber-soft')}
                >
                  <td className="text-ink-3 p-1 text-xs">{num(i + 1)}</td>
                  {table.columns.map((c) => {
                    const err = parsed.cells[cellId(i, c.name)];
                    const label = t('rows.cellLabel', { row: num(i + 1), column: c.name });
                    const changed = diff?.rows[i]?.changed.includes(c.name) && status === 'changed';
                    return (
                      <td key={c.name} className={cn('p-1 align-top', changed && 'font-semibold')}>
                        {c.type === 'enum' ? (
                          <Select
                            aria-label={label}
                            value={row[c.name] ?? ''}
                            onChange={(e) => set(i, c.name, e.target.value)}
                            error={err ? t(`rows.errors.${err}`) : undefined}
                            options={[
                              ...(c.nullable ? [{ value: '', label: t('rows.any') }] : []),
                              ...(c.enum ?? []).map((v) => ({ value: v, label: v })),
                            ]}
                          />
                        ) : (
                          <TextInput
                            aria-label={label}
                            type={c.type === 'number' ? 'number' : 'text'}
                            inputMode={c.type === 'number' ? 'decimal' : undefined}
                            dir="ltr"
                            placeholder={c.nullable ? t('rows.open') : undefined}
                            value={row[c.name] ?? ''}
                            onChange={(e) => set(i, c.name, e.target.value)}
                            error={err ? t(`rows.errors.${err}`) : undefined}
                          />
                        )}
                      </td>
                    );
                  })}
                  <td className="p-1 align-top whitespace-nowrap">
                    <Button
                      size="sm"
                      variant="ghost"
                      aria-label={t('rows.up', { row: num(i + 1) })}
                      disabled={i === 0}
                      onClick={() => onChange(moveRow(draft, i, i - 1))}
                    >
                      ↑
                    </Button>
                    <Button
                      size="sm"
                      variant="ghost"
                      aria-label={t('rows.down', { row: num(i + 1) })}
                      disabled={i === draft.length - 1}
                      onClick={() => onChange(moveRow(draft, i, i + 1))}
                    >
                      ↓
                    </Button>
                    <Button
                      size="sm"
                      variant="ghost"
                      aria-label={t('rows.remove', { row: num(i + 1) })}
                      onClick={() => onChange(removeRow(draft, i))}
                    >
                      <Icon name="close" size={14} />
                    </Button>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
      <div className="flex flex-wrap items-center gap-3">
        <Button
          size="sm"
          variant="ghost"
          disabled={table.max_rows > 0 && draft.length >= table.max_rows}
          onClick={() => onChange([...draft, emptyRow(table)])}
        >
          <Icon name="plus" size={14} />
          {t('rows.add')}
        </Button>
        <span className="text-ink-3 text-xs">
          {t('rows.limits', { min: num(table.min_rows), max: num(table.max_rows) })}
        </span>
      </div>
      {parsed.table ? <p className="text-danger text-sm">{t(`rows.errors.${parsed.table}`)}</p> : null}
      {diff ? <DiffSummary diff={diff} /> : null}
      {current.length > 0 ? (
        <details className="text-xs">
          <summary className="text-ink-3 cursor-pointer">{t('rows.inForce', { count: num(current.length) })}</summary>
          <ol className="text-ink-2 mt-1 flex flex-col gap-0.5" dir="ltr">
            {current.map((r, i) => (
              <li key={i}>{formatRow(table, r, num, t('rows.open'))}</li>
            ))}
          </ol>
        </details>
      ) : null}
    </div>
  );
}

function DiffSummary({ diff }: { diff: ReturnType<typeof diffRows> }) {
  const t = useTranslations('clinicalConfig');
  const locale = useLocale();
  const count = (s: 'added' | 'changed') => diff.rows.filter((r) => r.status === s).length;
  const changed = count('changed');
  const added = count('added');
  if (changed + added + diff.removed === 0) return <p className="text-ink-3 text-xs">{t('rows.noChange')}</p>;
  return (
    <p className="text-ink-2 text-xs">
      {t('rows.diff', {
        changed: formatNumber(changed, locale),
        added: formatNumber(added, locale),
        removed: formatNumber(diff.removed, locale),
      })}
    </p>
  );
}
