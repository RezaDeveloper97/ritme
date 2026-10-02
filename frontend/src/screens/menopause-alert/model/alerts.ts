import { z } from 'zod';

import type { IconName, Tone } from '@/shared/ui';

/*
 * The `meno_alerts` catalog (docs/canvas-build/menopause.md §5) as the alert
 * screen reads it. Clinical copy is admin-editable and [needs clinical review];
 * the screen only decides layout. No React, no locale.
 */

export interface MenoAlert {
  code: string;
  title: string | null;
  body: string | null;
  severity: 'caution' | 'urgent';
  primary: boolean;
  /** Dialled number for an emergency item («115»), digits only. */
  hotline: string | null;
  /** Card heading of the primary item («این مورد را به پزشک بگو»), picked by the API. */
  cta: string | null;
}

const text = z.string().trim().min(1).nullable().catch(null);

const alertSchema = z
  .object({
    code: z.string(),
    title: text,
    body: text,
    meta: z
      .object({
        severity: z.enum(['caution', 'urgent']).catch('caution'),
        primary: z.boolean().catch(false),
        hotline: z.string().regex(/^\d{2,6}$/).nullable().catch(null),
        cta: text,
      })
      .partial()
      .nullable()
      .catch(null),
  })
  .transform(
    (d): MenoAlert => ({
      code: d.code,
      title: d.title,
      body: d.body,
      severity: d.meta?.severity ?? 'caution',
      primary: d.meta?.primary ?? false,
      hotline: d.meta?.hotline ?? null,
      cta: d.meta?.cta ?? null,
    }),
  );

/** `GET /catalog/meno_alerts` → `data`; a malformed item is dropped, never the list. */
export const menoAlertsSchema = z
  .object({ items: z.array(z.unknown()).catch([]) })
  .catch({ items: [] })
  .transform((g): MenoAlert[] =>
    g.items.flatMap((raw) => {
      const parsed = alertSchema.safeParse(raw);
      return parsed.success ? [parsed.data] : [];
    }),
  );

/** The card of the screen: `postmenopausal_bleeding` (the `primary` item), else null. */
export function primaryAlert(items: readonly MenoAlert[]): MenoAlert | null {
  return items.find((a) => a.primary) ?? items.find((a) => a.code === 'postmenopausal_bleeding') ?? null;
}

/** «این موارد را هم زود خبر بده» — every other item, catalog order. */
export function tellEarlyAlerts(items: readonly MenoAlert[]): MenoAlert[] {
  const primary = primaryAlert(items);
  return items.filter((a) => a !== primary && (a.title || a.body));
}

const LOOK: Record<string, { icon: IconName; tone: Tone }> = {
  heavy_perimenopause_bleeding: { icon: 'drop', tone: 'period' },
  chest_pain_palpitations: { icon: 'zap', tone: 'danger' },
  one_sided_leg_swelling: { icon: 'walk', tone: 'danger' },
  breast_change: { icon: 'ribbon', tone: 'bloom' },
  persistent_low_mood: { icon: 'brain', tone: 'brand' },
  fragility_fracture: { icon: 'symptom', tone: 'brand' },
};

/** Icon disc of a list row; unknown (admin-added) codes get a severity-toned warning. */
export function alertLook(alert: Pick<MenoAlert, 'code' | 'severity'>): { icon: IconName; tone: Tone } {
  return LOOK[alert.code] ?? { icon: 'warning', tone: alert.severity === 'urgent' ? 'danger' : 'warm' };
}
