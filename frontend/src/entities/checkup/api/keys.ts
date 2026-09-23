import type { CheckupListFilter, CheckupRecordFilter } from '../model/types';

export interface CheckupRecordFilters {
  filter?: CheckupRecordFilter;
  /** Checkup type id («همه» under سابقه → history of one type). */
  type?: number | null;
}

/**
 * Query-key factory for `/checkups/*` (CLAUDE.md §8). Mutations in
 * `features/record-checkup` and `features/manage-custom-checkup` invalidate
 * through these — never hand-written arrays.
 *
 * The `*All()` keys are prefixes: invalidating one refreshes every variant
 * (every filter, id or page) below it.
 */
export const checkupKeys = {
  all: ['checkups'] as const,
  listAll: () => [...checkupKeys.all, 'list'] as const,
  list: (filter: CheckupListFilter = 'all') => [...checkupKeys.listAll(), filter] as const,
  home: () => [...checkupKeys.all, 'home'] as const,
  detailAll: () => [...checkupKeys.all, 'detail'] as const,
  detail: (id: number) => [...checkupKeys.detailAll(), id] as const,
  recordsAll: () => [...checkupKeys.all, 'records'] as const,
  records: (filters: CheckupRecordFilters = {}) =>
    [...checkupKeys.recordsAll(), { filter: filters.filter ?? 'all', type: filters.type ?? null }] as const,
  previewNextAll: () => [...checkupKeys.all, 'preview-next'] as const,
  previewNext: (typeId: number, doneOn: string) =>
    [...checkupKeys.previewNextAll(), typeId, doneOn] as const,
  /** On-device report files (not server state, but cached the same way). */
  attachmentsAll: () => [...checkupKeys.all, 'attachments'] as const,
  attachment: (recordId: number) => [...checkupKeys.attachmentsAll(), recordId] as const,
};
