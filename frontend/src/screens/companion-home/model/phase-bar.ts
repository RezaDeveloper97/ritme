/**
 * The partner card's cycle bar (nbl_Hamdam_Home): period · follicular ·
 * fertile · luteal segments of her current cycle and today's marker. The
 * companion view carries only day and length, so without engine numbers the
 * segments are the textbook split (period 5 days, ovulation 14 days before the
 * next period). When the caller has her engine period length and ovulation day
 * they win, the fertile window is O−5 … O (never later, never before the period
 * ends), and `fertilitySuppressed` drops the fertile segment altogether. It is
 * an orientation aid, not her personal prediction (that stays on her side).
 */
export interface PhaseBar {
  /** Segment lengths in days, in order period, follicular, fertile, luteal. */
  segments: { key: 'period' | 'follicular' | 'fertile' | 'luteal'; days: number }[];
  /** Today's position along the bar, 0–100 (start of the day). */
  markerPercent: number;
}

/** Engine numbers for her cycle; every field is optional (Laravel payloads have none). */
export interface PhaseBarEngine {
  /** Her effective period length in days. */
  periodLength?: number | null;
  /** 1-based cycle day of the estimated ovulation; null/absent = length − 14. */
  ovulationDay?: number | null;
  /** Clinical suppression: no ovulation, no fertile segment. */
  fertilitySuppressed?: boolean;
}

const FALLBACK_PERIOD_DAYS = 5;
const LUTEAL_DAYS = 14;
/** Days before ovulation the fertile window opens. */
const FERTILE_LEAD_DAYS = 5;

export function phaseBar(cycleDay: number, cycleLength: number, engine: PhaseBarEngine = {}): PhaseBar | null {
  if (!Number.isFinite(cycleDay) || !Number.isFinite(cycleLength) || cycleLength < 20 || cycleDay < 1) return null;
  const length = Math.round(cycleLength);
  const day = Math.min(Math.round(cycleDay), length);
  const markerPercent = Math.round(((day - 0.5) / length) * 1000) / 10;
  const wanted = engine.periodLength != null && engine.periodLength >= 1 ? Math.round(engine.periodLength) : FALLBACK_PERIOD_DAYS;

  if (engine.fertilitySuppressed) {
    const period = Math.min(wanted, length - 1);
    return {
      segments: [
        { key: 'period', days: period },
        { key: 'follicular', days: length - period },
        { key: 'fertile', days: 0 },
        { key: 'luteal', days: 0 },
      ],
      markerPercent,
    };
  }

  const ovulation = Math.min(
    length,
    Math.max(2, engine.ovulationDay != null && engine.ovulationDay >= 1 ? Math.round(engine.ovulationDay) : length - LUTEAL_DAYS),
  ); // 1-based day
  const period = Math.min(wanted, ovulation - 1);
  const fertileStart = Math.max(period + 1, ovulation - FERTILE_LEAD_DAYS);
  const follicular = fertileStart - 1 - period;
  const fertile = ovulation - fertileStart + 1;
  const luteal = length - ovulation;
  return {
    segments: [
      { key: 'period', days: period },
      { key: 'follicular', days: follicular },
      { key: 'fertile', days: fertile },
      { key: 'luteal', days: luteal },
    ],
    markerPercent,
  };
}
