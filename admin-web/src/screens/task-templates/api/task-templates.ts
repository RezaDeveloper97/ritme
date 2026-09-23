'use client';

import { z } from 'zod';

import { createResource, optionSchema, pagedSchema, translationsSchema } from '@/shared/api';

/** `TaskTemplate` (admin-api.md §11). */
export const taskTemplateSchema = z.object({
  id: z.number(),
  key: z.string(),
  title: translationsSchema,
  description: translationsSchema,
  category: z.string(),
  icon: z.string().nullable(),
  cycle_phase: z.string().nullable(),
  is_active: z.boolean(),
  sort_order: z.number(),
});
export type TaskTemplate = z.infer<typeof taskTemplateSchema>;

export const taskTemplatesApi = createResource({
  key: 'task-templates',
  path: '/task-templates',
  list: pagedSchema(taskTemplateSchema, {}),
  detail: z.object({ task_template: taskTemplateSchema }),
  options: z.object({ phases: z.array(optionSchema), categories: z.array(optionSchema) }),
  alsoInvalidate: [['dashboard']],
});
