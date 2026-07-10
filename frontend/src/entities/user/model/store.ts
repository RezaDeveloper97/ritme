'use client';

import { create } from 'zustand';
import { persist } from 'zustand/middleware';

import type { JalaliParts } from '@/shared/lib/date';

import type {
  ChronicCondition,
  HeightUnit,
  JalaliBirth,
  PregnancyBasis,
  PregnancyIntention,
  WeightUnit,
} from './types';

interface OnboardingStore {
  phone: string;
  name: string;
  birth: JalaliBirth;
  weightUnit: WeightUnit;
  weight: number;
  heightUnit: HeightUnit;
  height: number;
  intention: PregnancyIntention | null;
  pregnancyBasis: PregnancyBasis;
  chronicConditions: ChronicCondition[];
  periodLen: number;
  cycleDuration: number;
  lastPeriod: JalaliParts | null;

  setPhone: (phone: string) => void;
  setName: (name: string) => void;
  setBirth: (birth: JalaliBirth) => void;
  setWeight: (weight: number) => void;
  setWeightUnit: (unit: WeightUnit) => void;
  setHeight: (height: number) => void;
  setHeightUnit: (unit: HeightUnit) => void;
  setIntention: (intention: PregnancyIntention) => void;
  setPregnancyBasis: (patch: Partial<PregnancyBasis>) => void;
  toggleCondition: (condition: ChronicCondition) => void;
  setPeriodLen: (len: number) => void;
  setCycleDuration: (len: number) => void;
  setLastPeriod: (date: JalaliParts) => void;
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
      birth: { d: 25, m: 9, y: 1373 },
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
    }),
    { name: 'ritme-onboarding' },
  ),
);
