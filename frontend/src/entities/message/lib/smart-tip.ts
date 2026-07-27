import type { DailyMessage } from '../model/types';

/** Where the shown tip came from — useful for styling/telemetry, never for copy. */
export type SmartTipSource = 'message' | 'correlation' | 'pattern' | 'supplement';

/** The fully server-derived content of the «نکته هوشمند» card. */
export interface SmartTip {
  /** The paragraph. Always non-empty when a tip is returned. */
  body: string;
  /** The highlighted one-liner; empty when the source had no action to suggest. */
  action: string;
  source: SmartTipSource;
  /** Extra insights beyond the headline one, for the secondary list. */
  extras: string[];
}

const clean = (value: string | undefined | null): string => (value ?? '').trim();

const firstNonEmpty = (...values: (string | undefined | null)[]): string =>
  values.map(clean).find(v => v.length > 0) ?? '';

/**
 * Picks what the smart-tip card shows for a day, in the same priority order the
 * backend's own home-page section uses: the personalized message first, then a
 * detected correlation, then a multi-day pattern, then the supplement modules.
 * Returns `null` when the day genuinely has no server content — the caller must
 * then render nothing rather than invent a tip (§11: no fabricated health copy).
 */
export function selectSmartTip(message: DailyMessage | null | undefined): SmartTip | null {
  if (!message) return null;

  const { primary, correlations, patterns, supplements, tips } = message;
  const [correlation] = correlations;
  const [pattern] = patterns;

  const body = firstNonEmpty(
    primary.longMessage,
    primary.shortMessage,
    correlation?.insightMessage,
    pattern?.message,
    supplements.nutritionTip,
    supplements.exerciseTip,
    supplements.sleepTips[0],
    tips[0],
  );
  if (!body) return null;

  const source: SmartTipSource = clean(primary.longMessage) || clean(primary.shortMessage)
    ? 'message'
    : clean(correlation?.insightMessage)
      ? 'correlation'
      : clean(pattern?.message)
        ? 'pattern'
        : 'supplement';

  const action = firstNonEmpty(
    primary.actionSuggestion,
    correlation?.action,
    pattern?.recommendation,
    supplements.nutritionTip,
    supplements.exerciseTip,
  );

  // Anything already used as the body or the action would only read as a repeat.
  const extras = [
    correlation?.insightMessage,
    pattern?.message,
    supplements.sleepFocus,
    ...tips,
  ]
    .map(clean)
    .filter(v => v.length > 0 && v !== body && v !== action)
    .filter((v, i, all) => all.indexOf(v) === i)
    .slice(0, 2);

  return { body, action, source, extras };
}
