import type { MenopauseSection } from '@/entities/health-record';
import type { Locale } from '@/shared/i18n';
import { formatDayMonth, formatDecimal, formatNumber, fromApiDate, monthName, toParts } from '@/shared/lib/date';

import type { PaperBlock, Translate } from './paper';

/**
 * The menopause doctor report «گزارش برای پزشک» (CB-MENO-11, nbl_Meno_Report) as rows: the screen draws them and the
 * same rows go into bloom's paper (on-device PDF + the shared view), so the two never drift. Pure; `t` is
 * `menopause.report.paper`.
 */

export interface MenoRow {
  key: string;
  label: string;
  value: string;
}

export interface MenoSymptomBar {
  key: string;
  label: string;
  percent: number;
  share: string;
}

export interface MenoReportModel {
  summary: MenoRow[];
  symptoms: MenoSymptomBar[];
  treatment: MenoRow[];
}

const STAGES = ['peri', 'meno', 'post', 'unsure'];
const SIDE_EFFECTS = ['breast_tenderness', 'spotting', 'headache', 'bloating', 'mood_change'];

function list(items: string[], loc: Locale): string {
  try {
    return new Intl.ListFormat(loc, { style: 'long', type: 'conjunction' }).format(items);
  } catch {
    return items.join('، ');
  }
}

export function buildMenopauseModel(m: MenopauseSection, t: Translate, loc: Locale): MenoReportModel {
  const n = (x: number) => formatNumber(x, loc);
  const d = (x: number) => formatDecimal(x, loc);
  const none = t('none');

  const stageLabel = m.stage.code && STAGES.includes(m.stage.code) ? t(`stage.${m.stage.code}`) : null;
  const months = m.stage.monthsWithoutPeriod;
  const stage = stageLabel
    ? months !== null && months > 0
      ? t('stageValue', { stage: stageLabel, months: n(months) })
      : stageLabel
    : none;

  const score = m.score
    ? m.score.first !== m.score.last
      ? t('scoreChange', { first: n(m.score.first), last: n(m.score.last), max: n(m.score.max) })
      : t('scoreValue', { last: n(m.score.last), max: n(m.score.max) })
    : none;

  const lastBleed = m.bleeding.dates.at(-1);
  const summary: MenoRow[] = [
    { key: 'stage', label: t('rows.stage'), value: stage },
    { key: 'score', label: t('rows.score'), value: score },
    {
      key: 'hotFlashes',
      label: t('rows.hotFlashes'),
      value: m.hotFlashes.perDay !== null && m.hotFlashes.total > 0 ? t('perDay', { value: d(m.hotFlashes.perDay) }) : none,
    },
    {
      key: 'nightSweats',
      label: t('rows.nightSweats'),
      value: m.nightSweats.perWeek !== null && m.nightSweats.nights > 0 ? t('perWeek', { value: d(m.nightSweats.perWeek) }) : none,
    },
    { key: 'sleep', label: t('rows.sleep'), value: m.sleepAvg !== null ? t('sleepValue', { value: d(m.sleepAvg) }) : none },
    {
      key: 'bleeding',
      label: t('rows.bleeding'),
      value:
        m.bleeding.events > 0 && lastBleed
          ? t('bleedingValue', { count: m.bleeding.events, date: formatDayMonth(fromApiDate(lastBleed), loc) })
          : t('bleedingNone'),
    },
    {
      key: 'bp',
      label: t('rows.bp'),
      value: m.bloodPressure ? t('bpValue', { value: `${n(m.bloodPressure.systolic)}/${n(m.bloodPressure.diastolic)}` }) : none,
    },
  ];

  const symptoms = m.topSymptoms.map((s) => ({
    key: s.key,
    label: s.label,
    percent: Math.max(0, Math.min(100, s.percent)),
    share: t('share', { percent: n(s.percent) }),
  }));

  const treatment: MenoRow[] = [];
  for (const item of m.items) {
    if (item.kind !== 'hrt') continue;
    const parts = [item.dose ? `${item.name} ${item.dose}` : item.name];
    if (item.startedOn) parts.push(t('since', { month: monthName(toParts(fromApiDate(item.startedOn), loc).month, loc) }));
    if (item.adherencePct !== null) parts.push(t('adherence', { pct: n(item.adherencePct) }));
    if (item.stoppedOn) parts.push(t('stopped'));
    treatment.push({ key: `hrt-${treatment.length}`, label: t('rows.hrt'), value: parts.join(' · ') });
  }
  const effects = m.sideEffects.filter((e) => SIDE_EFFECTS.includes(e.code));
  if (effects.length) {
    treatment.push({
      key: 'sideEffects',
      label: t('rows.sideEffects'),
      value: list(
        effects.map((e) => t('effectDays', { effect: t(`effect.${e.code}`), days: e.days })),
        loc,
      ),
    });
  }
  if (m.supplements.length) {
    treatment.push({ key: 'supplements', label: t('rows.supplements'), value: list(m.supplements, loc) });
  }
  for (const l of m.lifestyle) {
    treatment.push({
      key: `life-${l.name}`,
      label: t('rows.lifestyle'),
      // minutes goals (e.g. 150 min walking) read «… دقیقه در هفته», session goals «… بار در هفته»
      value: l.perWeek !== null ? t('lifestyleValue', { name: l.name, value: d(l.perWeek), unit: l.goalUnit ?? 'sessions' }) : l.name,
    });
  }

  return { summary, symptoms, treatment };
}

/** The section's blocks on bloom's paper (heading + label/value rows), for the PDF and the shared view. */
export function menopausePaperBlocks(m: MenopauseSection, empty: boolean, t: Translate, loc: Locale): PaperBlock[] {
  const out: PaperBlock[] = [{ kind: 'heading', text: t('title') }];
  if (empty) return [...out, { kind: 'empty', text: t('empty') }];
  const model = buildMenopauseModel(m, t, loc);
  const w = [2, 3];
  for (const r of model.summary) out.push({ kind: 'row', cells: [r.label, r.value], weights: w });
  if (model.symptoms.length) {
    out.push({ kind: 'heading', text: t('symptomsTitle') });
    for (const s of model.symptoms) out.push({ kind: 'row', cells: [s.label, s.share], weights: w });
  }
  if (model.treatment.length) {
    out.push({ kind: 'heading', text: t('treatmentTitle') });
    for (const r of model.treatment) out.push({ kind: 'row', cells: [r.label, r.value], weights: w });
  }
  return out;
}
