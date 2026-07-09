'use client';

import { create } from 'zustand';
import { persist } from 'zustand/middleware';

import type { JalaliParts } from '@/shared/lib/date';

import type { Gender, HeightUnit, JalaliBirth, WeightUnit } from './types';

interface OnboardingStore {
  phone: string;
  name: string;
  gender: Gender | null;
  birth: JalaliBirth;
  weightUnit: WeightUnit;
  weight: number;
  heightUnit: HeightUnit;
  height: number;
  periodLen: number;
  cycleDuration: number;
  lastPeriod: JalaliParts | null;

  setPhone: (phone: string) => void;
  setName: (name: string) => void;
  setGender: (gender: Gender) => void;
  setBirth: (birth: JalaliBirth) => void;
  setWeight: (weight: number) => void;
  setWeightUnit: (unit: WeightUnit) => void;
  setHeight: (height: number) => void;
  setHeightUnit: (unit: HeightUnit) => void;
  setPeriodLen: (len: number) => void;
  setCycleDuration: (len: number) => void;
  setLastPeriod: (date: JalaliParts) => void;
}

export const useOnboardingStore = create<OnboardingStore>()(
  persist(
    (set) => ({
      phone: '',
      name: '',
      gender: null,
      birth: { d: 25, m: 9, y: 1373 },
      weightUnit: 'kg',
      weight: 60,
      heightUnit: 'cm',
      height: 165,
      periodLen: 5,
      cycleDuration: 28,
      lastPeriod: null,

      setPhone: (phone) => set({ phone }),
      setName: (name) => set({ name }),
      setGender: (gender) => set({ gender }),
      setBirth: (birth) => set({ birth }),
      setWeight: (weight) => set({ weight }),
      setWeightUnit: (weightUnit) => set({ weightUnit }),
      setHeight: (height) => set({ height }),
      setHeightUnit: (heightUnit) => set({ heightUnit }),
      setPeriodLen: (periodLen) => set({ periodLen }),
      setCycleDuration: (cycleDuration) => set({ cycleDuration }),
      setLastPeriod: (lastPeriod) => set({ lastPeriod }),
    }),
    { name: 'ritme-onboarding' },
  ),
);
