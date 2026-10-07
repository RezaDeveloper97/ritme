/** Query keys of the health record (CLAUDE.md §8). */
export const healthRecordKeys = {
  all: ['health-record'] as const,
  record: () => [...healthRecordKeys.all, 'record'] as const,
  /** The doctor report preview of one selection (B-N6-04). */
  report: (params: Record<string, string>) => [...healthRecordKeys.all, 'report', params] as const,
  /** The owner's share links (B-N6-04). */
  shareLinks: () => [...healthRecordKeys.all, 'share-links'] as const,
  /** The owner's emergency card (CB-REC-03). */
  emergencyCard: () => [...healthRecordKeys.all, 'emergency-card'] as const,
  /** The app-lock screen's minimal card (CB-PRIV-01). */
  emergencyCardLock: () => [...healthRecordKeys.all, 'emergency-card-lock'] as const,
};
