'use client';

import { useQuery } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type {
  Appointment,
  AppointmentScope,
  CareEnums,
  CareToday,
  Medication,
} from '../model/types';
import {
  appointmentListSchema,
  appointmentSchema,
  careEnumsSchema,
  careTodaySchema,
  medicationListSchema,
  medicationSchema,
} from './schema';
import { careKeys, type MedicationFilters } from './keys';

/*
 * Reads for `/api/v1/care/*` (CLAUDE.md §8 — server state in TanStack Query).
 * Every hook is disabled until a token exists so public screens never fire it.
 * Personal health data (§11): never log what these return.
 */

/** GET /care/medications (`?active=1` when `filters.active`). */
export async function fetchMedications(filters: MedicationFilters = {}): Promise<Medication[]> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/care/medications', {
    params: filters.active ? { active: 1 } : undefined,
  });
  return medicationListSchema.parse(data.data ?? []);
}

export function useMedications(filters: MedicationFilters = {}) {
  return useQuery({
    queryKey: careKeys.medications(filters),
    queryFn: () => fetchMedications(filters),
    enabled: isAuthenticated(),
    retry: false,
  });
}

/** GET /care/medications/{id}. */
export async function fetchMedication(id: number): Promise<Medication> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>(`/care/medications/${id}`);
  return medicationSchema.parse(data.data);
}

export function useMedication(id: number | null) {
  return useQuery({
    queryKey: careKeys.medication(id ?? 0),
    queryFn: () => fetchMedication(id as number),
    enabled: isAuthenticated() && id !== null,
    retry: false,
  });
}

/** GET /care/appointments?scope= (backend default: upcoming, cancelled excluded). */
export async function fetchAppointments(scope: AppointmentScope = 'upcoming'): Promise<Appointment[]> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/care/appointments', {
    params: { scope },
  });
  return appointmentListSchema.parse(data.data ?? []);
}

export function useAppointments(scope: AppointmentScope = 'upcoming') {
  return useQuery({
    queryKey: careKeys.appointments(scope),
    queryFn: () => fetchAppointments(scope),
    enabled: isAuthenticated(),
    retry: false,
  });
}

/** GET /care/appointments/{id}. */
export async function fetchAppointment(id: number): Promise<Appointment> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>(`/care/appointments/${id}`);
  return appointmentSchema.parse(data.data);
}

export function useAppointment(id: number | null) {
  return useQuery({
    queryKey: careKeys.appointment(id ?? 0),
    queryFn: () => fetchAppointment(id as number),
    enabled: isAuthenticated() && id !== null,
    retry: false,
  });
}

/** GET /care/today (`?date=Y-m-d`; omitted = today in Tehran, server-side). */
export async function fetchCareToday(date?: string): Promise<CareToday> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/care/today', {
    params: date ? { date } : undefined,
  });
  return careTodaySchema.parse(data.data);
}

export function useCareToday(date?: string) {
  return useQuery({
    queryKey: careKeys.today(date),
    queryFn: () => fetchCareToday(date),
    enabled: isAuthenticated(),
    retry: false,
  });
}

/** GET /care/enums — localized option labels for the forms. */
export async function fetchCareEnums(): Promise<CareEnums> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/care/enums');
  return careEnumsSchema.parse(data.data ?? {});
}

export function useCareEnums() {
  return useQuery({
    queryKey: careKeys.enums(),
    queryFn: fetchCareEnums,
    enabled: isAuthenticated(),
    staleTime: 30 * 60_000,
    retry: false,
  });
}
