/** Query keys of `/api/v1/labs/*` (B-N6-07). Invalidate `labKeys.all` after any write. */
export const labKeys = {
  all: ['labs'] as const,
  list: () => [...labKeys.all, 'list'] as const,
  detail: (id: number) => [...labKeys.all, 'detail', id] as const,
  status: (id: number) => [...labKeys.all, 'status', id] as const,
  marker: (id: number, markerId: number) => [...labKeys.all, 'marker', id, markerId] as const,
  trends: () => [...labKeys.all, 'trends'] as const,
  catalog: () => [...labKeys.all, 'catalog'] as const,
  consent: () => ['consents', 'ai_lab_analysis'] as const,
};
