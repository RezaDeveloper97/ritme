/** Query keys of the condition programs (CB-COND-02). */
export const conditionKeys = {
  all: ['condition'] as const,
  overview: () => [...conditionKeys.all, 'overview'] as const,
  nudges: () => [...conditionKeys.all, 'nudges'] as const,
  catalog: (group: string, locale: string) => [...conditionKeys.all, 'catalog', group, locale] as const,
};
