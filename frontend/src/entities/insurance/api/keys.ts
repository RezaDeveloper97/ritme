/** Query keys for `/insurance` (CLAUDE.md §8). Personal data: never log what these return. */
export const insuranceKeys = {
  all: ['insurance'] as const,
  policy: () => [...insuranceKeys.all, 'policy'] as const,
  policies: () => [...insuranceKeys.all, 'policies'] as const,
  policyById: (id: number) => [...insuranceKeys.all, 'policies', id] as const,
  insurers: () => [...insuranceKeys.all, 'insurers'] as const,
  centres: (type: string, q: string) => [...insuranceKeys.all, 'centres', type, q] as const,
  claim: (id: number) => [...insuranceKeys.all, 'claims', 'detail', id] as const,
  recentClaims: (policyId?: number | null) => [...insuranceKeys.all, 'claims', 'recent', policyId ?? 'current'] as const,
  questionnaires: () => [...insuranceKeys.all, 'questionnaires'] as const,
  questionnaire: (id: number) => [...insuranceKeys.all, 'questionnaires', id] as const,
};
