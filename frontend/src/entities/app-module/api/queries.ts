'use client';

import { useQuery } from '@tanstack/react-query';
import { z } from 'zod';

import { type ApiEnvelope, apiClient } from '@/shared/api';

import { ALL_MODULES_ON, APP_MODULES, type AppModule, type AppModules } from '../model/modules';

export const appModuleKeys = {
  all: ['app-modules'] as const,
};

const schema = z
  .record(z.string(), z.unknown())
  .transform((raw): AppModules => {
    const out = { ...ALL_MODULES_ON };
    for (const m of APP_MODULES) if (raw[m] === false) out[m] = false;
    return out;
  });

/** GET /modules — public (onboarding reads it before the profile exists). */
export async function fetchAppModules(): Promise<AppModules> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/modules');
  return schema.parse(data.data);
}

/** The module switches; every module reads as on until (or unless) the read succeeds. */
export function useAppModules(): AppModules {
  const query = useQuery({
    queryKey: appModuleKeys.all,
    queryFn: fetchAppModules,
    staleTime: 60_000,
    retry: 1,
  });
  return query.data ?? ALL_MODULES_ON;
}

/** Whether one module is on. */
export function useModuleEnabled(module: AppModule): boolean {
  return useAppModules()[module];
}

/** The raw query, for screens that must wait for the real answer before deciding (route guard, onboarding skip). */
export function useAppModulesQuery() {
  return useQuery({
    queryKey: appModuleKeys.all,
    queryFn: fetchAppModules,
    staleTime: 60_000,
    retry: 1,
  });
}
