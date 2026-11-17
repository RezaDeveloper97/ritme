'use client';

import { AutopilotHost } from '@/shared/autopilot';

import { PILOT_CATALOG } from './catalog';

/**
 * The catalog carries functions (`reveal`), so the server layout can't pass it as a prop — this client
 * wrapper binds it to the host, the way `SheetHost` binds the sheet registry.
 */
export function AutopilotMount() {
  return <AutopilotHost catalog={PILOT_CATALOG} />;
}
