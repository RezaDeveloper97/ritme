import { z } from 'zod';

import { api } from '@/shared/api';

import type { DurationChoice } from '../model/duration';
import { chunk } from '../model/phones';

const resultSchema = z.object({
  grants: z.array(z.object({ id: z.number().int(), phone: z.string(), renewed: z.boolean() })),
  summary: z.object({ total: z.number().int() }),
});

export interface GrantRequest {
  scope: 'group' | 'course';
  targetId: number;
  duration: DurationChoice;
  until?: string;
}

export interface GrantProgress {
  total: number;
  renewed: number;
  /** Phones the server accepted so far (a failed chunk leaves the rest unsent). */
  sent: number;
}

/**
 * POST /grants in chunks of 200. The answer is uniform per number (no registered / status info), so the
 * result only carries counts. On a failed chunk the error is thrown with the progress made so far attached.
 */
export async function sendGrants(
  phones: string[],
  req: GrantRequest,
  onProgress?: (p: GrantProgress) => void,
): Promise<GrantProgress> {
  const progress: GrantProgress = { total: 0, renewed: 0, sent: 0 };
  for (const part of chunk(phones)) {
    try {
      const res = await api.post(
        '/grants',
        {
          phones: part,
          scope: req.scope,
          target_id: req.targetId,
          duration: req.duration,
          ...(req.duration === 'until' ? { until: req.until } : {}),
        },
        { schema: resultSchema },
      );
      progress.total += res.summary.total;
      progress.renewed += res.grants.filter((g) => g.renewed).length;
      progress.sent += part.length;
      onProgress?.({ ...progress });
    } catch (e) {
      throw Object.assign(e as Error, { progress: { ...progress } });
    }
  }
  return progress;
}
