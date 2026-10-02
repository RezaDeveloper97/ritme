'use client';

import { useQuery } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';
import { careKeys, type Appointment, appointmentSchema, useAppointment } from '@/entities/care-reminder';

/*
 * The appointment a form edits. Own record: the entity's detail query. An
 * owner's record (B-N4-06, `?for=<ownerId>`): GET with `?for_user_id=` — a
 * companion with view or edit on her appointments (B-N4-02), else 403
 * `companion_forbidden`. Its key sits under `careKeys.appointment(id)` so every
 * appointment mutation's invalidation reaches it.
 */

/** Key factory of an owner's record, nested under the entity's detail key. */
export const delegatedAppointmentKeys = {
  detail: (id: number, forUserId: number) => [...careKeys.appointment(id), 'for', forUserId] as const,
};

export async function fetchAppointmentFor(id: number, forUserId: number): Promise<Appointment> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>(`/care/appointments/${id}`, {
    params: { for_user_id: forUserId },
  });
  return appointmentSchema.parse(data.data);
}

/** null id = a new appointment (no request). */
export function useAppointmentFor(id: number | null, forUserId: number | null) {
  const own = useAppointment(forUserId === null ? id : null);
  const delegated = useQuery({
    queryKey: delegatedAppointmentKeys.detail(id ?? 0, forUserId ?? 0),
    queryFn: () => fetchAppointmentFor(id as number, forUserId as number),
    enabled: isAuthenticated() && id !== null && forUserId !== null,
    retry: false,
  });
  return forUserId === null ? own : delegated;
}
