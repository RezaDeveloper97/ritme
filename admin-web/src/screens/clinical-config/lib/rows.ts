import type { Cell, ConfigColumn, ConfigRow, ConfigTable } from '../api/clinical-config';

/**
 * Table editor model of a rows key (T-M10-16): every cell is edited as text ('' = null, an open bound or "any") and
 * parsed back per column. The server validates the whole table again (sort order, overlaps, gaps, unique rows).
 */
export type DraftRow = Record<string, string>;

/** Per-cell problems the editor shows under the cell; `count` is a table-level problem. */
export type CellError = 'required' | 'number' | 'integer' | 'range' | 'enum' | 'length';
export type TableError = 'count' | 'sort';

export interface ParsedTable {
  rows: ConfigRow[] | null;
  cells: Record<string, CellError>;
  table: TableError | null;
}

export const cellId = (row: number, col: string) => `${row}:${col}`;

export function toDraft(table: ConfigTable, rows: ConfigRow[]): DraftRow[] {
  return rows.map((r) =>
    Object.fromEntries(table.columns.map((c) => [c.name, r[c.name] === null || r[c.name] === undefined ? '' : String(r[c.name])])),
  );
}

export function emptyRow(table: ConfigTable): DraftRow {
  return Object.fromEntries(table.columns.map((c) => [c.name, '']));
}

export function moveRow(draft: DraftRow[], from: number, to: number): DraftRow[] {
  if (to < 0 || to >= draft.length || from === to) return draft;
  const next = [...draft];
  const [row] = next.splice(from, 1);
  next.splice(to, 0, row!);
  return next;
}

export function removeRow(draft: DraftRow[], at: number): DraftRow[] {
  return draft.filter((_, i) => i !== at);
}

function parseCell(col: ConfigColumn, raw: string): { value: Cell } | { error: CellError } {
  const s = raw.trim();
  if (s === '') return col.nullable ? { value: null } : { error: 'required' };
  if (col.type === 'enum') return (col.enum ?? []).includes(s) ? { value: s } : { error: 'enum' };
  if (col.type === 'string') return s.length > (col.max_length ?? 64) ? { error: 'length' } : { value: s };
  const n = Number(s);
  if (!Number.isFinite(n)) return { error: 'number' };
  if (col.integer && !Number.isInteger(n)) return { error: 'integer' };
  if ((col.min !== undefined && n < col.min) || (col.max !== undefined && n > col.max)) return { error: 'range' };
  return { value: n };
}

/** Parses the editor's draft; rows is null while any cell or the row count is invalid. */
export function parseDraft(table: ConfigTable, draft: DraftRow[]): ParsedTable {
  const cells: Record<string, CellError> = {};
  const rows: ConfigRow[] = draft.map((d, i) => {
    const row: ConfigRow = {};
    for (const col of table.columns) {
      const res = parseCell(col, d[col.name] ?? '');
      if ('error' in res) cells[cellId(i, col.name)] = res.error;
      else row[col.name] = res.value;
    }
    return row;
  });
  let tableError: TableError | null = null;
  if (draft.length < table.min_rows || (table.max_rows > 0 && draft.length > table.max_rows)) tableError = 'count';
  else if (Object.keys(cells).length === 0 && table.sort_by && !sorted(rows, table.sort_by.column, table.sort_by.strict)) {
    tableError = 'sort';
  }
  const ok = Object.keys(cells).length === 0 && tableError === null;
  return { rows: ok ? rows : null, cells, table: tableError };
}

/** Ascending in col; a null (open) value only on the last row. */
function sorted(rows: ConfigRow[], col: string, strict: boolean): boolean {
  for (let i = 0; i < rows.length; i++) {
    const cur = rows[i]![col];
    if (cur === null || cur === undefined) {
      if (i !== rows.length - 1) return false;
      continue;
    }
    if (i === 0) continue;
    const prev = rows[i - 1]![col] as number;
    if (typeof cur === 'number' && (cur < prev || (strict && cur === prev))) return false;
  }
  return true;
}

export type RowStatus = 'same' | 'changed' | 'added';

export interface RowsDiff {
  rows: { status: RowStatus; changed: string[] }[];
  /** Rows of the version in force past the end of the new table. */
  removed: number;
}

/** Row-by-row diff of a proposed table against the version in force (rows are ordered, so compared by position). */
export function diffRows(table: ConfigTable, current: ConfigRow[], next: ConfigRow[]): RowsDiff {
  const rows = next.map((r, i) => {
    const old = current[i];
    if (!old) return { status: 'added' as const, changed: table.columns.map((c) => c.name) };
    const changed = table.columns.filter((c) => (old[c.name] ?? null) !== (r[c.name] ?? null)).map((c) => c.name);
    return { status: changed.length ? ('changed' as const) : ('same' as const), changed };
  });
  return { rows, removed: Math.max(0, current.length - next.length) };
}

/** «62 · 5» for one row; null cells print as `open`. */
export function formatRow(table: ConfigTable, row: ConfigRow, num: (n: number) => string, open: string): string {
  return table.columns
    .map((c) => {
      const v = row[c.name];
      if (v === null || v === undefined) return open;
      return typeof v === 'number' ? num(v) : v;
    })
    .join(' · ');
}
