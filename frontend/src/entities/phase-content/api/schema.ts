import { z } from 'zod';

import type { PhaseContent, PhaseSectionKey } from '../model/types';

/**
 * Boundary parser for GET /cycle/phase-content/{phase} (§10 — validate external
 * data with zod). Lenient by design: unknown/extra section keys are tolerated
 * and snake_case maps to the domain shape. The API only sends sections that
 * have copy, so the screen can render whatever survives without extra guards.
 */
export const phaseContentSchema = z
  .object({
    phase: z.string(),
    phase_label: z.string().default(''),
    sections: z.record(z.string(), z.string()).default({}),
  })
  .transform(
    (v): PhaseContent => ({
      phase: v.phase,
      phaseLabel: v.phase_label,
      sections: v.sections as Partial<Record<PhaseSectionKey, string>>,
    }),
  );
