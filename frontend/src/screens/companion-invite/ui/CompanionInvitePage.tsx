'use client';

import { useState } from 'react';

import { PLUS_COMPANION, usePlusLocked } from '@/entities/plus';
import { InviteCompanionFlow } from '@/features/invite-companion';
import { useRouter } from '@/shared/i18n';
import { PlusRequiredSheet } from '@/widgets/plus-required-sheet';

/**
 * `/companions/new` — the «ریتمی همراه» invite wizard. Making the invite code is a Ritme Plus feature
 * (`plus.companion`): a free user walks the whole wizard and gets the Plus sheet with its offer on the last CTA.
 */
export function CompanionInvitePage() {
  const router = useRouter();
  const plusLocked = usePlusLocked(PLUS_COMPANION);
  const [plusOpen, setPlusOpen] = useState(false);
  return (
    <>
      <InviteCompanionFlow
        onExit={() => router.push('/companions')}
        plusLocked={plusLocked}
        onPlusRequired={() => setPlusOpen(true)}
      />
      <PlusRequiredSheet feature="companion" open={plusOpen} onClose={() => setPlusOpen(false)} />
    </>
  );
}
