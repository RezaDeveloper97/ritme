'use client';

import { create } from 'zustand';
import { persist } from 'zustand/middleware';

import type { Locale } from '@/shared/i18n';
import { calendarSystem, convertParts, type DateParts } from '@/shared/lib/date';

import type {
  ChronicCondition,
  HeightUnit,
  BirthParts,
  PregnancyBasis,
  PregnancyIntention,
  WeightUnit,
} from './types';

interface OnboardingStore {
  phone: string;
  name: string;
  /** Calendar the stored date parts are expressed in — see `syncCalendar`. */
  locale: Locale;
  birth: BirthParts;
  weightUnit: WeightUnit;
  weight: number;
  heightUnit: HeightUnit;
  height: number;
  intention: PregnancyIntention | null;
  pregnancyBasis: PregnancyBasis;
  chronicConditions: ChronicCondition[];
  periodLen: number;
  cycleDuration: number;
  lastPeriod: DateParts | null;

  setPhone: (phone: string) => void;
  setName: (name: string) => void;
  setBirth: (birth: BirthParts) => void;
  setWeight: (weight: number) => void;
  setWeightUnit: (unit: WeightUnit) => void;
  setHeight: (height: number) => void;
  setHeightUnit: (unit: HeightUnit) => void;
  setIntention: (intention: PregnancyIntention) => void;
  setPregnancyBasis: (patch: Partial<PregnancyBasis>) => void;
  toggleCondition: (condition: ChronicCondition) => void;
  setPeriodLen: (len: number) => void;
  setCycleDuration: (len: number) => void;
  setLastPeriod: (date: DateParts) => void;
  /**
   * Re-express every stored date in `next`'s calendar. Onboarding answers are
   * persisted, so switching the language mid-flow would otherwise re-read a
   * Jalali 1373 as a Gregorian 1373. Idempotent — a no-op when the calendar is
   * already `next`'s, so pages can call it unconditionally on mount.
   */
  syncCalendar: (next: Locale) => void;
}

const emptyBasis: PregnancyBasis = {
  source: null,
  lmp: null,
  ultrasoundDate: null,
  ultrasoundWeeks: null,
  ultrasoundDays: null,
  manualWeeks: null,
  manualDays: null,
};

export const useOnboardingStore = create<OnboardingStore>()(
  persist(
    (set) => ({
      phone: '',
      name: '',
      // fa is the default locale (CLAUDE.md §6), so the seeded birthday below
      // is Jalali; `syncCalendar` converts it if the user starts in English.
      locale: 'fa',
      birth: { d: 25, m: 10, y: 1373 },
      weightUnit: 'kg',
      weight: 60,
      heightUnit: 'cm',
      height: 165,
      intention: null,
      pregnancyBasis: emptyBasis,
      chronicConditions: [],
      periodLen: 5,
      cycleDuration: 28,
      lastPeriod: null,

      setPhone: (phone) => set({ phone }),
      setName: (name) => set({ name }),
      setBirth: (birth) => set({ birth }),
      setWeight: (weight) => set({ weight }),
      setWeightUnit: (weightUnit) => set({ weightUnit }),
      setHeight: (height) => set({ height }),
      setHeightUnit: (heightUnit) => set({ heightUnit }),
      setIntention: (intention) => set({ intention }),
      setPregnancyBasis: (patch) =>
        set((s) => ({ pregnancyBasis: { ...s.pregnancyBasis, ...patch } })),
      toggleCondition: (condition) =>
        set((s) => ({
          chronicConditions: s.chronicConditions.includes(condition)
            ? s.chronicConditions.filter((c) => c !== condition)
            : [...s.chronicConditions, condition],
        })),
      setPeriodLen: (periodLen) => set({ periodLen }),
      setCycleDuration: (cycleDuration) => set({ cycleDuration }),
      setLastPeriod: (lastPeriod) => set({ lastPeriod }),

      syncCalendar: (next) =>
        set((s) => {
          if (calendarSystem(s.locale) === calendarSystem(next)) return {};
          const move = (parts: DateParts | null) =>
            parts ? convertParts(parts, s.locale, next) : null;
          const birth = convertParts({ year: s.birth.y, month: s.birth.m, day: s.birth.d }, s.locale, next);
          return {
            locale: next,
            birth: { y: birth.year, m: birth.month, d: birth.day },
            lastPeriod: move(s.lastPeriod),
            pregnancyBasis: {
              ...s.pregnancyBasis,
              lmp: move(s.pregnancyBasis.lmp),
              ultrasoundDate: move(s.pregnancyBasis.ultrasoundDate),
            },
          };
        }),
    }),
    {
      name: 'ritme-onboarding',
      // v0 stored `birth.m` as the raw 0-based wheel index while the API mapper
      // read it as a 1-based Jalali month, so every saved birthday was a month
      // early. v1 is 1-based; shift anything persisted under the old shape.
      // v2 records which calendar the stored dates are in; anything persisted
      // before it was necessarily Jalali, since fa was the only calendar.
      version: 2,
      migrate: (state, version) => {
        let s = state as OnboardingStore;
        if (version < 1 && s?.birth) {
          s = { ...s, birth: { ...s.birth, m: s.birth.m + 1 } };
        }
        if (version < 2) s = { ...s, locale: 'fa' };
        return s;
      },
    },
  ),
);
