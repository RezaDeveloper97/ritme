import type { LogCategory, LogDayValues, LogParam } from '@/entities/health-log';

/**
 * What the «ثبت امروز» card says about today's log (B-N1-06). Reads the log v2 day
 * (`GET /logs/days/{date}`) — the shape the log sheet saves — labelled through the taxonomy,
 * so every item the user picked is counted, not only the few that mirror into the legacy row.
 */
export interface TodayLogSummary {
  /** «شدت خونریزی: متوسط», «لکه‌بینی», or null when no bleeding was logged. */
  bleeding: string | null;
  /** Labels of the logged symptoms (every `items` param of the `symptoms` category), in taxonomy order. */
  symptoms: string[];
  /** Labels of the logged moods. */
  moods: string[];
}

export interface TodayLogLabels {
  /** Labels of the user's custom items, by item code. */
  extraLabels: Record<string, string>;
  /** A stored code the taxonomy can't name. */
  unknown: string;
}

function findParam(categories: readonly LogCategory[], category: string, param: string): LogParam | undefined {
  return categories.find((c) => c.code === category)?.params.find((p) => p.code === param);
}

function optionLabel(param: LogParam | undefined, code: string, labels: TodayLogLabels): string {
  return param?.options.find((o) => o.value === code)?.label ?? labels.extraLabels[code] ?? labels.unknown;
}

/** Item codes of an `items` value that were logged as present (`no` = explicitly «ندارم»). */
function loggedItems(value: unknown): string[] {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return [];
  return Object.entries(value as Record<string, { level?: unknown }>)
    .filter(([, v]) => v && typeof v === 'object' && v.level !== 'no')
    .map(([code]) => code);
}

export function summarizeTodayLog(
  day: LogDayValues | null | undefined,
  categories: readonly LogCategory[],
  labels: TodayLogLabels,
): TodayLogSummary {
  const values = day ?? {};

  const flow = values.bleeding?.flow;
  const flowParam = findParam(categories, 'bleeding', 'flow');
  const bleeding =
    typeof flow === 'string' && flow !== ''
      ? `${flowParam?.label ?? labels.unknown}: ${optionLabel(flowParam, flow, labels)}`
      : values.bleeding?.spotting === true
        ? (findParam(categories, 'bleeding', 'spotting')?.label ?? labels.unknown)
        : null;

  // Taxonomy order first; a stored param the taxonomy doesn't list still counts.
  const symptomValues = values.symptoms ?? {};
  const symptomCat = categories.find((c) => c.code === 'symptoms');
  const known = (symptomCat?.params ?? []).filter((p) => p.type === 'items').map((p) => p.code);
  const paramCodes = [...known, ...Object.keys(symptomValues).filter((k) => !known.includes(k))];
  const symptoms = paramCodes.flatMap((code) => {
    const param = symptomCat?.params.find((p) => p.code === code);
    return loggedItems(symptomValues[code]).map((item) => optionLabel(param, item, labels));
  });

  const moodValue = values.mood?.moods;
  const moodParam = findParam(categories, 'mood', 'moods');
  const moods = Array.isArray(moodValue)
    ? moodValue.filter((m): m is string => typeof m === 'string').map((m) => optionLabel(moodParam, m, labels))
    : [];

  return { bleeding, symptoms, moods };
}
