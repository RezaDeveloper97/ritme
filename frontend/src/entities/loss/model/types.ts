/*
 * Pregnancy loss path (CB-LOSS-01, backend-go/internal/loss) as the screens read
 * it — camel-cased from `GET /api/v1/loss`. Codes and dates only: every label is
 * catalog content (`loss_*` groups, admin-editable, [needs clinical review]).
 *
 * Privacy (CLAUDE.md §11): this is among the most sensitive data in the app.
 * Never log it, put it in a URL, or send it anywhere but the loss endpoints.
 */

/** «چه اتفاقی افتاد؟» — catalog `loss_types`, board order; `unspecified` = «ترجیح می‌دهم نگویم». */
export const LOSS_TYPES = ['early_miscarriage', 'late_miscarriage', 'ectopic', 'chemical', 'unspecified'] as const;
export type LossType = (typeof LOSS_TYPES)[number];

/** «امروز چطوری؟» — catalog `loss_moods`. */
export const LOSS_MOODS = ['sad', 'numb', 'angry', 'a_bit_better'] as const;
export type LossMood = (typeof LOSS_MOODS)[number];

/** «از اینجا به بعد» — catalog `loss_next_steps`. */
export const LOSS_NEXT_STEPS = ['cycle', 'ttc', 'nothing'] as const;
export type LossNextStep = (typeof LOSS_NEXT_STEPS)[number];

/** The scheduled care appointment a follow-up row is linked to (only the parts the screen shows). */
export interface LossAppointment {
  id: number;
  /** Tehran wall clock `Y-m-d H:i:s`. */
  scheduledAt: string;
}

export interface LossEvent {
  id: number;
  type: LossType;
  /** `Y-m-d`, approximate; null when she didn't say. */
  occurredOn: string | null;
  notifyCompanion: boolean;
  companionNotified: boolean;
  contentStopped: boolean;
  nextStep: LossNextStep | null;
  /** ISO-8601 with offset. */
  createdAt: string | null;
}

export interface LossFollowup {
  bleeding: { stopped: boolean; stoppedOn: string | null; today: string | null };
  beta: { negative: boolean; negativeOn: string | null; nextOn: string | null; appointment: LossAppointment | null };
  visit: { suggestedOn: string | null; appointment: LossAppointment | null };
}

export interface LossState {
  /** null = no loss recorded (every other part is then empty). */
  loss: LossEvent | null;
  lossesCount: number;
  /** ≥ 2 recorded losses → the «سقط دوم یا سوم» note. */
  recurrentHint: boolean;
  followup: LossFollowup | null;
  mood: { today: LossMood | null; recent: { date: string; mood: LossMood }[] } | null;
  note: { hasNote: boolean; updatedAt: string | null } | null;
}

export interface LossNote {
  note: string | null;
  updatedAt: string | null;
}

/** POST /loss — every answer optional (an absent type = unspecified). */
export interface RecordLossInput {
  type?: LossType | null;
  occurredOn?: string | null;
  notifyCompanion?: boolean;
}

/** PUT /loss/followup — only the given keys change; null clears. */
export interface LossFollowupInput {
  bleedingStopped?: boolean;
  betaNextOn?: string | null;
  betaNegative?: boolean;
  /** Tehran wall clock `Y-m-d H:i`. */
  visitAt?: string | null;
}

/** One localized catalog item of a `loss_*` group. */
export interface LossCatalogItem {
  code: string;
  title: string | null;
  body: string | null;
  meta: Record<string, unknown> | null;
}

/** The loss copy groups the screens read (GET /catalog/:group). */
export const LOSS_CATALOG_GROUPS = [
  'loss_types',
  'loss_warning_signs',
  'loss_hotlines',
  'loss_followups',
  'loss_moods',
  'loss_support',
  'loss_next_steps',
] as const;
export type LossCatalogGroup = (typeof LOSS_CATALOG_GROUPS)[number];

/** A dialled number with its catalog label («صدای مشاور», «۱۴۸۰»). */
export interface LossHotline {
  number: string;
  label: string | null;
}
