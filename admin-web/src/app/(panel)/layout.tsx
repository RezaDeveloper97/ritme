import { Suspense, type ReactNode } from 'react';

import { AppShell } from '@/widgets/shell';

/** Every signed-in screen: auth gate + sidebar + header. */
export default function PanelLayout({ children }: { children: ReactNode }) {
  return (
    <Suspense>
      <AppShell>{children}</AppShell>
    </Suspense>
  );
}
