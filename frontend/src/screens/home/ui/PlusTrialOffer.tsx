'use client';

import { useState } from 'react';

import { PlusTrialBanner } from '@/widgets/plus-trial-banner';
import { PlusTrialSheet } from '@/widgets/plus-trial-sheet';

/**
 * The Plus trial offer on a home (B-N2-08): the floating countdown banner and
 * the sheet it opens. Render it as a direct child of `.view` (the banner floats
 * above the bottom nav and the scroll tail clears it). Never for teen
 * (QUESTIONS #70/#72) — the caller decides.
 */
export function PlusTrialOffer() {
  const [open, setOpen] = useState(false);
  return (
    <>
      <PlusTrialBanner onOpen={() => setOpen(true)} />
      <PlusTrialSheet open={open} onClose={() => setOpen(false)} />
    </>
  );
}
