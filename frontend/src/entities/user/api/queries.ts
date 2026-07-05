'use client';

import { useQuery } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type { AuthUser, UserProfile } from '../model/types';
import { authUserSchema, userProfileSchema } from './schema';

/**
 * Query-key factory for the authenticated account (CLAUDE.md §8). Mutations
 * (verify-otp, logout, profile updates) invalidate via this factory — never
 * hand-written arrays.
 */
export const userKeys = {
  all: ['user'] as const,
  current: () => [...userKeys.all, 'current'] as const,
  profile: () => [...userKeys.all, 'profile'] as const,
};

interface CurrentUserResponse {
  user: unknown;
}

/** GET /auth/user — the current account. Requires a bearer token. */
export async function fetchCurrentUser(): Promise<AuthUser> {
  const { data } = await apiClient.get<ApiEnvelope<CurrentUserResponse>>('/auth/user');
  return authUserSchema.parse(data.data?.user);
}

/**
 * Reads the current account from the server (§8 — server state stays in
 * TanStack Query, never mirrored into Zustand). Disabled until a token exists
 * so it doesn't fire an unauthenticated request on public screens.
 */
export function useCurrentUser() {
  return useQuery({
    queryKey: userKeys.current(),
    queryFn: fetchCurrentUser,
    enabled: isAuthenticated(),
    staleTime: 5 * 60_000,
    retry: false,
  });
}

/** GET /profile — account identity plus the health profile. Requires a token. */
export async function fetchUserProfile(): Promise<UserProfile> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/profile');
  return userProfileSchema.parse(data.data);
}

/**
 * Reads the full profile (identity + health data) from the server (§8 — server
 * state stays in TanStack Query, never mirrored into Zustand). Disabled until a
 * token exists so it doesn't fire on public screens. Sensitive data (§11):
 * consumed for display only, never logged.
 */
export function useUserProfile() {
  return useQuery({
    queryKey: userKeys.profile(),
    queryFn: fetchUserProfile,
    enabled: isAuthenticated(),
    staleTime: 5 * 60_000,
    retry: false,
  });
}
