/** Query keys of the health record (CLAUDE.md §8). */
export const healthRecordKeys = {
  all: ['health-record'] as const,
  record: () => [...healthRecordKeys.all, 'record'] as const,
};
