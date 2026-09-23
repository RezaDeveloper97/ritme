import type { ReactNode } from 'react';

import { cn } from '@/shared/lib';

export type BadgeTone = 'neutral' | 'brand' | 'green' | 'red' | 'amber' | 'data';

export function Badge({ tone = 'neutral', children }: { tone?: BadgeTone; children: ReactNode }) {
  return <span className={cn('badge', tone !== 'neutral' && `badge-${tone}`)}>{children}</span>;
}
