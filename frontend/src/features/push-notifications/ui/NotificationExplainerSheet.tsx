'use client';

import { useTranslations } from 'next-intl';
import { useState } from 'react';

import { AppSheet } from '@/shared/sheet';
import { PrimaryButton, SecondaryButton } from '@/shared/ui';

import { enable } from '../model/channel';

/**
 * The in-context ask shown before the system prompt (never on page load).
 * «روشن کردن اعلان‌ها» runs the prompt, «فعلاً نه» just moves on; either way
 * `onDone` fires once so the caller can continue what the user was doing.
 */
export function NotificationExplainerSheet({ open, onDone }: { open: boolean; onDone: () => void }) {
  const t = useTranslations('pwa.push');
  const [pending, setPending] = useState(false);

  const turnOn = async () => {
    setPending(true);
    try {
      await enable();
    } catch {
      // The row in notification settings shows the real state and can retry.
    } finally {
      setPending(false);
      onDone();
    }
  };

  return (
    <AppSheet
      open={open}
      onClose={onDone}
      size="half"
      title={t('sheet.title')}
      footer={
        <div className="flex gap-2.5">
          <SecondaryButton block={false} className="flex-1" onClick={onDone}>
            {t('sheet.later')}
          </SecondaryButton>
          <PrimaryButton block={false} className="flex-1" onClick={() => void turnOn()} loading={pending}>
            {t('sheet.enable')}
          </PrimaryButton>
        </div>
      }
    >
      <p className="sub">{t('sheet.body')}</p>
    </AppSheet>
  );
}
