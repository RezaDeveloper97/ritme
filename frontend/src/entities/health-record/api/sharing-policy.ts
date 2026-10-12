import { z } from 'zod';

/*
 * What the owner may share right now (user decision 2026-10-12, CB-REC-06 L5), from GET /health-record/sharing and
 * GET /health-record/share-links: a teen-mode account makes doctor codes and 7-day report links only while she has an
 * active parent link; her emergency card is always available. The server enforces it (403 teen_share_needs_parent) —
 * the client only hides / explains.
 */

export interface SharingPolicy {
  teen: boolean;
  parentLinked: boolean;
  doctorSharing: boolean;
}

export const OPEN_SHARING_POLICY: SharingPolicy = { teen: false, parentLinked: false, doctorSharing: true };

/** The 403 error code of a doctor code / report link create by a teen without a parent link. */
export const TEEN_SHARE_NEEDS_PARENT = 'teen_share_needs_parent';

export const sharingPolicySchema = z
  .object({ teen: z.boolean(), parent_linked: z.boolean(), doctor_sharing: z.boolean() })
  .transform((p): SharingPolicy => ({ teen: p.teen, parentLinked: p.parent_linked, doctorSharing: p.doctor_sharing }))
  .optional()
  .catch(undefined)
  .transform((p): SharingPolicy => p ?? OPEN_SHARING_POLICY);
