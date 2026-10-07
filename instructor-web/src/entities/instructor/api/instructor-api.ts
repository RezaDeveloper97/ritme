import { api } from '@/shared/api';

import { meSchema, type ApplyInput, type Me } from '../model/schema';

export const instructorKeys = {
  all: ['instructor'] as const,
  me: () => [...instructorKeys.all, 'me'] as const,
};

/** GET /api/instructor/v1/me — open to every signed-in user (never 403). */
export function fetchMe(signal?: AbortSignal): Promise<Me> {
  return api.get('/me', { schema: meSchema, signal });
}

/** POST /api/instructor/v1/apply — 201 new pending application, 200 update. */
export function applyInstructor(input: ApplyInput): Promise<Me> {
  return api.post('/apply', input, { schema: meSchema });
}
