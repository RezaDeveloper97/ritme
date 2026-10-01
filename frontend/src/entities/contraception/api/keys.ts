/**
 * Query-key factory for `/contraception` (CLAUDE.md §8). Every write answers
 * the whole screen, so mutations write {@link contraceptionKeys.overview}
 * directly instead of refetching.
 */
export const contraceptionKeys = {
  all: ['contraception'] as const,
  overview: () => [...contraceptionKeys.all, 'overview'] as const,
};
