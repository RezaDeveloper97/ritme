'use client';

import { usePathname } from 'next/navigation';
import { useEffect, useRef } from 'react';

import {
  AppSheet,
  closeSheet,
  closeSheetOnRouteChange,
  syncSheetWithHistory,
  topOf,
  useSheetStore,
} from '@/shared/sheet';

import { sheetDefinition } from './registry';

/**
 * Renders whichever secondary screen the URL currently asks for, as a sheet
 * over the screen underneath. Mounted once, in the locale layout, so the sheet
 * outlives the screen it was opened from and the screen keeps its scroll
 * position and state while it is covered.
 *
 * Only the top of the stack is rendered — the sheets beneath it are hidden
 * behind it anyway, and keeping them mounted would let three copies of the
 * article query run at once.
 */
export function SheetHost() {
  const stack = useSheetStore((state) => state.stack);
  const pathname = usePathname();
  const firstPathname = useRef(pathname);

  // Adopt a sheet named in the URL on first load, then follow back/forward.
  useEffect(() => syncSheetWithHistory(), []);

  // A route change replaces the screen underneath, which would leave the sheet
  // hovering over something it has nothing to do with.
  useEffect(() => {
    if (pathname === firstPathname.current) return;
    firstPathname.current = pathname;
    closeSheetOnRouteChange();
  }, [pathname]);

  const target = topOf(stack);
  const definition = target ? sheetDefinition(target.id) : null;

  // An unrecognized `?sheet=` — a stale link, a renamed sheet — must not strand
  // the user behind an empty panel.
  useEffect(() => {
    if (target && !definition) closeSheet();
  }, [target, definition]);

  const Content = definition?.Component;
  const Title = definition?.Title;

  return (
    <AppSheet
      open={definition !== null}
      onClose={closeSheet}
      size={definition?.size ?? 'full'}
      title={Title ? <Title arg={target?.arg} /> : undefined}
    >
      {Content ? <Content arg={target?.arg} /> : null}
    </AppSheet>
  );
}
