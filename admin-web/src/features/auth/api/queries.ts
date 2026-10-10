'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { can, type Admin, type PermissionLevel, type Permissions } from '@/entities/admin';
import { withBasePath } from '@/shared/config';

import { fetchMe, login, logout, type LoginInput } from './auth-api';

export const authKeys = {
  all: ['auth'] as const,
  me: () => [...authKeys.all, 'me'] as const,
};

/** The signed-in admin. A 401 here runs the global handler → /login. */
export function useMe() {
  return useQuery({
    queryKey: authKeys.me(),
    queryFn: ({ signal }) => fetchMe({ signal }),
    staleTime: 5 * 60_000,
    retry: false,
  });
}

/** For code rendered inside <AuthGate>: the admin is loaded by then. */
export function useCurrentAdmin(): Admin | null {
  return useMe().data?.admin ?? null;
}

/** The signed-in admin's capability matrix (B-N9-02); `{}` before it loads. */
export function useCurrentPermissions(): Permissions {
  return useMe().data?.permissions ?? EMPTY;
}

const EMPTY: Permissions = {};

/** Whether the signed-in admin holds `capability` at `level` (the API enforces the same with 403). */
export function useCan(capability: string, level: PermissionLevel = 'view'): boolean {
  const me = useMe().data;
  return me ? can(me.admin.role, me.permissions, capability, level) : false;
}

export function useLogin() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (input: LoginInput) => login(input),
    onSuccess: (session) => client.setQueryData(authKeys.me(), session),
  });
}

export function useLogout() {
  return useMutation({
    mutationFn: logout,
    // Full document replace: no React tree or query cache survives sign-out.
    onSettled: () => window.location.replace(withBasePath('/login?signed_out=1')),
  });
}
