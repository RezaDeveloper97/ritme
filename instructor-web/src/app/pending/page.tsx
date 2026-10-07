'use client';

import { PendingScreen } from '@/screens/pending';
import { AccessGate } from '@/widgets/access-gate';

const ALLOW = ['pending'] as const;

export default function PendingPage() {
  return (
    <AccessGate allow={ALLOW}>
      {(instructor) => (instructor ? <PendingScreen instructor={instructor} /> : null)}
    </AccessGate>
  );
}
