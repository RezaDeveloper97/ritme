import Link from 'next/link';
import type { ReactNode } from 'react';

import { Icon } from './Icon';

/**
 * Top of a form/detail screen: a link back to the list (never history.back —
 * frontend/CLAUDE.md §4.2), the record's title and its page-level actions.
 */
export function PageHeader({
  title,
  backHref,
  backLabel,
  meta,
  actions,
}: {
  title: ReactNode;
  backHref?: string;
  backLabel?: string;
  meta?: ReactNode;
  actions?: ReactNode;
}) {
  return (
    <div className="page-head">
      <div className="flex min-w-0 flex-col gap-1">
        {backHref ? (
          <Link href={backHref} className="back-link">
            <Icon name="chevronRight" size={15} className="-scale-x-100 rtl:scale-x-100" />
            {backLabel}
          </Link>
        ) : null}
        <h2 className="page-title">{title}</h2>
        {meta ? <div className="flex flex-wrap items-center gap-2 text-[13px] text-ink-3">{meta}</div> : null}
      </div>
      {actions ? <div className="flex flex-wrap items-center gap-2">{actions}</div> : null}
    </div>
  );
}
