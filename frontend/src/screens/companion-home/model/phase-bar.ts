/**
 * The partner card's cycle bar (nbl_Hamdam_Home): period · follicular ·
 * fertile · luteal segments of her current cycle and today's marker. The
 * companion view carries only day, length and phase, so the segments are the
 * textbook split — period 5 days, ovulation 14 days before the next period,
 * fertile = the 5 days up to and including ovulation, luteal after it. It is
 * an orientation aid, not her personal prediction (that stays on her side).
 */
export interface PhaseBar {
  /** Segment lengths in days, in order period, follicular, fertile, luteal. */
  segments: { key: 'period' | 'follicular' | 'fertile' | 'luteal'; days: number }[];
  /** Today's position along the bar, 0–100 (start of the day). */
  markerPercent: number;
}

const PERIOD_DAYS = 5;
const LUTEAL_DAYS = 14;
const FERTILE_DAYS = 5;

export function phaseBar(cycleDay: number, cycleLength: number): PhaseBar | null {
  if (!Number.isFinite(cycleDay) || !Number.isFinite(cycleLength) || cycleLength < 20 || cycleDay < 1) return null;
  const length = Math.round(cycleLength);
  const ovulation = length - LUTEAL_DAYS; // 1-based day
  const period = Math.min(PERIOD_DAYS, ovulation - FERTILE_DAYS);
  const fertileStart = Math.max(period + 1, ovulation - FERTILE_DAYS + 1);
  const follicular = Math.max(0, fertileStart - 1 - period);
  const fertile = ovulation - fertileStart + 1;
  const luteal = length - ovulation;
  const day = Math.min(Math.round(cycleDay), length);
  return {
    segments: [
      { key: 'period', days: period },
      { key: 'follicular', days: follicular },
      { key: 'fertile', days: fertile },
      { key: 'luteal', days: luteal },
    ],
    markerPercent: Math.round(((day - 0.5) / length) * 1000) / 10,
  };
}
