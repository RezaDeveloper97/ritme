'use client';

import { ApplyScreen } from '@/screens/apply';
import { AccessGate } from '@/widgets/access-gate';

const ALLOW = ['required', 'revoked'] as const;

export default function ApplyPage() {
  return (
    <AccessGate allow={ALLOW}>
      {(instructor, access) => <ApplyScreen instructor={instructor} access={access} />}
    </AccessGate>
  );
}
