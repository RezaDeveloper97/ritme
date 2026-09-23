'use client';

import { z } from 'zod';

import { createResource, optionSchema } from '@/shared/api';

/** GET /phase-contents: one entry per content-backed sub-phase (+ legacy rows), with its row id or null. */
const mapSchema = z.object({
  items: z.array(z.object({ value: z.string(), label: z.string(), legacy: z.boolean(), id: z.number().nullable() })),
  fields: z.array(z.string()),
  phases: z.array(optionSchema),
});
export type PhaseMap = z.infer<typeof mapSchema>;

/** `PhaseContent`: id, phase and the nine translatable sections named by `fields`. */
const phaseContentSchema = z.object({ id: z.number(), phase: z.string() }).catchall(z.unknown());
export type PhaseContent = z.infer<typeof phaseContentSchema>;

export const phaseContentsApi = createResource({
  key: 'phase-contents',
  path: '/phase-contents',
  list: mapSchema,
  detail: z.object({ phase_content: phaseContentSchema }),
});
