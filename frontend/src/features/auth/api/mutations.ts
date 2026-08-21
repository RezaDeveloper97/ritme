'use client';

import { useMutation, useQueryClient } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import {
  clearAuthToken,
  clearOnboardingPending,
  getOnboardingPending,
  setAuthToken,
  setOnboardingPending,
} from '@/shared/session';
import { env } from '@/shared/config';
import {
  authUserSchema,
  resetOnboardingFor,
  type AuthUser,
  userKeys,
} from '@/entities/user';

/**
 * The OTP login/registration actions (CLAUDE.md §8.1). One backend account
 * covers both sign-in and sign-up: `send-otp` creates the user if new, and
 * `new_user` tells the UI whether to route into onboarding.
 *
 * Privacy (§11): the mobile number is the only PII sent, and nothing here is
 * logged.
 */

interface SendOtpResult {
  newUser: boolean;
  expiresIn: number;
}

/** POST /auth/send-otp — sends a 4-digit code to the mobile number. */
export function useSendOtp() {
  return useMutation<SendOtpResult, unknown, string>({
    mutationFn: async (mobile) => {
      const { data } = await apiClient.post<
        ApiEnvelope<{ new_user: boolean; expires_in: number }>
      >('/auth/send-otp', { mobile, is_test: env.otpTestMode });
      return {
        newUser: data.data?.new_user ?? false,
        expiresIn: data.data?.expires_in ?? 120,
      };
    },
  });
}

interface VerifyOtpInput {
  mobile: string;
  code: string;
}

interface VerifyOtpResult {
  user: AuthUser;
  newUser: boolean;
  /**
   * Whether this account ever finished signup onboarding. Distinct from
   * `newUser`, which only reports that the account was created by *this*
   * verification: someone who abandoned the flow last time is a returning user
   * with nothing registered, and must be sent back into it.
   */
  profileCompleted: boolean;
}

/**
 * POST /auth/verify-otp — exchanges the code for a JWT. On success the token is
 * stored (so subsequent requests carry the bearer header) and the current-user
 * cache is seeded/invalidated via the entity's key factory.
 */
export function useVerifyOtp() {
  const queryClient = useQueryClient();
  return useMutation<VerifyOtpResult, unknown, VerifyOtpInput>({
    mutationFn: async ({ mobile, code }) => {
      const { data } = await apiClient.post<
        ApiEnvelope<{
          user: unknown;
          new_user: boolean;
          profile_completed?: boolean;
          access_token: string;
        }>
      >('/auth/verify-otp', { mobile, code });

      const token = data.data?.access_token;
      if (!token) throw new Error('Missing access token');
      setAuthToken(token);

      const user = authUserSchema.parse(data.data?.user);
      return {
        user,
        newUser: data.data?.new_user ?? false,
        profileCompleted: data.data?.profile_completed ?? false,
      };
    },
    onSuccess: ({ user, profileCompleted }) => {
      queryClient.setQueryData(userKeys.current(), user);
      // Answers are persisted per device; a different account signing in on this
      // one must not inherit them (nor a stale resume position).
      const switchedAccount = resetOnboardingFor(user.id);
      if (profileCompleted) {
        clearOnboardingPending();
        return;
      }
      // Resume where this device left off, unless the answers just belonged to
      // someone else — then the flow starts over.
      const resumeAt = switchedAccount ? null : getOnboardingPending();
      setOnboardingPending(resumeAt ?? 'name');
    },
  });
}

/**
 * POST /auth/logout — revokes the token server-side, then clears local session
 * and cache regardless of the network outcome (a stale token is worse than a
 * failed logout call).
 */
export function useLogout() {
  const queryClient = useQueryClient();
  return useMutation<void, unknown, void>({
    mutationFn: async () => {
      await apiClient.post('/auth/logout').catch(() => undefined);
    },
    onSettled: () => {
      clearAuthToken();
      queryClient.clear();
    },
  });
}
