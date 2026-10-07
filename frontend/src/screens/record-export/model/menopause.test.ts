import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

import { describe, expect, it } from 'vitest';

import { healthReportSchema, menopauseSectionSchema } from '@/entities/health-record';

import { buildMenopauseModel, menopausePaperBlocks } from './menopause';
import { buildPaper, type Translate } from './paper';

/* The menopause doctor report (CB-MENO-11) against the Go-recorded goldens of CB-MENO-03. */
const GOLDEN = resolve(process.cwd(), '../backend-go/contract/golden/menopause');
const steps = (name: string) =>
  (JSON.parse(readFileSync(resolve(GOLDEN, `${name}.json`), 'utf8')) as { steps: { body: { data?: unknown } }[] }).steps;

const t: Translate = (key, values) => (values ? `${key}(${Object.values(values).join(',')})` : key);

describe('menopause doctor report', () => {
  it('builds the board rows from GET /menopause/report', () => {
    const data = steps('report_flow').at(-1)?.body.data as { report: unknown };
    const section = menopauseSectionSchema.parse(data.report);
    const m = buildMenopauseModel(section, t, 'en');
    expect(m.summary.map((r) => r.key)).toEqual(['stage', 'score', 'hotFlashes', 'nightSweats', 'sleep', 'bleeding', 'bp']);
    expect(m.summary[0]?.value).toBe('stageValue(stage.meno,14)');
    expect(m.summary[1]?.value).toBe('scoreValue(14,44)');
    expect(m.summary[6]?.value).toBe('none');
    expect(m.symptoms).toEqual([{ key: 'symptoms.general.hot_flashes', label: 'Hot flashes', percent: 25, share: 'share(25)' }]);
    expect(m.treatment.map((r) => r.label)).toEqual(['rows.hrt', 'rows.sideEffects']);
    expect(m.treatment[0]?.value).toContain('Estrogen gel');
    expect(m.treatment[0]?.value).toContain('adherence(25)');
  });

  it('reads the builder section and draws it on the paper', () => {
    const report = healthReportSchema.parse(steps('report_builder_section').at(-1)?.body.data);
    expect(report.menopause?.data.window.to).toBe('2026-09-23');
    const paper = buildPaper(report, null, t, t, 'en', t);
    const headings = paper.blocks.filter((b) => b.kind === 'heading').map((b) => (b.kind === 'heading' ? b.text : ''));
    expect(headings).toContain('title');
    // Without the menopause translator (older callers) the section is left out rather than half-drawn.
    const plain = buildPaper(report, null, t, t, 'en');
    expect(plain.blocks.length).toBeLessThan(paper.blocks.length);
  });

  it('passes the lifestyle goal unit so a minutes goal is not read as sessions (CB-MENO-13)', () => {
    const data = steps('report_flow').at(-1)?.body.data as { report: { treatment: { lifestyle: unknown[] } } };
    const lifestyle = [
      { name: 'Brisk walk', weekly_goal: 150, goal_unit: 'minutes', per_week: 90 },
      { name: 'Strength', weekly_goal: 2, goal_unit: 'sessions', per_week: 1.5 },
      { name: 'Yoga', weekly_goal: null, goal_unit: null, per_week: 1 },
    ];
    const section = menopauseSectionSchema.parse({ ...data.report, treatment: { ...data.report.treatment, lifestyle } });
    const rows = buildMenopauseModel(section, t, 'en').treatment.filter((r) => r.label === 'rows.lifestyle');
    expect(rows.map((r) => r.value)).toEqual([
      'lifestyleValue(Brisk walk,90,minutes)',
      'lifestyleValue(Strength,1.5,sessions)',
      'lifestyleValue(Yoga,1,sessions)',
    ]);
  });

  it('draws an empty section as one muted line', () => {
    const data = steps('report_flow').at(-1)?.body.data as { report: unknown };
    const section = menopauseSectionSchema.parse(data.report);
    expect(menopausePaperBlocks(section, true, t, 'en')).toEqual([
      { kind: 'heading', text: 'title' },
      { kind: 'empty', text: 'empty' },
    ]);
  });
});
