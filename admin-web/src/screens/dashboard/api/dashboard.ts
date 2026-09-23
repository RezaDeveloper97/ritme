'use client';

import { useQuery } from '@tanstack/react-query';
import { z } from 'zod';

import { api } from '@/shared/api';

/** GET /dashboard (admin-api.md §6). */
const dashboardSchema = z.object({
  stats: z.object({
    users: z.number(),
    users_blocked: z.number(),
    users_new_week: z.number(),
    users_new_today: z.number(),
    articles: z.number(),
    affirmations: z.number(),
    challenges: z.number(),
    task_templates: z.number(),
    messages: z.number(),
    messages_pending: z.number(),
  }),
  recent_users: z.array(
    z.object({
      id: z.number(),
      name: z.string().nullable(),
      mobile: z.string().nullable(),
      email: z.string().nullable(),
      is_blocked: z.boolean(),
      blocked_at: z.string().nullable(),
      created_at: z.string().nullable(),
    }),
  ),
});
export type Dashboard = z.infer<typeof dashboardSchema>;
export type RecentUser = Dashboard['recent_users'][number];

export const dashboardKeys = {
  all: ['dashboard'] as const,
};

export function useDashboard() {
  return useQuery({
    queryKey: dashboardKeys.all,
    queryFn: ({ signal }) => api.get('/dashboard', { schema: dashboardSchema, signal }),
  });
}
