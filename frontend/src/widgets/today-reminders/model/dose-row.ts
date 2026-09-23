import type { CareToday, MedicationForm, TodayDose } from '@/entities/care-reminder';
import type { IconName } from '@/shared/ui';

/*
 * Pure view-state for the «یادآورهای امروز» card — no React, no locale, so the
 * mapping from `/care/today` to what each row shows is unit-tested on its own.
 */

/** Which part of the day a `HH:MM` slot falls in (`care.slotPeriod.*`). */
export type SlotPeriod = 'morning' | 'noon' | 'evening' | 'night';

/** Tile tint: brand soft by default, teal for capsules (README "Colors"). */
export type DoseTone = 'brand' | 'teal';

export interface DoseRowState {
  /** Stable React key — one medication can have several slots a day. */
  key: string;
  reminderId: number;
  slot: string;
  title: string;
  taken: boolean;
  icon: IconName;
  tone: DoseTone;
  /** 12-hour clock for the slot, Latin digits (`08:00` → `8:00`, `21:30` → `9:30`). */
  clock: string;
  period: SlotPeriod;
  /** What tapping the check does next: tick an untaken dose, untick a taken one. */
  nextTaken: boolean;
}

const FORM_ICON: Record<MedicationForm, IconName> = {
  tablet: 'tablet',
  capsule: 'capsule',
  syrup: 'glass',
  injection: 'pill',
  drops: 'drop',
};

function parseSlot(slot: string): { hour: number; minute: number } {
  const [h, m] = slot.split(':');
  const hour = Number(h);
  const minute = Number(m);
  return {
    hour: Number.isFinite(hour) ? Math.min(23, Math.max(0, hour)) : 0,
    minute: Number.isFinite(minute) ? Math.min(59, Math.max(0, minute)) : 0,
  };
}

/** 05–11 morning, 12–14 noon, 15–18 evening, 19–04 night. */
export function slotPeriod(slot: string): SlotPeriod {
  const { hour } = parseSlot(slot);
  if (hour >= 5 && hour < 12) return 'morning';
  if (hour >= 12 && hour < 15) return 'noon';
  if (hour >= 15 && hour < 19) return 'evening';
  return 'night';
}

/** `HH:MM` → 12-hour `h:MM` (the period word carries am/pm, as in the artboard). */
export function slotClock(slot: string): string {
  const { hour, minute } = parseSlot(slot);
  const h12 = hour % 12 === 0 ? 12 : hour % 12;
  return `${h12}:${String(minute).padStart(2, '0')}`;
}

export function doseRowState(dose: TodayDose): DoseRowState {
  return {
    key: `${dose.reminderId}-${dose.slot}`,
    reminderId: dose.reminderId,
    slot: dose.slot,
    title: dose.title,
    taken: dose.taken,
    icon: FORM_ICON[dose.form] ?? 'pill',
    tone: dose.form === 'capsule' ? 'teal' : 'brand',
    clock: slotClock(dose.slot),
    period: slotPeriod(dose.slot),
    nextTaken: !dose.taken,
  };
}

export type CardStatus = 'loading' | 'hidden' | 'empty' | 'ready';

/**
 * Which face the card shows. An error hides the card entirely (the home feed
 * closes up rather than showing a broken block); a day with no doses and no
 * upcoming appointment gets the short prompt + add button.
 */
export function cardStatus(query: {
  isLoading: boolean;
  isError: boolean;
  data: CareToday | undefined;
}): CardStatus {
  if (query.isError) return 'hidden';
  if (query.isLoading) return 'loading';
  if (!query.data) return 'hidden';
  if (query.data.doses.length === 0 && query.data.nextAppointment === null) return 'empty';
  return 'ready';
}
