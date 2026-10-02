'use client';

import { type QueryClient, useMutation, useQueryClient } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { careKeys, type Medication, medicationSchema } from '@/entities/care-reminder';
import { reminderKeys } from '@/entities/reminder';

import { type MedicationInput, type MedicationPatch, toMedicationBody, withForUser } from '../model/body';

/*
 * Create / update / delete / switch a medication reminder (v13_AddMedication,
 * v13_Reminders). Every mutation invalidates through `careKeys` — the lists,
 * the detail and today's doses — plus the legacy `/reminders` list, which
 * shows the same rows (README: legacy readers).
 *
 * Privacy (§11): medication names, doses and notes go to the API only.
 */

function invalidate(queryClient: QueryClient, id?: number): void {
  void queryClient.invalidateQueries({ queryKey: careKeys.medicationsAll() });
  void queryClient.invalidateQueries({ queryKey: careKeys.todayAll() });
  if (id !== undefined) void queryClient.invalidateQueries({ queryKey: careKeys.medication(id) });
  void queryClient.invalidateQueries({ queryKey: reminderKeys.all });
}

/** «ثبت برای …» (B-N4-06): the owner a companion with edit records for; omitted = own record. */
export interface ForUser {
  forUserId?: number | null;
}

/** POST /care/medications → 201 (`forUserId`: into that owner's list, edit grant required, else 403). */
export function useCreateMedication() {
  const queryClient = useQueryClient();
  return useMutation<Medication, unknown, MedicationInput & ForUser>({
    mutationFn: async ({ forUserId, ...input }) => {
      const { data } = await apiClient.post<ApiEnvelope<unknown>>(
        '/care/medications',
        withForUser(toMedicationBody(input), forUserId),
      );
      return medicationSchema.parse(data.data);
    },
    onSuccess: () => invalidate(queryClient),
  });
}

export interface UpdateMedicationVars extends ForUser {
  id: number;
  patch: MedicationPatch;
}

/** PUT /care/medications/{id} — partial (`forUserId`: the owner's record, edit grant required). */
export function useUpdateMedication() {
  const queryClient = useQueryClient();
  return useMutation<Medication, unknown, UpdateMedicationVars>({
    mutationFn: async ({ id, patch, forUserId }) => {
      const { data } = await apiClient.put<ApiEnvelope<unknown>>(
        `/care/medications/${id}`,
        withForUser(toMedicationBody(patch), forUserId),
      );
      return medicationSchema.parse(data.data);
    },
    onSuccess: (_data, { id }) => invalidate(queryClient, id),
  });
}

export interface ToggleMedicationVars {
  id: number;
  isActive: boolean;
}

/** The list switch: PUT /care/medications/{id} `{is_active}`. */
export function useToggleMedicationActive() {
  const queryClient = useQueryClient();
  return useMutation<Medication, unknown, ToggleMedicationVars>({
    mutationFn: async ({ id, isActive }) => {
      const { data } = await apiClient.put<ApiEnvelope<unknown>>(`/care/medications/${id}`, {
        is_active: isActive,
      });
      return medicationSchema.parse(data.data);
    },
    onSuccess: (_data, { id }) => invalidate(queryClient, id),
  });
}

/** DELETE /care/medications/{id}. */
export function useDeleteMedication() {
  const queryClient = useQueryClient();
  return useMutation<void, unknown, number>({
    mutationFn: async (id) => {
      await apiClient.delete(`/care/medications/${id}`);
    },
    onSuccess: (_data, id) => {
      queryClient.removeQueries({ queryKey: careKeys.medication(id) });
      invalidate(queryClient);
    },
  });
}
