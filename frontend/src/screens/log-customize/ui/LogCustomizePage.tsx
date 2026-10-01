'use client';

import { CustomizeLog } from '@/features/customize-log';
import { categoryLook } from '@/features/log-day';
import { useRouter } from '@/shared/i18n';
import { SkyLayer } from '@/shared/ui';

/**
 * `/log/customize` (B-N3-04, nbl_/nbd_Log_Customize) — opened by the log sheet's gear. Back returns to
 * the log screen, which re-reads the saved preferences.
 */
export function LogCustomizePage() {
  const router = useRouter();
  return (
    <div className="view lcz-page">
      <SkyLayer />
      <div className="scroll lcz-scroll">
        <CustomizeLog lookOf={categoryLook} onExit={() => router.push('/log')} />
      </div>
    </div>
  );
}
