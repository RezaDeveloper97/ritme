'use client';

import dynamic from 'next/dynamic';
import type { ReactNode } from 'react';

import { moduleOfRoute, useAppModulesQuery } from '@/entities/app-module';
import { usePathname } from '@/shared/i18n';

// The «not available» card (and its CSS) loads only when a module is actually off (T-M12-27b).
const ModuleOff = dynamic(() => import('./ModuleOff').then((m) => m.ModuleOff), { ssr: false });

/**
 * Deep links into a module the admin switched off (GET /modules) show a «not available» card instead of the
 * screen, whose API would answer 403 module_disabled anyway. Links and tiles are hidden at their source; this
 * catches a bookmarked or typed URL. Renders the route until the switches are known (fail open).
 */
export function ModuleRouteGate({ children }: { children: ReactNode }) {
  const pathname = usePathname();
  const modules = useAppModulesQuery();
  const owner = moduleOfRoute(pathname);
  if (!owner || !modules.data || modules.data[owner]) return <>{children}</>;
  return <ModuleOff />;
}
