import { z } from 'zod';

import { adminSchema } from '@/entities/admin';
import { api, setCsrfToken } from '@/shared/api';

const sessionSchema = z.object({ admin: adminSchema, csrf_token: z.string() });
export type Session = z.infer<typeof sessionSchema>;

export interface LoginInput {
  email: string;
  password: string;
  remember: boolean;
}

export async function login(input: LoginInput): Promise<Session> {
  const session = await api.post('/auth/login', input, { schema: sessionSchema });
  setCsrfToken(session.csrf_token);
  return session;
}

export async function fetchMe(opts: { signal?: AbortSignal; probe?: boolean } = {}): Promise<Session> {
  const session = await api.get('/auth/me', {
    schema: sessionSchema,
    signal: opts.signal,
    skipAuthRedirect: opts.probe,
  });
  setCsrfToken(session.csrf_token);
  return session;
}

export async function logout(): Promise<void> {
  try {
    await api.post('/auth/logout');
  } finally {
    setCsrfToken(null);
  }
}
