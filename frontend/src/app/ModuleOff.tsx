'use client';

import { useTranslations } from 'next-intl';

import { useLifeStage } from '@/entities/user';
import { useRouter } from '@/shared/i18n';
import { EmptyState, PrimaryButton } from '@/shared/ui';
import '@/shared/ui/styles/module.css';

export function ModuleOff() {
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
