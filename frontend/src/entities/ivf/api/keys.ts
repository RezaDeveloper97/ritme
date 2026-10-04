/** Query keys of the IVF reads (CB-IVF-02, CB-IVF-03). */
export const ivfKeys = {
  all: ['ivf'] as const,
  home: () => [...ivfKeys.all, 'home'] as const,
  stages: (locale: string) => [...ivfKeys.all, 'catalog', 'ivf_stages', locale] as const,
  /** CB-IVF-03: the injection schedule (`GET /ivf/meds`). */
  meds: () => [...ivfKeys.all, 'meds'] as const,
  /** CB-IVF-03: another `ivf_*` catalog group (sites, presets, guidance). */
  catalog: (group: string, locale: string) => [...ivfKeys.all, 'catalog', group, locale] as const,
};
