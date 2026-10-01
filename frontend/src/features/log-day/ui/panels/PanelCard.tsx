import type { ReactNode } from 'react';

import { IconCircle, type IconName, type Tone } from '@/shared/ui';

/** A titled card of a log detail panel: tone disc + 16/800 title, then the controls. */
export function PanelCard({
  title,
  icon,
  tone,
  children,
}: {
  title: ReactNode;
  icon: IconName;
  tone: Tone;
  children: ReactNode;
}) {
  return (
    <section className="nb-card lday-pcard">
      <h3 className="lday-pcard-h">
        <IconCircle icon={icon} tone={tone} size="sm" />
        <span>{title}</span>
      </h3>
      {children}
    </section>
  );
}
