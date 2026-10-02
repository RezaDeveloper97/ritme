'use client';

import { useQuery } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';
import { careKeys, type Medication, medicationSchema, useMedication } from '@/entities/care-reminder';

/*
 * The medication a form edits. Own record: the entity's detail query. An
 * owner's record (B-N4-06, `?for=<ownerId>`): GET with `?for_user_id=` — a
 * companion with view or edit on her meds (B-N4-02), else 403
 * `companion_forbidden`. Its key sits under `careKeys.medication(id)` so every
 * medication mutation's invalidation reaches it.
 */

/** Key factory of an owner's record, nested under the entity's detail key. */
export const delegatedMedicationKeys = {
  detail: (id: number, forUserId: number) => [...careKeys.medication(id), 'for', forUserId] as const,
};

export async function fetchMedicationFor(id: number, forUserId: number): Promise<Medication> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>(`/care/medications/${id}`, {
    params: { for_user_id: forUserId },
  });
  return medicationSchema.parse(data.data);
}

/** null id = a new medication (no request). */
export function useMedicationFor(id: number | null, forUserId: number | null) {
  const own = useMedication(forUserId === null ? id : null);
  const delegated = useQuery({
    queryKey: delegatedMedicationKeys.detail(id ?? 0, forUserId ?? 0),
    queryFn: () => fetchMedicationFor(id as number, forUserId as number),
    enabled: isAuthenticated() && id !== null && forUserId !== null,
    retry: false,
  });
  return forUserId === null ? own : delegated;
}
