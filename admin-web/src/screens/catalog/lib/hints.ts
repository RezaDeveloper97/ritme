/**
 * Per-group `meta` hints for the editor (catalog.md §4). Each epic that ships a group documents its
 * meta shape in its own doc and adds an entry here (message key `catalog.hint.<key>` + an example).
 * Examples are copied from the shipped seeds, so «insert example» gives a shape the app reads.
 * Unknown groups get the generic conventions only.
 */
export const HINT_KEYS = [
  'generic',
  'faq',
  'missedPill',
  'pelvicLevels',
  'alerts',
  'scoreItems',
  'tips',
  'menoScoreItems',
  'menoScoreBands',
  'menoAlerts',
  'menoTips',
  'menoCheckupGroups',
  'conditionPrograms',
  'painTypes',
  'painAssociated',
  'pmddItems',
  'conditionAlerts',
] as const;
export type HintKey = (typeof HINT_KEYS)[number];

export interface GroupHint {
  key: HintKey;
  /** A meta example the editor can insert (null: the group normally has no meta). */
  example: Record<string, unknown> | null;
}

const EXACT: Record<string, GroupHint> = {
  // 00017_contraception (CB-CONTRA-01): `methods` scopes a rule to pill types; `pack_week` marks the week-1 rule.
  missed_pill_rules: {
    key: 'missedPill',
    example: {
      methods: ['combined_pill'],
      missed: 1,
      severity: 'caution',
      steps: [
        { fa: 'قرص جاافتاده را همین حالا بخور، حتی اگر یعنی امروز دو قرص بخوری.', en: 'Take the missed pill now, even if it means taking two pills today.' },
        { fa: 'بقیه قرص‌ها را طبق معمول ادامه بده.', en: 'Carry on with the rest of the pack as usual.' },
      ],
    },
  },
  // 00010_pelvic_floor (CB-PELV-01).
  pelvic_levels: { key: 'pelvicLevels', example: { week_from: 1, hold_sec: 3, rest_sec: 3, reps: 10, sets: 3 } },

  // 00022_menopause (CB-MENO-01, docs/canvas-build/menopause.md §5).
  meno_score_items: {
    key: 'menoScoreItems',
    example: { domain: 'somatic', max: 4, log: ['symptoms.general.hot_flashes', 'symptoms.general.night_sweats'] },
  },
  meno_score_bands: { key: 'menoScoreBands', example: { min: 5, max: 8 } },
  meno_alerts: {
    key: 'menoAlerts',
    example: {
      severity: 'urgent',
      primary: true,
      stages: ['meno', 'post'],
      cta: { fa: 'این مورد را به پزشک بگو', en: 'Tell your doctor about this' },
    },
  },
  meno_tips: { key: 'menoTips', example: { placement: 'treatment_lifestyle', weekly_goal: 150, goal_unit: 'minutes' } },
  meno_checkup_groups: {
    key: 'menoCheckupGroups',
    example: { checkups: ['meno_blood_pressure', 'meno_blood_sugar', 'meno_lipids', 'meno_weight_waist'] },
  },

  // 00023_condition_programs (CB-COND-01, docs/canvas-build/catalog.md §5).
  condition_programs: {
    key: 'conditionPrograms',
    example: { logs: { fa: 'محل درد، شدت، نوع درد، اثر دارو', en: 'pain location, intensity, type of pain, painkiller effect' } },
  },
  pain_types: { key: 'painTypes', example: null },
  pain_associated: { key: 'painAssociated', example: { log: 'symptoms.digestive.bloating' } },
  pmdd_items: { key: 'pmddItems', example: null },
  condition_alerts: {
    key: 'conditionAlerts',
    example: {
      severity: 'urgent',
      hotlines: [
        { number: '1480', label: { fa: 'صدای مشاور', en: 'Counselling line' } },
        { number: '123', label: { fa: 'اورژانس اجتماعی', en: 'Social emergency' } },
      ],
    },
  },
};

/** Suffix conventions shared by many groups (`teen_faq`, `pelvic_alerts`, …). */
const SUFFIX: Array<[string, GroupHint]> = [
  ['_faq', { key: 'faq', example: null }],
  ['_alerts', { key: 'alerts', example: { severity: 'urgent', cta: { fa: 'تماس با پزشک', en: 'Call your doctor' } } }],
  ['_score_items', { key: 'scoreItems', example: { domain: 'somatic', max: 4 } }],
  ['_tips', { key: 'tips', example: null }],
];

export function hintFor(group: string): GroupHint {
  const exact = EXACT[group];
  if (exact) return exact;
  for (const [suffix, hint] of SUFFIX) if (group.endsWith(suffix)) return hint;
  return { key: 'generic', example: null };
}
