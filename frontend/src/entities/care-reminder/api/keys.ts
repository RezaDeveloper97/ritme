import type { AppointmentScope } from '../model/types';

export interface MedicationFilters {
  /** Only medications whose switch is on. */
  active?: boolean;
}

/**
 * Query-key factory for `/care/*` (CLAUDE.md §8). Mutations in
 * `features/manage-medication`, `manage-appointment` and `log-intake`
 * invalidate through these — never hand-written arrays.
 *
 * The `*All()` keys are prefixes: invalidating one refreshes every variant
 * (every filter, scope, id or date) below it.
 */
export const careKeys = {
  all: ['care'] as const,
  medicationsAll: () => [...careKeys.all, 'medications'] as const,
  medications: (filters: MedicationFilters = {}) =>
    [...careKeys.medicationsAll(), 'list', { active: filters.active === true }] as const,
  medication: (id: number) => [...careKeys.medicationsAll(), 'detail', id] as const,
  appointmentsAll: () => [...careKeys.all, 'appointments'] as const,
  appointments: (scope: AppointmentScope = 'upcoming') =>
    [...careKeys.appointmentsAll(), 'list', scope] as const,
  appointment: (id: number) => [...careKeys.appointmentsAll(), 'detail', id] as const,
  todayAll: () => [...careKeys.all, 'today'] as const,
  /** `date` omitted = "today" as the server resolves it. */
  today: (date?: string) => [...careKeys.todayAll(), date ?? 'current'] as const,
  enums: () => [...careKeys.all, 'enums'] as const,
};
