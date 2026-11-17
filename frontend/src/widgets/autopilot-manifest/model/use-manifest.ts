'use client';

import { useTranslations } from 'next-intl';
import { useMemo } from 'react';

import { useAppModules, type AppModule } from '@/entities/app-module';
import { useLogTaxonomy } from '@/entities/health-log';
import { logTargetsOf } from '@/features/log-day';
import { AUTOPILOT_CONFIG, useAutopilotCatalog } from '@/shared/autopilot';
import { buildAutopilotManifest, type AutopilotManifest } from '@/shared/pilot';

export interface AutopilotManifestState {
  /** `null` until the log taxonomy has loaded. */
  manifest: AutopilotManifest | null;
  loading: boolean;
}

/**
 * The agent manifest for this user, right now: the app's catalog plus the fields of their own log taxonomy
 * (mode-specific, in their language), minus the modules an admin switched off.
 *
 * This is what a surface sends to the model NEXT TO the user's request so the answer is a script the
 * engine can run. It holds vocabulary only — nothing the user logged (§11). The request that carries it
 * is an AI call and goes through `useAiGate` (§11.5); this hook itself calls no model.
 */
export function useAutopilotManifest(mode?: string): AutopilotManifestState {
  const t = useTranslations('autopilotCatalog');
  const catalog = useAutopilotCatalog();
  const modules = useAppModules();
  const taxonomy = useLogTaxonomy(mode);

  const manifest = useMemo(() => {
    if (!taxonomy.data) return null;
    return buildAutopilotManifest({
      catalog,
      dynamic: logTargetsOf(taxonomy.data),
      // Catalog titles are keys of this namespace, checked against the message files by catalog.test.ts.
      t: (key) => t(key as Parameters<typeof t>[0]),
      moduleEnabled: (module) => modules[module as AppModule] ?? true,
      blockedTargets: AUTOPILOT_CONFIG.safety.blockedTargets,
      blockedRoutes: AUTOPILOT_CONFIG.safety.blockedRoutes,
    });
  }, [catalog, modules, taxonomy.data, t]);

  return { manifest, loading: taxonomy.isLoading };
}
