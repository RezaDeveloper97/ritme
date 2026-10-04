/** Query keys of the children reads (B-N5-05). B-N5-06 hangs growth/vaccines/milestones/learn under `detail(id)`. */
export const childKeys = {
  all: ['children'] as const,
  list: () => [...childKeys.all, 'list'] as const,
  detail: (id: number) => [...childKeys.all, 'detail', id] as const,
};
