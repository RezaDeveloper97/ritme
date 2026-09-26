'use client';

import { useTranslations } from 'next-intl';

import { cn } from '@/shared/lib';
import { Icon } from '@/shared/ui';

import { checkupIcon } from '../lib/icon';
import { toneClass } from '../lib/tone';
import { useCheckupLabels } from './labels';

type Row = Parameters<ReturnType<typeof useCheckupLabels>['timing']>[0] & {
  icon: string | null;
  tone: string | null;
  performed_by: string;
};

/** The app's catalog row (icon tile + title + interval + timing), in light and dark. */
export function CheckupPreview({ row, title, subtitle }: { row: Row; title: string; subtitle: string }) {
  const t = useTranslations('checkupTypes');
  const { timing } = useCheckupLabels();
  return (
    <section className="flex flex-col gap-2" aria-label={t('preview')}>
      <span className="field-label">{t('preview')}</span>
      <p className="field-hint m-0">{t('previewHint')}</p>
      <div className="grid gap-3 md:grid-cols-2">
        {(['light', 'dark'] as const).map((theme) => (
          <div key={theme} data-theme={theme} className="rounded-xl border border-line bg-[var(--page)] p-3 text-[var(--ink)]">
            <span className="mb-2 block text-xs text-[var(--muted)]">{t(theme === 'light' ? 'previewLight' : 'previewDark')}</span>
            <div className="flex items-center gap-3 rounded-xl bg-[var(--surface)] p-3">
              <span className={cn('grid size-11 shrink-0 place-items-center rounded-xl', toneClass(row.tone))}>
                <Icon name={checkupIcon(row.icon, row.performed_by)} size={22} />
              </span>
              <span className="flex min-w-0 flex-col">
                <strong className="truncate text-[14px]">{title || t('previewUntitled')}</strong>
                {subtitle ? <span className="truncate text-xs text-[var(--ink-3)]">{subtitle}</span> : null}
                <span className="text-xs text-[var(--muted)]">{timing(row)}</span>
              </span>
            </div>
          </div>
        ))}
      </div>
    </section>
  );
}
