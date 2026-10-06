import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

import { describe, expect, it } from 'vitest';

import { healthReportSchema, sharedReportSchema } from '@/entities/health-record';

import { buildPaper, paperPdfBlocks, type Translate } from './paper';

/* The paper model against the Go-recorded goldens of /health-record/report and /shared-reports (B-N6-04). */
const GOLDEN = resolve(process.cwd(), '../backend-go/contract/golden/sharelinks');
const steps = (name: string) =>
  (JSON.parse(readFileSync(resolve(GOLDEN, `${name}.json`), 'utf8')) as { steps: { body: { data?: unknown } }[] }).steps;

const t: Translate = (key, values) => (values ? `${key}(${Object.values(values).join(',')})` : key);

describe('doctor report paper', () => {
  it('parses the full report and builds every chosen section', () => {
    const report = healthReportSchema.parse(steps('report.fa')[0]?.body.data);
    expect(report.range.key).toBe('3m');
    expect(report.basics?.data.heightCm).toBe(165);
    expect(report.medications?.data.items).toEqual([]);
    const m = buildPaper(report, '  سؤال من  ', t, t, 'fa');
    expect(m.question).toBe('سؤال من');
    expect(m.stats.map((s) => s.label)).toEqual(['bmi', 'bp', 'fasting', 'cycle']);
    const headings = m.blocks.filter((b) => b.kind === 'heading').map((b) => (b.kind === 'heading' ? b.text : ''));
    expect(headings[0]).toBe('health');
    expect(headings.some((h) => h.startsWith('cycles('))).toBe(true);
    const pdf = paperPdfBlocks(m, 'question');
    expect(pdf[0]).toEqual({ kind: 'title', text: 'title' });
    expect(pdf.find((b) => b.kind === 'note')?.text).toBe('question سؤال من');
  });

  it('builds only the sections of a custom report', () => {
    const report = healthReportSchema.parse(steps('report_custom')[0]?.body.data);
    expect(report.basics).toBeUndefined();
    expect(report.vitals).toBeDefined();
    const m = buildPaper(report, null, t, t, 'en');
    expect(m.stats.map((s) => s.label)).toEqual(['bp', 'fasting', 'cycle']);
    expect(m.question).toBeNull();
  });

  it('parses the frozen shared report', () => {
    const flow = steps('share_flow');
    const shared = sharedReportSchema.parse(flow[4]?.body.data);
    expect(shared.question).toBe('آیا فشار شبانه‌ام نگران‌کننده است؟');
    expect(shared.expiresAt).toBe('2026-09-30T10:00:00+03:30');
    expect(shared.cycle).toBeUndefined();
  });
});
