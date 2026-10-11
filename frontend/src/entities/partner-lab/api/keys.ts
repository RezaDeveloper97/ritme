/** Query keys of the home lab booking requests (CLAUDE.md §8). */
export const partnerLabKeys = {
  all: ['partner-lab'] as const,
  services: () => [...partnerLabKeys.all, 'services'] as const,
  requests: () => [...partnerLabKeys.all, 'requests'] as const,
  request: (id: number) => [...partnerLabKeys.all, 'request', id] as const,
};
