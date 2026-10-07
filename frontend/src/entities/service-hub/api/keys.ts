/** Query keys of the services hub (CLAUDE.md §8). */
export const serviceHubKeys = {
  all: ['service-hub'] as const,
  hub: () => [...serviceHubKeys.all, 'hub'] as const,
};
