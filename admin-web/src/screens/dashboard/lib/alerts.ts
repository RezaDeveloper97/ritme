/** View-models of the dashboard alerts (T-M10-34, SPEC v6 §5.1, D-180). Pure: no React, no fetching. */

export type AlertRuleKey = 'ocr_correction_rate' | 'message_ctr' | 'rui_users' | 'engine_mae' | 'safety_selftest_failed';

export interface AlertInput {
  key: string;
  subject: string;
  state: string;
  severity: string;
  text: string;
  day: string;
  value: number;
  threshold: number;
  opened_at: string;
  resolved_at?: string | null;
}

export type AlertUnit = 'pct' | 'days' | 'count';

/** The unit a rule's value and threshold are stored in: OCR and CTR-drop are fractions, the others plain numbers. */
export function ruleUnit(key: string): AlertUnit {
  switch (key) {
    case 'ocr_correction_rate':
    case 'message_ctr':
      return 'pct';
    case 'rui_users':
    case 'engine_mae':
      return 'days';
    default:
      return 'count';
  }
}

/** The number shown to an admin: fractions become percent, everything else is untouched. */
export function toDisplay(key: string, stored: number): number {
  return ruleUnit(key) === 'pct' ? round(stored * 100, 2) : round(stored, 2);
}

/** The inverse of toDisplay, for the rule editor. */
export function fromDisplay(key: string, shown: number): number {
  return ruleUnit(key) === 'pct' ? round(shown / 100, 4) : round(shown, 4);
}

function round(v: number, digits: number): number {
  const f = 10 ** digits;
  return Math.round(v * f) / f;
}

/** The screen an alert points to; a CTR alert opens the message rule behind its subject («r{rule}v{version}»). */
export function sourceHref(key: string, subject: string): string {
  switch (key) {
    case 'message_ctr': {
      const ref = parseSubject(subject);
      return ref ? `/messages/rules/${ref.rule}` : '/messages/rules';
    }
    case 'safety_selftest_failed':
      return '/safety';
    default:
      return '/kpi';
  }
}

/** «r{rule}v{version}» → the message rule and version a CTR alert is about. */
export function parseSubject(subject: string): { rule: number; version: number } | null {
  const m = /^r(\d+)v(\d+)$/.exec(subject);
  return m ? { rule: Number(m[1]), version: Number(m[2]) } : null;
}

export interface AlertRow {
  id: string;
  key: string;
  subject: string;
  ref: { rule: number; version: number } | null;
  critical: boolean;
  resolved: boolean;
  value: number;
  threshold: number;
  unit: AlertUnit;
  openedAt: string;
  resolvedAt: string | null;
  href: string;
}

export function alertRow(a: AlertInput): AlertRow {
  return {
    id: `${a.key}:${a.subject}:${a.opened_at}`,
    key: a.key,
    subject: a.subject,
    ref: parseSubject(a.subject),
    critical: a.severity === 'critical',
    resolved: a.state === 'resolved',
    value: toDisplay(a.key, a.value),
    threshold: toDisplay(a.key, a.threshold),
    unit: ruleUnit(a.key),
    openedAt: a.opened_at,
    resolvedAt: a.resolved_at ?? null,
    href: sourceHref(a.key, a.subject),
  };
}

/** Open alerts first (critical before warning, then newest), as the panel lists them. */
export function alertRows(open: AlertInput[], resolved: AlertInput[] = []): { open: AlertRow[]; resolved: AlertRow[] } {
  const newest = (a: AlertRow, b: AlertRow) => (a.openedAt < b.openedAt ? 1 : a.openedAt > b.openedAt ? -1 : 0);
  return {
    open: open.map(alertRow).sort((a, b) => Number(b.critical) - Number(a.critical) || newest(a, b)),
    resolved: resolved.map(alertRow).sort(newest),
  };
}
