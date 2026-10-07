import type { HealthReport, ValueSummary } from '@/entities/health-record';
import type { Locale } from '@/shared/i18n';
import { formatDecimal, formatLongDate, formatMonthLabel, formatNumber, fromApiDate, toParts } from '@/shared/lib/date';
import type { PdfBlock } from '@/shared/lib/pdf';

import { menopausePaperBlocks } from './menopause';

/**
 * One description of the doctor report «گزارش سلامت» (nbl_Record_Preview) that both the on-screen paper and the
 * on-device PDF render, so the two never drift. Pure: the caller passes translators (`recordExport.paper.*` and the
 * `healthRecord` codes) — nothing here logs or sends anything.
 */

export type Translate = (key: string, values?: Record<string, string | number>) => string;

export type PaperBlock =
  | { kind: 'heading'; text: string }
  | { kind: 'row'; cells: string[]; weights?: number[]; head?: boolean }
  | { kind: 'text'; text: string }
  | { kind: 'empty'; text: string };

export interface PaperModel {
  title: string;
  sub: string;
  name: string | null;
  meta: string;
  stats: { label: string; value: string }[];
  blocks: PaperBlock[];
  question: string | null;
  disclaimer: string;
}

const CHRONIC = ['diabetes', 'hypertension', 'thyroid', 'asthma', 'anemia', 'migraine', 'other'];
const GYN = ['pcos', 'endometriosis', 'fibroids', 'recurrent_infections', 'other'];
const PROFILE_MEDS = ['contraceptive_pill', 'iud', 'hormonal_medication'];
const RESULTS = ['normal', 'follow_up', 'pending'];
const DASH = '—';

function monthYear(iso: string, loc: Locale): string {
  const p = toParts(fromApiDate(iso), loc);
  return formatMonthLabel(p.year, p.month, loc);
}

function valueRange(v: ValueSummary, unit: string, n: (x: number) => string): string {
  return `${n(v.avg)}${unit} (${n(v.min)} – ${n(v.max)})`;
}

/**
 * Builds the paper of `report` (only its present sections) with the patient's optional question. `tm`
 * (`menopause.report.paper`) draws the CB-MENO-11 `menopause` section when the report has one.
 */
