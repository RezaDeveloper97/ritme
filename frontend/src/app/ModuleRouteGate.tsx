'use client';

import { useTranslations } from 'next-intl';
import type { ReactNode } from 'react';

import { moduleOfRoute, useAppModulesQuery } from '@/entities/app-module';
import { useLifeStage } from '@/entities/user';
import { usePathname, useRouter } from '@/shared/i18n';
import { EmptyState, PrimaryButton } from '@/shared/ui';

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

function ModuleOff() {
  const t = useTranslations('common.moduleOff');
  const router = useRouter();
  const life = useLifeStage();
  // A companion account's home is the companion panel itself, so it gets no way «back» into the same card.
  const showHome = !life.data?.companion;
  return (
    <main className="module-off">
      <EmptyState
        icon="lock"
        title={t('title')}
        body={t('body')}
        action={showHome ? <PrimaryButton onClick={() => router.replace('/home')}>{t('home')}</PrimaryButton> : undefined}
      />
    </main>
  );
}
