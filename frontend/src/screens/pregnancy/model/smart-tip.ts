import type { DailyMessage } from '@/entities/message';
import type { WeekTip } from '@/entities/pregnancy';

/**
 * The pregnancy home's «نکتهٔ هوشمند» — the same idea as the cycle home's smart tip: the week's admin tip
 * (`pregnancy_week_tip`) is the base, a symptom logged today replaces it for the day, and the engine's general
 * layers (pregnancy / common correlations, trimester nutrition, sleep and exercise) follow as short extras. All
 * text is server-written and already localized; with no server text at all the card renders nothing (§11).
 */
export interface PregnancySmartTip {
  /** `symptom`: today's log matched a symptom; `week`: the week's tip (or the engine's week/trimester copy). */
  kind: 'symptom' | 'week';
  /** The heading; empty when the source has none (the card then shows only the body). */
  title: string;
  body: string;
  /** The highlighted one-liner; empty when there is nothing to suggest. */
  action: string;
  /** Up to {@link MAX_EXTRAS} further lines, never repeating the body or the action. */
  extras: string[];
}

export const MAX_EXTRAS = 3;

const clean = (v: string | null | undefined): string => (v ?? '').trim();

export function selectPregnancySmartTip(
  week: WeekTip | null | undefined,
  daily: DailyMessage | null | undefined,
): PregnancySmartTip | null {
  // A cached cycle bundle (mode just switched) must never speak on the pregnancy home.
  const bundle = daily?.mode === 'pregnancy' ? daily : null;
  const override = bundle?.primary.override ?? null;

  let head: Omit<PregnancySmartTip, 'extras'> | null = null;
  if (override && clean(override.message)) {
    head = { kind: 'symptom', title: clean(override.label), body: clean(override.message), action: clean(override.action) };
  } else if (week && (clean(week.title) || clean(week.body))) {
    head = { kind: 'week', title: clean(week.title), body: clean(week.body), action: clean(bundle?.primary.actionSuggestion) };
  } else if (bundle) {
    const body = clean(bundle.primary.longMessage) || clean(bundle.primary.shortMessage);
    if (body) head = { kind: 'week', title: '', body, action: clean(bundle.primary.actionSuggestion) };
  }
  if (!head) return null;

  const seen = new Set([head.body, head.action, head.title].filter(Boolean));
  const extras: string[] = [];
  const candidates = bundle
    ? [
        ...bundle.correlations.map((c) => c.insightMessage),
        bundle.supplements.nutritionTip,
        bundle.supplements.sleepTips[0],
        bundle.supplements.exerciseTip,
      ]
    : [];
  for (const raw of candidates) {
    const line = clean(raw);
    if (!line || seen.has(line)) continue;
    seen.add(line);
    extras.push(line);
    if (extras.length === MAX_EXTRAS) break;
  }
  return { ...head, action: head.action === head.body ? '' : head.action, extras };
}
