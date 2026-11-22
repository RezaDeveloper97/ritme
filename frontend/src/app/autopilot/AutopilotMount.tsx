'use client';

import dynamic from 'next/dynamic';
import { useEffect, useState } from 'react';

/**
 * The catalog carries functions (`reveal`), so the server layout can't pass it as a prop — this client
 * wrapper binds it to the host, the way `SheetHost` binds the sheet registry.
 *
 * T-M12-28: the catalog (every screen's targets) and the engine are ~300 KB nobody needs until a script
 * starts, so the first-load bundle keeps only the tiny wake listener. `runAutopilot` / `useAutopilotCatalog`
 * call `requestAutopilotRuntime()`; that flips this component to load the chunk and mount the host.
 * (The tests still import `./catalog` statically.)
 */
const LazyHost = dynamic(
  async () => {
    const [{ AutopilotHost }, { PILOT_CATALOG }] = await Promise.all([import('@/shared/autopilot'), import('./catalog')]);
    return { default: () => <AutopilotHost catalog={PILOT_CATALOG} /> };
  },
  { ssr: false },
);

// Same string as `AUTOPILOT_WAKE_EVENT` in shared/autopilot/core/wake.ts (importing it would drag the engine into this chunk).
const WAKE_EVENT = 'ritme:autopilot-wake';

export function AutopilotMount() {
  const [wanted, setWanted] = useState(false);
  useEffect(() => {
    const wake = () => setWanted(true);
    if ((window as { __ritmeAutopilotWanted?: boolean }).__ritmeAutopilotWanted) wake();
    window.addEventListener(WAKE_EVENT, wake, { once: true });
    return () => window.removeEventListener(WAKE_EVENT, wake);
  }, []);
  return wanted ? <LazyHost /> : null;
}
