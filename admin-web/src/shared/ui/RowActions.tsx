'use client';

import type { ReactNode } from 'react';

/**
 * The trailing cell of a table row. Clicks inside it never reach the row's own
 * onClick (which opens the record).
 */
export function RowActions({ children }: { children: ReactNode }) {
  return (
    <div className="row-actions" onClick={(e) => e.stopPropagation()}>
      {children}
    </div>
  );
}
