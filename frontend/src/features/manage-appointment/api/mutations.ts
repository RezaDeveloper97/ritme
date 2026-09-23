'use client';

import { type QueryClient, useMutation, useQueryClient } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { type Appointment, appointmentSchema, careKeys } from '@/entities/care-reminder';
import { reminderKeys } from '@/entities/reminder';

import {
  type AppointmentInput,
  type AppointmentPatch,
  setPrepItemDone,
  toAppointmentBody,
} from '../model/body';

/*
 * Create / update / delete / cancel an appointment and tick its prep list
 * (v13_AddAppointment, v13_AppointmentDetail). Every mutation invalidates
 * through `careKeys` — the lists, the detail and today's next appointment —
 * plus the legacy `/reminders` list, which shows the same rows.
 *
 * Privacy (§11): who/where/why of a doctor visit goes to the API only.
 */

function invalidate(queryClient: QueryClient, id?: number): void {
  void queryClient.invalidateQueries({ queryKey: careKeys.appointmentsAll() });
  void queryClient.invalidateQueries({ queryKey: careKeys.todayAll() });
  if (id !== undefined) void queryClient.invalidateQueries({ queryKey: careKeys.appointment(id) });
  void queryClient.invalidateQueries({ queryKey: reminderKeys.all });
}

/** POST /care/appointments → 201. */
export function useCreateAppointment() {
  const queryClient = useQueryClient();
  return useMutation<Appointment, unknown, AppointmentInput>({
    mutationFn: async (input) => {
      const { data } = await apiClient.post<ApiEnvelope<unknown>>(
        '/care/appointments',
        toAppointmentBody(input),
      );
      return appointmentSchema.parse(data.data);
    },
    onSuccess: () => invalidate(queryClient),
  });
}

export interface UpdateAppointmentVars {
  id: number;
  patch: AppointmentPatch;
}

/** PUT /care/appointments/{id} — partial (also the reminder switch: `{isActive}`). */
export function useUpdateAppointment() {
  const queryClient = useQueryClient();
  return useMutation<Appointment, unknown, UpdateAppointmentVars>({
    mutationFn: async ({ id, patch }) => {
      const { data } = await apiClient.put<ApiEnvelope<unknown>>(
        `/care/appointments/${id}`,
        toAppointmentBody(patch),
      );
      return appointmentSchema.parse(data.data);
    },
    onSuccess: (_data, { id }) => invalidate(queryClient, id),
  });
}

/** DELETE /care/appointments/{id}. */
export function useDeleteAppointment() {
  const queryClient = useQueryClient();
  return useMutation<void, unknown, number>({
    mutationFn: async (id) => {
      await apiClient.delete(`/care/appointments/${id}`);
    },
    onSuccess: (_data, id) => {
      queryClient.removeQueries({ queryKey: careKeys.appointment(id) });
      invalidate(queryClient);
    },
  });
}

/** POST /care/appointments/{id}/cancel — status=cancelled, reminder off, row kept. */
export function useCancelAppointment() {
  const queryClient = useQueryClient();
  return useMutation<void, unknown, number>({
    mutationFn: async (id) => {
      await apiClient.post(`/care/appointments/${id}/cancel`);
    },
    onSuccess: (_data, id) => invalidate(queryClient, id),
  });
}

export interface TogglePrepItemVars {
  appointment: Appointment;
  itemId: string;
  done: boolean;
}

/**
 * Ticks/unticks one prep item, optimistically on the detail cache.
 *
 * The contract's dedicated route is `PATCH /care/appointments/{id}/prep/{itemId}`,
 * but the shared client has no `patch` yet (shared/api is outside this slice),
 * so this sends the whole updated list through the partial PUT, which the
 * contract also accepts. Swap the call once `apiClient.patch` exists.
 */
export function useTogglePrepItem() {
  const queryClient = useQueryClient();
  return useMutation<void, unknown, TogglePrepItemVars, { previous?: Appointment }>({
    mutationFn: async ({ appointment, itemId, done }) => {
      await apiClient.put(`/care/appointments/${appointment.id}`, {
        prep: setPrepItemDone(appointment.prep, itemId, done),
      });
    },
    onMutate: async ({ appointment, itemId, done }) => {
      const key = careKeys.appointment(appointment.id);
      await queryClient.cancelQueries({ queryKey: key });
      const previous = queryClient.getQueryData<Appointment>(key);
      queryClient.setQueryData<Appointment>(key, (current) =>
        current ? { ...current, prep: setPrepItemDone(current.prep, itemId, done) } : current,
      );
      return { previous };
    },
    onError: (_error, { appointment }, context) => {
      if (context?.previous) {
        queryClient.setQueryData(careKeys.appointment(appointment.id), context.previous);
      }
    },
    onSettled: (_data, _error, { appointment }) => invalidate(queryClient, appointment.id),
  });
}
