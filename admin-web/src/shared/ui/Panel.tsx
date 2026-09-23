import type { ReactNode } from 'react';

import { cn } from '@/shared/lib';

/** The one container for a block of admin content (list, form section, figures). */
export function Panel({
  title,
  actions,
  children,
  className,
  bodyClassName = 'p-4 sm:p-5',
}: {
  title?: ReactNode;
  actions?: ReactNode;
  children: ReactNode;
  className?: string;
  /** `''` for edge-to-edge content (tables). */
  bodyClassName?: string;
}) {
  return (
    <section className={cn('panel overflow-clip', className)}>
      {title || actions ? (
        <header className="panel-head">
          {title ? <h2 className="panel-title">{title}</h2> : null}
          <div className="flex-1" />
          {actions}
        </header>
      ) : null}
      <div className={bodyClassName}>{children}</div>
    </section>
  );
}
