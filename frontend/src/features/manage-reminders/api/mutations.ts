'use client';

import { useMutation, useQueryClient } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import {
  type Reminder,
  type ReminderRecurrence,
  type ReminderType,
  reminderKeys,
  reminderSchema,
} from '@/entities/reminder';

/**
 * Create/update/delete actions for reminders (CLAUDE.md §8). All of them
 * invalidate the reminder cache through the entity's key factory.
 *
 * Privacy (§11): reminder titles/notes are personal health context — they go
 * to the API and nowhere else (no logs, no analytics).
 */

export interface CreateReminderInput {
  type: ReminderType;
  title: string;
  subtitle?: string;
  notes?: string;
  /** ISO datetime for a one-off reminder. */
  scheduledAt?: string;
  recurrence?: ReminderRecurrence;
  /** "HH:MM" time of day for recurring reminders. */
  recurrenceTime?: string;
  startsOn?: string;
  endsOn?: string;
}

export interface UpdateReminderInput extends Partial<CreateReminderInput> {
  id: string;
  isActive?: boolean;
}

/** camelCase input → the API's snake_case body, skipping undefined fields. */
function toBody(input: Partial<CreateReminderInput> & { isActive?: boolean }) {
  const body: Record<string, string | boolean> = {};
  if (input.type !== undefined) body.type = input.type;
  if (input.title !== undefined) body.title = input.title;
  if (input.subtitle !== undefined) body.subtitle = input.subtitle;
  if (input.notes !== undefined) body.notes = input.notes;
  if (input.scheduledAt !== undefined) body.scheduled_at = input.scheduledAt;
  if (input.recurrence !== undefined) body.recurrence = input.recurrence;
  if (input.recurrenceTime !== undefined) body.recurrence_time = input.recurrenceTime;
  if (input.startsOn !== undefined) body.starts_on = input.startsOn;
  if (input.endsOn !== undefined) body.ends_on = input.endsOn;
  if (input.isActive !== undefined) body.is_active = input.isActive;
  return body;
}

/** POST /reminders — creates a reminder and refreshes the list. */
export function useCreateReminder() {
  const queryClient = useQueryClient();
  return useMutation<Reminder, unknown, CreateReminderInput>({
    mutationFn: async (input) => {
      const { data } = await apiClient.post<ApiEnvelope<unknown>>('/reminders', toBody(input));
      return reminderSchema.parse(data.data);
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: reminderKeys.all });
    },
  });
}

/** PUT /reminders/{id} — updates any subset of fields (incl. is_active). */
export function useUpdateReminder() {
  const queryClient = useQueryClient();
  return useMutation<Reminder, unknown, UpdateReminderInput>({
    mutationFn: async ({ id, ...input }) => {
      const { data } = await apiClient.put<ApiEnvelope<unknown>>(
        `/reminders/${id}`,
        toBody(input),
      );
      return reminderSchema.parse(data.data);
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: reminderKeys.all });
    },
  });
}

/** DELETE /reminders/{id} — removes a reminder and refreshes the list. */
export function useDeleteReminder() {
  const queryClient = useQueryClient();
  return useMutation<void, unknown, string>({
    mutationFn: async (id) => {
      await apiClient.delete(`/reminders/${id}`);
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: reminderKeys.all });
    },
  });
}