export function buildPaper(
  report: HealthReport,
  question: string | null,
  t: Translate,
  hr: Translate,
  loc: Locale,
  tm?: Translate,
): PaperModel {
  const n = (x: number) => formatNumber(x, loc);
  const d = (x: number) => formatDecimal(x, loc);
  const p = report.person;
  const meta = [
    p.age !== null ? t('age', { age: n(p.age) }) : null,
    p.gender === 'female' || p.gender === 'male' ? t(p.gender) : null,
  ]
    .filter(Boolean)
    .join(' · ');

  const stats: PaperModel['stats'] = [];
  if (report.basics) stats.push({ label: t('bmi'), value: report.basics.data.bmi ? d(report.basics.data.bmi.value) : DASH });
  const bp = report.vitals?.data.bloodPressure;
  if (report.vitals) stats.push({ label: t('bp'), value: bp ? `${n(bp.systolic)}/${n(bp.diastolic)}` : DASH });
  if (report.vitals) {
    const f = report.vitals.data.glucoseFasting;
    stats.push({ label: t('fasting'), value: f ? n(f.avg) : DASH });
  }
  if (report.cycle) {
    const c = report.cycle.data;
    stats.push({
      label: t('cycle'),
      value:
        c.medianCycle !== null
          ? c.variation !== null
            ? t('cycleValue', { median: n(c.medianCycle), variation: n(c.variation) })
            : n(c.medianCycle)
          : DASH,
    });
  }

  const blocks: PaperBlock[] = [];

  // Conditions · medications · allergies (one table, like the artboard).
  if (report.conditions || report.medications || report.allergies) {
    blocks.push({ kind: 'heading', text: t('health') });
    const rows: PaperBlock[] = [];
    const c = report.conditions?.data;
    for (const code of (c?.chronicIllnesses ?? []).filter((x) => CHRONIC.includes(x))) {
      rows.push({ kind: 'row', cells: [hr(`conditions.chronicIllness.${code}`), '', ''], weights: [3, 3, 2] });
    }
    for (const code of (c?.gynConditions ?? []).filter((x) => GYN.includes(x))) {
      rows.push({ kind: 'row', cells: [hr(`conditions.gynCondition.${code}`), '', ''], weights: [3, 3, 2] });
    }
    const m = report.medications?.data;
    for (const med of m?.items ?? []) {
      const days = new Set(med.weekdays).size;
      const schedule = days === 0 || days >= 7 ? t('daily') : t('weekly', { count: days });
      rows.push({ kind: 'row', cells: [med.title, med.dose ?? '', schedule], weights: [3, 3, 2] });
    }
    for (const code of (m?.profileMedications ?? []).filter((x) => PROFILE_MEDS.includes(x))) {
      rows.push({ kind: 'row', cells: [hr(`medications.profile.${code}`), '', ''], weights: [3, 3, 2] });
    }
    for (const a of report.allergies?.data.items ?? []) {
      rows.push({ kind: 'row', cells: [t('allergy'), a, ''], weights: [3, 3, 2] });
    }
    blocks.push(...(rows.length ? rows : [{ kind: 'empty' as const, text: t('empty') }]));
  }

  if (report.cycle) {
    const c = report.cycle.data;
    blocks.push({ kind: 'heading', text: t('cycles', { count: c.basedOn }) });
    if (report.cycle.empty || c.medianCycle === null) {
      blocks.push({ kind: 'empty', text: t('empty') });
    } else {
      const parts = [
        t('cycleSummary', { median: n(c.medianCycle), period: c.medianPeriod !== null ? n(c.medianPeriod) : DASH }),
        t('regularity', { value: hr(`cycle.${c.regularity === 'regular' || c.regularity === 'irregular' ? c.regularity : 'not_enough_data'}`) }),
      ];
      if (c.topSymptoms.length) parts.push(t('topSymptoms', { list: c.topSymptoms.map((s) => s.label).join('، ') }));
      blocks.push({ kind: 'text', text: parts.join(' · ') });
    }
  }

  if (report.vitals) {
    const v = report.vitals.data;
    blocks.push({ kind: 'heading', text: t('vitals', { days: n(v.days) }) });
    const w = [3, 4, 2];
    const rows: PaperBlock[] = [];
    if (v.bloodPressure) {
      const b = v.bloodPressure;
      const range = b.min && b.max ? ` (${n(b.min.systolic)}/${n(b.min.diastolic)} – ${n(b.max.systolic)}/${n(b.max.diastolic)})` : '';
      rows.push({ kind: 'row', cells: [t('bpRow'), `${n(b.systolic)}/${n(b.diastolic)}${range}`, n(b.readings)], weights: w });
    }
    if (v.heartRate) rows.push({ kind: 'row', cells: [t('hrRow'), valueRange(v.heartRate, ' bpm', n), n(v.heartRate.readings)], weights: w });
    if (v.glucoseFasting) rows.push({ kind: 'row', cells: [t('fastingRow'), valueRange(v.glucoseFasting, ' mg/dL', n), n(v.glucoseFasting.readings)], weights: w });
    if (v.glucoseAfterMeal) rows.push({ kind: 'row', cells: [t('mealRow'), valueRange(v.glucoseAfterMeal, ' mg/dL', n), n(v.glucoseAfterMeal.readings)], weights: w });
    if (v.glucoseOther) rows.push({ kind: 'row', cells: [t('otherRow'), valueRange(v.glucoseOther, ' mg/dL', n), n(v.glucoseOther.readings)], weights: w });
    if (rows.length) {
      blocks.push({ kind: 'row', cells: [t('metric'), t('avgRange'), t('count')], weights: w, head: true }, ...rows);
    } else {
      blocks.push({ kind: 'empty', text: t('empty') });
    }
  }

  if (report.pregnancies) {
    const pr = report.pregnancies.data;
    blocks.push({ kind: 'heading', text: t('pregnancies') });
    if (report.pregnancies.empty) {
      blocks.push({ kind: 'empty', text: t('empty') });
    } else {
      blocks.push({ kind: 'text', text: t('pregnanciesValue', { count: pr.pregnanciesCount, births: pr.birthsCount }) });
      for (const e of pr.items) {
        blocks.push({
          kind: 'row',
          cells: [hr(`pregnancies.outcome.${e.outcome}`), e.date ? monthYear(e.date, loc) : '', e.babyCount ? hr('pregnancies.babies', { count: e.babyCount }) : ''],
          weights: [3, 3, 2],
        });
      }
    }
  }

  if (report.checkups) {
    blocks.push({ kind: 'heading', text: t('checkups') });
    const items = report.checkups.data.items;
    if (!items.length) blocks.push({ kind: 'empty', text: t('empty') });
    for (const c of items) {
      const result = RESULTS.includes(c.result) ? hr(`checkups.result.${c.result}`) : '';
      blocks.push({ kind: 'row', cells: [c.title, monthYear(c.doneOn, loc), result], weights: [3, 3, 2] });
    }
  }

  if (report.labs && report.labs.data.items.length) {
    blocks.push({ kind: 'heading', text: t('labs') });
    for (const l of report.labs.data.items) {
      const status = l.allNormal
        ? t('labNormal')
        : l.attention.length
          ? l.attention.map((a) => `${a.name} ${a.stateLabel}`).join('، ')
          : t('labAttention', { count: l.attentionCount });
      blocks.push({ kind: 'row', cells: [l.title, monthYear(l.date, loc), status], weights: [3, 2, 3] });
    }
  }

  if (report.menopause && tm) {
    blocks.push(...menopausePaperBlocks(report.menopause.data, report.menopause.empty, tm, loc));
  }

  return {
    title: t('title'),
    sub: t('sub', { from: formatLongDate(fromApiDate(report.range.from), loc), to: formatLongDate(fromApiDate(report.range.to), loc) }),
    name: p.name,
    meta,
    stats,
    blocks,
    question: question?.trim() ? question.trim() : null,
    disclaimer: t('disclaimer'),
  };
}

/** The paper as PDF blocks for `shared/lib/pdf` (drawn on the device, nothing uploaded). */
export function paperPdfBlocks(m: PaperModel, questionLabel: string): PdfBlock[] {
  const out: PdfBlock[] = [{ kind: 'title', text: m.title }, { kind: 'subtitle', text: m.sub }];
  if (m.name || m.meta) out.push({ kind: 'heading', text: [m.name, m.meta].filter(Boolean).join(' · ') });
  if (m.stats.length) out.push({ kind: 'row', cells: m.stats.map((s) => `${s.label}: ${s.value}`) });
  out.push({ kind: 'rule' });
  for (const b of m.blocks) {
    if (b.kind === 'heading') out.push({ kind: 'heading', text: b.text });
    else if (b.kind === 'row') out.push({ kind: 'row', cells: b.cells, weights: b.weights });
    else if (b.kind === 'text') out.push({ kind: 'text', text: b.text });
    else out.push({ kind: 'muted', text: b.text });
  }
  if (m.question) out.push({ kind: 'note', text: `${questionLabel} ${m.question}` });
  out.push({ kind: 'rule' }, { kind: 'muted', text: m.disclaimer });
  return out;
}
