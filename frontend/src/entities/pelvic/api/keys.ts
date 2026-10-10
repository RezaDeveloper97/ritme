/** Query keys of the pelvic floor program (CB-PELV-02). */
export const pelvicKeys = {
  all: ['pelvic'] as const,
  overview: () => [...pelvicKeys.all, 'overview'] as const,
  alerts: (locale: string) => [...pelvicKeys.all, 'alerts', locale] as const,
};
