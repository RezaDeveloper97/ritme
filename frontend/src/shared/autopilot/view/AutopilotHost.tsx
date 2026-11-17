'use client';

import { useEffect, useRef } from 'react';

import { usePathname, useRouter } from '@/shared/i18n';

import { mountAutopilotRuntime } from '../dom/engine';
import type { PilotCatalog } from '@/shared/pilot';
import { AutopilotOverlay } from './AutopilotOverlay';

/**
 * Mounted once in the shell, inside `.app-shell`. Lends the engine what only a component can hold — the
 * locale-aware router and the live path — plus the catalog, which `app` owns (it knows every screen;
 * `shared` must not). Renders the overlay; nothing while idle.
 */
export function AutopilotHost({ catalog }: { catalog: PilotCatalog }) {
  const router = useRouter();
  const pathname = usePathname();
  const routerRef = useRef(router);
  const pathRef = useRef(pathname);
  routerRef.current = router;
  pathRef.current = pathname;

  useEffect(
    () =>
      mountAutopilotRuntime({
        push: (href) => routerRef.current.push(href),
        path: () => pathRef.current,
        catalog,
      }),
    [catalog],
  );

  return <AutopilotOverlay />;
}
