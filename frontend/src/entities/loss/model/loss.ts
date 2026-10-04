import type { LossCatalogItem, LossEvent, LossHotline, LossNextStep } from './types';

/*
 * Pure rules of the loss path (no React, no locale). Clinical copy stays in the
 * catalog; these only decide what to show and where to go.
 */

/** Digits of a dialled number, or null («115», «1480»). */
function phone(value: unknown): string | null {
  return typeof value === 'string' && /^\d{2,6}$/.test(value) ? value : null;
}

/** The life mode a next step switches to (catalog `meta.life_mode`, falling back to the CB-LOSS-01 rule). */
export function lifeModeOfNextStep(step: LossNextStep, item?: LossCatalogItem | null): 'cycle' | 'ttc' {
  const fromMeta = item?.meta?.life_mode;
  if (fromMeta === 'ttc' || fromMeta === 'cycle') return fromMeta;
  return step === 'ttc' ? 'ttc' : 'cycle';
}

/** The emergency number of the danger card: a warning sign's `meta.hotline`, else the `emergency` hotline, else 115. */
export function emergencyNumber(warningSigns: readonly LossCatalogItem[], hotlines: readonly LossCatalogItem[]): string {
  for (const sign of warningSigns) {
    const n = phone(sign.meta?.hotline);
    if (n) return n;
  }
  const emergency = hotlines.find((h) => h.code === 'emergency' || h.meta?.kind === 'medical');
  return phone(emergency?.meta?.number) ?? '115';
}

/** Lower-cases a leading capital that starts an ordinary word («Severe» → «severe»; «hCG», «IVF», Persian unchanged). */
function midSentence(text: string): string {
  return /^\p{Lu}(?:\p{Ll}|\s)/u.test(text) ? text.charAt(0).toLowerCase() + text.slice(1) : text;
}

/**
 * The warning signs as one sentence part each: «خونریزی خیلی زیاد (پر شدن ۲ نوار …)».
 * Catalog titles are written to stand alone («A fever of 38 °C»), so inside the joined sentence every part but the
 * first, and every detail in brackets, drops its leading capital. Items without a title are skipped.
 */
export function warningSignParts(warningSigns: readonly LossCatalogItem[]): string[] {
  return warningSigns
    .filter((s) => s.title)
    .map((s, i) => {
      const title = i === 0 ? (s.title as string) : midSentence(s.title as string);
      return s.body ? `${title} (${midSentence(s.body)})` : title;
    });
}

/**
 * The crisis line's numbers (`loss_support/crisis` meta.hotlines), falling back to
 * the `loss_hotlines` that are not the medical emergency (1480, 123).
 */
export function crisisHotlines(crisis: LossCatalogItem | null | undefined, hotlines: readonly LossCatalogItem[]): LossHotline[] {
  const fromMeta = Array.isArray(crisis?.meta?.hotlines) ? (crisis.meta.hotlines as unknown[]) : [];
  const parsed = fromMeta.flatMap((raw): LossHotline[] => {
    if (typeof raw !== 'object' || raw === null) return [];
    const r = raw as Record<string, unknown>;
    const number = phone(r.number);
    return number ? [{ number, label: typeof r.label === 'string' && r.label.trim() ? r.label : null }] : [];
  });
  if (parsed.length) return parsed;
  return hotlines.flatMap((h): LossHotline[] => {
    const number = phone(h.meta?.number);
    if (!number || h.meta?.kind === 'medical' || h.code === 'emergency') return [];
    return [{ number, label: h.title }];
  });
}

/**
 * Whether «ادامه» on the Start screen should (re)record the loss. CB-LOSS-01
 * treats a POST on the same Tehran day as a correction and on a later day as a
 * new loss — so an earlier loss is only re-posted while she is pregnant again.
 */
export function shouldRecordLoss(loss: LossEvent | null, pregnant: boolean, todayIso: string): boolean {
  if (!loss) return true;
  if (pregnant) return true;
  return loss.createdAt?.slice(0, 10) === todayIso;
}

/** `Y-m-d` + `HH:mm` → the visit's Tehran wall clock (`Y-m-d H:i`). */
export function visitAt(day: string, time: string): string {
  return `${day} ${time}`;
}

/** The `Y-m-d` and `HH:mm` of a care appointment's `scheduled_at` (`Y-m-d H:i:s`). */
export function splitWallClock(at: string): { day: string; time: string } {
  const [day = '', clock = ''] = at.split(' ');
  return { day, time: clock.slice(0, 5) };
}
