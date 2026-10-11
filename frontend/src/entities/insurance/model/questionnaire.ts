/** Health questionnaire (CB-INS-05). Shapes: openapi.yaml `InsuranceQuestionnaire`. Health data: never log it. */
export type QuestionnairePurpose = 'issue' | 'renewal';
export type QuestionnaireStatus =
  | 'draft'
  | 'confirmed'
  | 'submitted'
  | 'reviewing'
  | 'doctor_review'
  | 'approved'
  | 'conditional'
  | 'rejected';
/** Steps she records after confirming (server order). */
export type QuestionnaireStep = 'submitted' | 'reviewing' | 'doctor_review' | 'approved' | 'conditional' | 'rejected';
export type QuestionAnswer = 'yes' | 'no';

export interface QuestionnaireQuestion {
  code: string;
  section: string;
  answer: QuestionAnswer | null;
  fromRecord: boolean;
  /** Record items behind the suggestion (free text from her record, or a code such as `ivf_iui`). */
  evidence: string[];
}

export interface QuestionnaireEvent {
  status: string;
  /** yyyy-mm-dd */
  on: string;
}

export interface InsuranceQuestionnaire {
  id: number;
  purpose: QuestionnairePurpose;
  label: string | null;
  status: QuestionnaireStatus;
  prefilledCount: number;
  answeredCount: number;
  questionCount: number;
  complete: boolean;
  confirmedAt: string | null;
  timeline: QuestionnaireEvent[];
  /** Detail response only. */
  sections: string[];
  questions: QuestionnaireQuestion[];
  explanation: string;
}

export interface QuestionnaireUpdate {
  answers?: Record<string, QuestionAnswer | null>;
  explanation?: string | null;
}

export const QUESTIONNAIRE_STEPS: readonly QuestionnaireStep[] = [
  'submitted',
  'reviewing',
  'doctor_review',
  'approved',
  'conditional',
  'rejected',
];

const OUTCOMES: readonly string[] = ['approved', 'conditional', 'rejected'];

export type TimelineRowId = 'submitted' | 'reviewing' | 'doctor_review' | 'outcome';

export interface TimelineRow {
  id: TimelineRowId;
  /** For `outcome`: which result she recorded (null while pending). */
  status: 'approved' | 'conditional' | 'rejected' | null;
  on: string | null;
  state: 'done' | 'current' | 'todo';
}

/**
 * The status page's four rows (nbl_Ins_Status): submitted, first review, trusted-doctor review, result. A row is done
 * when she recorded it; the first one not done is the current step (none once a result is in).
 */
export function buildTimeline(events: readonly QuestionnaireEvent[]): TimelineRow[] {
  const find = (ids: readonly string[]) => [...events].reverse().find((e) => ids.includes(e.status)) ?? null;
  const spec: { id: TimelineRowId; match: readonly string[] }[] = [
    { id: 'submitted', match: ['submitted'] },
    { id: 'reviewing', match: ['reviewing'] },
    { id: 'doctor_review', match: ['doctor_review'] },
    { id: 'outcome', match: OUTCOMES },
  ];
  const rows: TimelineRow[] = spec.map((s) => {
    const e = find(s.match);
    const status = e && OUTCOMES.includes(e.status) ? (e.status as TimelineRow['status']) : null;
    return { id: s.id, status, on: e?.on ?? null, state: e ? 'done' : 'todo' };
  });
  // A later step recorded means the ones before it happened too, even if she skipped logging them.
  const last = rows.map((r) => r.state).lastIndexOf('done');
  for (let i = 0; i < last; i += 1) rows[i]!.state = 'done';
  const next = rows[last + 1];
  if (next) next.state = 'current';
  return rows;
}

export interface EvidenceText {
  /** Set for the codes the server sends (`cesarean`, `surgical_menopause`, `ivf_iui`); otherwise `text` is hers. */
  code: 'cesarean' | 'surgical_menopause' | 'ivf_iui' | null;
  text: string;
  /** yyyy-mm-dd when the item carried a date. */
  date: string | null;
}

const DATE = /^\d{4}-\d{2}-\d{2}$/;

/** Splits a record item (`title · 2024-03-01`, `cesarean · 2024-03-01`, `ivf_iui`) into label code / text / date. */
export function evidenceLabel(raw: string): EvidenceText {
  const at = raw.lastIndexOf(' · ');
  const tail = at >= 0 ? raw.slice(at + 3) : '';
  const head = at >= 0 && DATE.test(tail) ? raw.slice(0, at) : raw;
  const date = head === raw ? null : tail;
  if (head === 'cesarean' || head === 'surgical_menopause' || head === 'ivf_iui') return { code: head, text: head, date };
  return { code: null, text: head, date };
}

export const QUESTION_CODES = [
  'smoker',
  'disability',
  'hospitalized_5y',
  'chronic_disease',
  'pregnant',
  'gyn_surgery',
  'regular_medication',
  'infertility_treatment',
  'family_history',
] as const;
export type QuestionCode = (typeof QUESTION_CODES)[number];

export const SECTION_CODES = ['general', 'history', 'family', 'review'] as const;
export type SectionCode = (typeof SECTION_CODES)[number];

/** A question code the copy knows, else null (the server may add one before the app has its wording). */
export function questionCode(code: string): QuestionCode | null {
  return (QUESTION_CODES as readonly string[]).includes(code) ? (code as QuestionCode) : null;
}

/** A section code the copy knows, else `general`. */
export function sectionCode(code: string): SectionCode {
  return (SECTION_CODES as readonly string[]).includes(code) ? (code as SectionCode) : 'general';
}
