import type { CycleTodayUi } from '@/entities/cycle';

import { isEnginePayload } from '@/widgets/cycle-predictions';

import { heroState, NEAR_PERIOD_DAYS, type HeroInput, type HeroState } from './hero-state';

/** The one banner the Today screen can show (§2.43, E-17). */
export type TodayBannerKind = 'end_missing' | 'overdue' | 'ask_last_period';

/** Lateness tiers (SPEC §2.22): 1–7 calm, 8–14 overdue, more than 14 ask for the last period. */
export type LatenessTier = 'none' | 'calm' | 'overdue' | 'ask';

export const OVERDUE_FROM_DAYS = 8;
export const ASK_FROM_DAYS = 15;

export function latenessTier(daysLate: number | null | undefined): LatenessTier {
  if (daysLate == null || daysLate <= 0) return 'none';
  if (daysLate < OVERDUE_FROM_DAYS) return 'calm';
  if (daysLate < ASK_FROM_DAYS) return 'overdue';
  return 'ask';
}

export interface TodayHero extends HeroState {
  /** At most one banner, already in the engine's priority order; null off today. */
  banner: TodayBannerKind | null;
  /** «ثبت تست بارداری» on the overdue banner. */
  showPregnancyTest: boolean;
  tier: LatenessTier;
  /** The decision came from `cycle_view.today_ui` (v6), not the legacy `hero-state` rules. */
  fromEngine: boolean;
}

export interface TodayHeroInput extends HeroInput {
  todayUi: CycleTodayUi | null | undefined;
}

const BANNERS: readonly string[] = ['end_missing', 'overdue', 'ask_last_period'];

// Lives with the predictions widget (it decides which dates the card shows); re-exported for the home.
export { isEnginePayload };

/**
 * With `today_ui` the ring action, the banner and the test button come from the engine; «هنوز نه» /
 * «هنوز ادامه دارد» stay local for the day. Without it (Laravel payload) the old `heroState` rules apply
 * unchanged.
 */
export function todayHero(i: TodayHeroInput): TodayHero {
  const tier = latenessTier(i.daysLate);
  const ui = i.todayUi;
  if (!ui) {
    return { ...heroState(i), banner: null, showPregnancyTest: false, tier, fromEngine: false };
  }

  const action = ui.ringAction;
  const during = i.inPeriod || action === 'end';
  const endPrompt = during && i.isToday && action === 'end' && !i.endDismissed;
  const startPrompt = !during && i.isToday && action === 'start' && !i.startDismissed;

  let variant: HeroState['variant'] = 'normal';
  if (during) variant = 'during';
  else if (i.daysLate != null && i.daysLate > 0) variant = 'late';
  else if (i.daysLeft != null && i.daysLeft >= 0 && i.daysLeft <= NEAR_PERIOD_DAYS) variant = 'near';

  let banner: TodayBannerKind | null = null;
  if (i.isToday && ui.banner && BANNERS.includes(ui.banner)) {
    banner = ui.banner as TodayBannerKind;
    // «هنوز ادامه دارد» silences the missing-end banner for the day, as before.
    if (banner === 'end_missing' && i.endDismissed) banner = null;
  }

  return {
    variant,
    startPrompt,
    endPrompt,
    endMissing: banner === 'end_missing',
    banner,
    showPregnancyTest: banner === 'overdue' && ui.showPregnancyTestButton,
    tier,
    fromEngine: true,
  };
}

/** What a banner button does; `TodayBanner` maps each to its handler. */
export type BannerAction = 'edit_end' | 'log_pregnancy_test' | 'edit_last_period';

/** The buttons a banner shows, in order. */
export function bannerActions(banner: TodayBannerKind, tier: LatenessTier, showPregnancyTest: boolean): BannerAction[] {
  if (banner === 'end_missing') return ['edit_end'];
  if (banner === 'ask_last_period') return ['edit_last_period'];
  const out: BannerAction[] = [];
  if (showPregnancyTest) out.push('log_pregnancy_test');
  if (tier === 'ask') out.push('edit_last_period');
  return out;
}
