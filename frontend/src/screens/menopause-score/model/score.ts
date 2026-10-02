import {
  MENOPAUSE_SCORE_DOMAINS,
  type MenopauseScoreBand,
  type MenopauseScoreDomain,
  type MenopauseScoreHistory,
  type MenopauseScoreQuestion,
} from '@/entities/menopause';
import type { Tone } from '@/shared/ui';

/*
 * Pure view logic of the monthly score screen (CB-MENO-08, nbl_Meno_Score).
 * No React, no locale — unit-tested.
 */

/** The monthly questionnaire's route (a flow under the score tab). */
export const QUESTIONNAIRE_PATH = '/menopause/score/questionnaire';

/** Band tint, low → high (board: turquoise · lavender · pink · deep rose). */
const BAND_TONES: readonly Tone[] = ['data', 'brand', 'bloom', 'danger'];

/** Domain bar tint (board: somatic deep rose, psychological pink, urogenital violet). */
const DOMAIN_TONES: Record<string, Tone> = { somatic: 'danger', psychological: 'bloom', urogenital: 'brand' };

/** A domain the screen has a label for (an admin-added one shows its code). */
export function isKnownDomain(code: string): code is MenopauseScoreDomain {
  return (MENOPAUSE_SCORE_DOMAINS as readonly string[]).includes(code);
}

export function domainTone(code: string): Tone {
  return DOMAIN_TONES[code] ?? 'brand';
}

export interface BandSegment {
  code: string;
  title: string | null;
  tone: Tone;
  /** Share of the bar, in % (proportional to the band's range of totals). */
  share: number;
}

/** The header's band bar: one segment per band, sized by how many totals it covers. */
export function bandSegments(bands: readonly MenopauseScoreBand[]): BandSegment[] {
  const sorted = [...bands].sort((a, b) => a.min - b.min);
  const total = sorted.reduce((s, b) => s + Math.max(1, b.max - b.min + 1), 0);
  return sorted.map((b, i) => ({
    code: b.code,
    title: b.title,
    tone: BAND_TONES[Math.min(i, BAND_TONES.length - 1)],
    share: total > 0 ? Math.round((Math.max(1, b.max - b.min + 1) / total) * 1000) / 10 : 0,
  }));
}

/** Where her total sits on the band bar, in % from the inline start (centre of its unit). */
export function markerPercent(total: number, bands: readonly MenopauseScoreBand[]): number {
  const sorted = [...bands].sort((a, b) => a.min - b.min);
  if (!sorted.length) return 0;
  const lo = sorted[0].min;
  const hi = sorted[sorted.length - 1].max;
  const units = hi - lo + 1;
  if (units <= 0) return 0;
  const v = Math.max(lo, Math.min(hi, total));
  return Math.round(((v - lo + 0.5) / units) * 1000) / 10;
}

/** The tone of the band her total is in (for the pill). */
export function bandTone(code: string | null | undefined, bands: readonly MenopauseScoreBand[]): Tone {
  const seg = bandSegments(bands).find((s) => s.code === code);
  return seg?.tone ?? 'neutral';
}

export type HrtNote = { kind: 'less' | 'more'; points: number } | { kind: 'same' } | null;

/** The HRT annotation under the chart; null when there is nothing to compare. */
export function hrtNote(change: number | null | undefined): HrtNote {
  if (change === null || change === undefined) return null;
  if (change === 0) return { kind: 'same' };
  return change < 0 ? { kind: 'less', points: -change } : { kind: 'more', points: change };
}

/** Is this Jalali month's questionnaire filled? (the trend's last point is the current month). */
export function filledThisMonth(history: Pick<MenopauseScoreHistory, 'trend'>): boolean {
  const last = history.trend.at(-1);
  return Boolean(last && last.total !== null);
}

/** Prefill: this month's answers when she already filled it, else nothing. */
export function initialAnswers(
  history: Pick<MenopauseScoreHistory, 'trend' | 'latest'> | undefined,
): Record<string, number> {
  if (!history?.latest || !filledThisMonth(history)) return {};
  const last = history.trend.at(-1);
  return last && last.month === history.latest.month ? { ...history.latest.answers } : {};
}

export interface QuestionGroup {
  domain: string;
  questions: MenopauseScoreQuestion[];
}

/** Questions grouped by domain, in catalog order of each domain's first question. */
export function groupQuestions(questions: readonly MenopauseScoreQuestion[]): QuestionGroup[] {
  const groups: QuestionGroup[] = [];
  for (const q of questions) {
    const g = groups.find((x) => x.domain === q.domain);
    if (g) g.questions.push(q);
    else groups.push({ domain: q.domain, questions: [q] });
  }
  return groups;
}

/** Every question answered within its range. */
export function isComplete(questions: readonly MenopauseScoreQuestion[], answers: Record<string, number>): boolean {
  return questions.length > 0 && questions.every((q) => Number.isInteger(answers[q.code]) && answers[q.code] >= 0 && answers[q.code] <= q.max);
}

/** Only the answers to the active questions (stale codes would fail validation). */
export function answersFor(questions: readonly MenopauseScoreQuestion[], answers: Record<string, number>): Record<string, number> {
  const out: Record<string, number> = {};
  for (const q of questions) if (answers[q.code] !== undefined) out[q.code] = answers[q.code];
  return out;
}
