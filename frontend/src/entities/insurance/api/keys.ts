/** Query keys for `/insurance` (CLAUDE.md §8). Personal data: never log what these return. */
export const insuranceKeys = {
  all: ['insurance'] as const,
  policy: () => [...insuranceKeys.all, 'policy'] as const,
  insurers: () => [...insuranceKeys.all, 'insurers'] as const,
  recentClaims: () => [...insuranceKeys.all, 'claims', 'recent'] as const,
};
