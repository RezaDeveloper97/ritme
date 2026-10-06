import { StatusPill } from '@/shared/ui';

import { stateTone } from '../model/range';
import type { MarkerState } from '../model/types';

/** «پایین» / «طبیعی» … — the server's localized state label in the state's tone. */
export function MarkerStatePill({ state, label, className }: { state: MarkerState; label: string; className?: string }) {
  if (!label) return null;
  return (
    <StatusPill tone={stateTone(state)} className={className}>
      {label}
    </StatusPill>
  );
}
