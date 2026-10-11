/** Query keys for `/insurance` (CLAUDE.md §8). Personal data: never log what these return. */
export const insuranceKeys = {
  all: ['insurance'] as const,
  policy: () => [...insuranceKeys.all, 'policy'] as const,
  insurers: () => [...insuranceKeys.all, 'insurers'] as const,
  centres: (type: string, q: string) => [...insuranceKeys.all, 'centres', type, q] as const,
  recentClaims: () => [...insuranceKeys.all, 'claims', 'recent'] as const,
  questionnaires: () => [...insuranceKeys.all, 'questionnaires'] as const,
  questionnaire: (id: number) => [...insuranceKeys.all, 'questionnaires', id] as const,
};
