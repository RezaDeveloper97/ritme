'use client';

import { createContext, useContext } from 'react';

import type { Instructor } from '@/entities/instructor';

// The approved instructor resolved by the panel layout's gate, for its pages.
const PanelInstructorContext = createContext<Instructor | null>(null);

export const PanelInstructorProvider = PanelInstructorContext.Provider;

export function usePanelInstructor(): Instructor {
  const instructor = useContext(PanelInstructorContext);
  if (!instructor) throw new Error('usePanelInstructor outside the panel layout');
  return instructor;
}
