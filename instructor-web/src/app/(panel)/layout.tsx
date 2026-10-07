'use client';

import type { ReactNode } from 'react';

import { AccessGate } from '@/widgets/access-gate';
import { PanelShell } from '@/widgets/panel-shell';

import { PanelInstructorProvider } from './panel-context';

const ALLOW = ['approved'] as const;

/** Every panel route: approved instructors only, inside the shell. */
export default function PanelLayout({ children }: { children: ReactNode }) {
  return (
    <AccessGate allow={ALLOW}>
      {(instructor) =>
        instructor ? (
          <PanelInstructorProvider value={instructor}>
            <PanelShell instructor={instructor}>{children}</PanelShell>
          </PanelInstructorProvider>
        ) : null
      }
    </AccessGate>
  );
}
