'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { api } from '@/shared/api';

/** App module kill-switches (GET / PUT /modules, backend-go internal/modules). */

export const MODULE_KEYS = [
  'companion',
  'ttc',
  'ivf',
  'menopause',
  'teen',
  'postpartum',
  'children',
  'contraception',
  'programs',
  'mother_child',
  'learning',
  'my_courses',
  'shop',
  'insurance',
  'record',
  'devices',
  'doctors',
  'todo',
  'reminders',
  'home_articles',
  'services',
  'vitals',
  'backup',
  'support',
] as const;
export type ModuleKey = (typeof MODULE_KEYS)[number];

/** The screen's panels: life stages & family, the «خدمات» surfaces, the rows of «من», then the sections of Today. */
export const MODULE_GROUPS: readonly { key: 'stages' | 'services' | 'profile' | 'today'; modules: readonly ModuleKey[] }[] = [
  { key: 'stages', modules: ['companion', 'ttc', 'ivf', 'menopause', 'teen', 'postpartum', 'children', 'contraception'] },
  { key: 'services', modules: ['programs', 'mother_child', 'learning', 'shop', 'insurance', 'record', 'doctors'] },
  // «من» rows (user request 2026-11-25): each hides its row there and every other entry point of the same section.
  { key: 'profile', modules: ['services', 'vitals', 'my_courses', 'devices', 'backup', 'support'] },
  { key: 'today', modules: ['todo', 'reminders', 'home_articles'] },
];

const bodySchema = z.object({
  modules: z.record(z.string(), z.boolean()),
  keys: z.array(z.string()),
});
export type ModulesBody = z.infer<typeof bodySchema>;

const key = ['app-modules'] as const;

export function useModules() {
  return useQuery({ queryKey: key, queryFn: ({ signal }) => api.get('/modules', { schema: bodySchema, signal }) });
}

export function useSaveModules() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (body: Partial<Record<ModuleKey, boolean>>) => api.put('/modules', body, { schema: bodySchema }),
    onSuccess: (data) => client.setQueryData(key, data),
  });
}
