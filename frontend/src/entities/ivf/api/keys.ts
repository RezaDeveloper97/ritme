/** Query keys of the IVF reads (CB-IVF-02). */
export const ivfKeys = {
  all: ['ivf'] as const,
  home: () => [...ivfKeys.all, 'home'] as const,
  stages: (locale: string) => [...ivfKeys.all, 'catalog', 'ivf_stages', locale] as const,
};
