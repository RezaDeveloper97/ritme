/** Query keys of the baby logs (B-N5-07). Everything of a child hangs under `child(id)`. */
export const babyLogKeys = {
  all: ['baby-log'] as const,
  child: (id: number) => [...babyLogKeys.all, id] as const,
  feeds: (id: number, date: string | null) => [...babyLogKeys.child(id), 'feeds', date ?? 'today'] as const,
  sleeps: (id: number, date: string | null) => [...babyLogKeys.child(id), 'sleeps', date ?? 'today'] as const,
  diapers: (id: number, date: string | null) => [...babyLogKeys.child(id), 'diapers', date ?? 'today'] as const,
  summary: (id: number, days: number) => [...babyLogKeys.child(id), 'summary', days] as const,
};
