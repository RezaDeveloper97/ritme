/**
 * Per-group `meta` hints for the editor (catalog.md §4). Each epic that ships a group documents its
 * meta shape in its own doc and adds an entry here (message key `catalog.hint.<key>` + an example).
 * Unknown groups get the generic conventions only.
 */
export type HintKey = 'generic' | 'faq' | 'missedPill' | 'pelvicLevels' | 'alerts' | 'scoreItems' | 'tips';

export interface GroupHint {
  key: HintKey;
  /** A meta example the editor can insert (null: the group normally has no meta). */
  example: Record<string, unknown> | null;
}

const EXACT: Record<string, GroupHint> = {
  missed_pill_rules: {
    key: 'missedPill',
    example: {
      missed: 1,
      severity: 'caution',
      steps: [
        { fa: 'قرص فراموش‌شده را همین حالا بخورید.', en: 'Take the missed pill now.' },
        { fa: 'بقیه بسته را طبق معمول ادامه دهید.', en: 'Continue the pack as usual.' },
      ],
    },
  },
  pelvic_levels: { key: 'pelvicLevels', example: { week_from: 1, hold_sec: 3, rest_sec: 3, reps: 10, sets: 3 } },
};

/** Suffix conventions shared by many groups (`teen_faq`, `meno_alerts`, `meno_tips`, …). */
const SUFFIX: Array<[string, GroupHint]> = [
  ['_faq', { key: 'faq', example: null }],
  ['_alerts', { key: 'alerts', example: { severity: 'urgent', cta: { fa: 'تماس با پزشک', en: 'Call your doctor' } } }],
  ['_score_items', { key: 'scoreItems', example: { score_max: 3, domain: 'vasomotor' } }],
  ['_tips', { key: 'tips', example: null }],
];

export function hintFor(group: string): GroupHint {
  const exact = EXACT[group];
  if (exact) return exact;
  for (const [suffix, hint] of SUFFIX) if (group.endsWith(suffix)) return hint;
  return { key: 'generic', example: null };
}
