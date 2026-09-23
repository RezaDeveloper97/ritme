'use client';

import { useMutation } from '@tanstack/react-query';
import { z } from 'zod';

import { api } from '@/shared/api';

/** PUT /auth/password — ends the admin's other sessions (admin-api.md §4). */
export function useChangePassword() {
  return useMutation({
    mutationFn: (body: { current_password: string; password: string; password_confirmation: string }) =>
      api.put('/auth/password', body, { schema: z.null().or(z.unknown()) }),
  });
}
