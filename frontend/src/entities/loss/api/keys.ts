/** Query keys of the loss path (CB-LOSS-02). */
export const lossKeys = {
  all: ['loss'] as const,
  state: () => [...lossKeys.all, 'state'] as const,
  note: () => [...lossKeys.all, 'note'] as const,
  catalog: (group: string, locale: string) => [...lossKeys.all, 'catalog', group, locale] as const,
};
