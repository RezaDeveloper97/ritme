import { z } from 'zod';

import type { CheckupItem } from '@/entities/checkup';
import type { Tone } from '@/shared/ui';

/*
 * Checkups in menopause mode (CB-MENO-09, nbl_Meno_Checkups): the plan rows the
 * API already filters by audience (CB-MENO-01b), grouped by the admin-editable
 * `meno_checkup_groups` catalog, with the board's status chips. Pure — no
 * React, no locale.
 */

export interface MenoCheckupGroup {
  code: string;
  title: string | null;
  /** `checkup_types.key`s of the group, in display order. */
  keys: string[];
}

const groupSchema = z
  .object({
    code: z.string(),
    title: z.string().trim().min(1).nullable().catch(null),
    meta: z
      .object({ checkups: z.array(z.string()).catch([]) })
      .nullable()
      .catch(null),
  })
  .transform((d): MenoCheckupGroup => ({ code: d.code, title: d.title, keys: d.meta?.checkups ?? [] }));

/** `GET /catalog/meno_checkup_groups` → `data`; malformed items are dropped. */
export const menoCheckupGroupsSchema = z
  .object({ items: z.array(z.unknown()).catch([]) })
  .catch({ items: [] })
  .transform((g): MenoCheckupGroup[] =>
    g.items.flatMap((raw) => {
      const parsed = groupSchema.safeParse(raw);
      return parsed.success ? [parsed.data] : [];
    }),
  );

/** `GET /catalog/meno_tips` → the `checkups_intro` body (the board's top note), else null. */
export const menoCheckupsIntroSchema = z
  .object({ items: z.array(z.unknown()).catch([]) })
  .catch({ items: [] })
  .transform((g): string | null => {
    for (const raw of g.items) {
      const parsed = z
        .object({ code: z.literal('checkups_intro'), body: z.string().trim().min(1) })
        .safeParse(raw);
      if (parsed.success) return parsed.data.body;
    }
    return null;
  });

export interface MenoSection {
  /** Catalog group code, or `more` for plan rows no group lists. */
  code: string;
  title: string | null;
  items: CheckupItem[];
}

/**
 * Plan rows by catalog group, in the group's key order; a row appears once
 * (first group wins). Rows no group lists (shared types such as the breast
 * self-exam, custom checkups) follow in one `more` section, plan order. Empty
 * groups are dropped — the API already hid what isn't for her.
 */
export function groupForMenopause(items: readonly CheckupItem[], groups: readonly MenoCheckupGroup[]): MenoSection[] {
  const byKey = new Map(items.filter((i) => i.key).map((i) => [i.key as string, i]));
  const used = new Set<CheckupItem>();
  const sections: MenoSection[] = [];
  for (const group of groups) {
    const rows: CheckupItem[] = [];
    for (const key of group.keys) {
      const item = byKey.get(key);
      if (item && !used.has(item)) {
        used.add(item);
        rows.push(item);
      }
    }
    if (rows.length) sections.push({ code: group.code, title: group.title, items: rows });
  }
  const rest = items.filter((i) => !used.has(i));
  if (rest.length) sections.push({ code: 'more', title: null, items: rest });
  return sections;
}

/** The board's chips: ثبت کن · بپرس · نزدیک · به‌روز · عقب افتاده (+ غیرفعال). */
export type MenoChip = 'log' | 'ask' | 'soon' | 'up_to_date' | 'overdue' | 'disabled';

/**
 * Due monthly (home) checks are «ثبت کن»; anything else due, or not yet for
 * her age, is «بپرس» — the doctor decides the timing (checkups_intro).
 */
export function menoChip(item: Pick<CheckupItem, 'status' | 'category'>): MenoChip {
  switch (item.status) {
    case 'overdue':
    case 'soon':
    case 'up_to_date':
    case 'disabled':
      return item.status;
    case 'due':
      return item.category === 'monthly' ? 'log' : 'ask';
    default:
      return 'ask';
  }
}

export const MENO_CHIP_TONE: Record<MenoChip, Tone> = {
  log: 'brand',
  ask: 'danger',
  soon: 'warm',
  up_to_date: 'data',
  overdue: 'danger',
  disabled: 'neutral',
};

/**
 * The cadence on line 2. The menopause types (`meno_*`) carry the board's
 * wording in their subtitle («ماهانه در خانه، سالانه نزد پزشک», «معمولاً هر ۱
 * تا ۳ سال»); shared types read their interval («هر ۱ تا ۲ سال»).
 */
export function menoCadence(item: Pick<CheckupItem, 'key' | 'intervalLabel' | 'subtitle'>): string | null {
  const own = item.key?.startsWith('meno_') ?? false;
  return own ? (item.subtitle ?? item.intervalLabel) : (item.intervalLabel ?? item.subtitle);
}

/** What follows the interval on line 2: the due date when it's near, else the last time (or «not done yet»). */
export type MenoWhen = { kind: 'next'; date: string } | { kind: 'last'; date: string } | { kind: 'never' };

export function menoWhen(item: Pick<CheckupItem, 'status' | 'lastDoneOn' | 'nextDueOn'>): MenoWhen {
  if (item.status === 'soon' && item.nextDueOn) return { kind: 'next', date: item.nextDueOn };
  if (item.lastDoneOn) return { kind: 'last', date: item.lastDoneOn };
  return { kind: 'never' };
}
