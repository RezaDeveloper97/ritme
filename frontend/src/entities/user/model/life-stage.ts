/**
 * Life-stage modes (B-N2-01 backend, B-N2-03 switcher — Me → «مرحله زندگی»,
 * `nbl_Me_Mode`). The mode picks the tabs, the home and the log tiles. The list
 * order is the screen order of the artboard.
 */
export const LIFE_MODES = ['cycle', 'ttc', 'pregnancy', 'postpartum', 'menopause', 'teen'] as const;

export type LifeMode = (typeof LIFE_MODES)[number];

export function isLifeMode(value: unknown): value is LifeMode {
  return typeof value === 'string' && (LIFE_MODES as readonly string[]).includes(value);
}

/** `GET|PUT /profile/life-stage`. */
export interface LifeStage {
  /** The effective mode: an active pregnancy profile always wins (the pregnancy domain owns that switch). */
  mode: LifeMode;
  /** What the user picked; `null` = never chose (legacy derivation from `user_goal`). */
  storedMode: LifeMode | null;
  /** «درمان ناباروری (IVF/IUI) دارم» — the TTC card's switch. Nothing behind it yet (roadmap E03-ivf). */
  ivfIui: boolean;
  /** «روش پیشگیری را هم پیگیری کن» — nothing behind it yet (roadmap E06-contra). */
  trackContraception: boolean;
}

/** Partial `PUT /profile/life-stage` body. */
export interface LifeStageUpdate {
  mode?: LifeMode;
  ivfIui?: boolean;
  trackContraception?: boolean;
}

/**
 * Admin-edited texts of the calm pregnancy exit (`GET /profile/life-stage/loss-copy`).
 * A `null` text = no admin copy yet → the screen uses its bundled fallback.
 */
export interface LossCopy {
  title: string | null;
  body: string | null;
  confirm: string | null;
  cancel: string | null;
  doneTitle: string | null;
  doneBody: string | null;
  doneAction: string | null;
}
